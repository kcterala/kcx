package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

type TunnelClient struct {
    config     *Config
    conn       *websocket.Conn
    httpClient *http.Client
    logger     *log.Logger
    writeQueue chan TunnelMessage // Channel for queuing writes
    ctx        context.Context
    cancel     context.CancelFunc
}

type TunnelMessage struct {
    Type       string            `json:"type"`
    Subdomain  string            `json:"subdomain,omitempty"`
    AuthToken  string            `json:"authToken,omitempty"`
    PublicURL  string            `json:"publicUrl,omitempty"`
    Error      string            `json:"error,omitempty"`
    RequestID  string            `json:"requestId,omitempty"`
    Method     string            `json:"method,omitempty"`
    Path       string            `json:"path,omitempty"`
    Headers    map[string]string `json:"headers,omitempty"`
    Body       string            `json:"body,omitempty"` // base64 encoded
    StatusCode int               `json:"statusCode,omitempty"`
}

func NewTunnelClient(config *Config) *TunnelClient {
    logger := log.New(os.Stdout, "[TUNNEL] ", log.LstdFlags)
    if !config.Verbose {
        logger.SetOutput(io.Discard)
    }

    ctx, cancel := context.WithCancel(context.Background())

    return &TunnelClient{
        config: config,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
        logger:     logger,
        writeQueue: make(chan TunnelMessage, 1000), // Buffer for 1000 messages
        ctx:        ctx,
        cancel:     cancel,
    }
}

func (tc *TunnelClient) Start() error {
    // Generate subdomain if not provided
    if tc.config.Subdomain == "" {
        tc.config.Subdomain = generateRandomSubdomain()
    }

    tc.logger.Printf("Starting tunnel client...")
    tc.logger.Printf("Local port: %d", tc.config.LocalPort)
    tc.logger.Printf("Subdomain: %s", tc.config.Subdomain)
    tc.logger.Printf("Server: %s", tc.config.ServerURL)

    // Connect to server
    if err := tc.connect(); err != nil {
        return fmt.Errorf("failed to connect to server: %w", err)
    }
    defer tc.conn.Close()

    // Start write worker
    go tc.writeWorker()

    // Register tunnel
    if err := tc.register(); err != nil {
        return fmt.Errorf("failed to register tunnel: %w", err)
    }

    // Handle messages
    go tc.handleMessages()

    // Wait for interrupt signal
    tc.waitForInterrupt()
    
    return nil
}

func (tc *TunnelClient) connect() error {
    u, err := url.Parse(tc.config.ServerURL)
    if err != nil {
        return err
    }

    tc.logger.Printf("Connecting to %s", u.String())
    
    // Configure dialer with larger message size limits
    dialer := websocket.DefaultDialer
    dialer.ReadBufferSize = 64 * 1024 * 1024  // 64MB
    dialer.WriteBufferSize = 64 * 1024 * 1024 // 64MB
    
    conn, _, err := dialer.Dial(u.String(), nil)
    if err != nil {
        return err
    }

    // Set read and write limits on the connection
    conn.SetReadLimit(64 * 1024 * 1024) // 64MB limit for incoming messages
    
    tc.conn = conn
    tc.logger.Printf("Connected to tunnel server")
    return nil
}

// writeWorker handles all WebSocket writes in a single goroutine to prevent concurrent access
func (tc *TunnelClient) writeWorker() {
    tc.logger.Printf("Starting write worker")
    defer tc.logger.Printf("Write worker stopped")
    
    for {
        select {
        case msg := <-tc.writeQueue:
            if err := tc.writeMessage(msg); err != nil {
                tc.logger.Printf("Failed to write message: %v", err)
                // Consider implementing retry logic or error handling here
            }
        case <-tc.ctx.Done():
            tc.logger.Printf("Write worker context cancelled")
            return
        }
    }
}

func (tc *TunnelClient) writeMessage(msg TunnelMessage) error {
    data, err := json.Marshal(msg)
    if err != nil {
        return fmt.Errorf("failed to marshal message: %w", err)
    }

    tc.logger.Printf("Writing WebSocket message of size: %d bytes, type: %s", len(data), msg.Type)
    
    // Log large message warning
    if len(data) > 10*1024*1024 { // 10MB
        tc.logger.Printf("WARNING: Writing very large message (%d bytes)", len(data))
    }

    // Set write deadline to prevent hanging
    tc.conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
    defer tc.conn.SetWriteDeadline(time.Time{}) // Remove deadline
    
    err = tc.conn.WriteMessage(websocket.TextMessage, data)
    if err != nil {
        return fmt.Errorf("failed to write WebSocket message: %w", err)
    }

    tc.logger.Printf("Message written successfully")
    return nil
}

func (tc *TunnelClient) register() error {
    registerMsg := TunnelMessage{
        Type:      "register",
        Subdomain: tc.config.Subdomain,
        AuthToken: tc.config.AuthToken,
    }

    if err := tc.sendMessage(registerMsg); err != nil {
        return fmt.Errorf("failed to send register message: %w", err)
    }

    // Wait for registration response with timeout
    tc.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
    defer tc.conn.SetReadDeadline(time.Time{}) // Remove deadline
    
    _, message, err := tc.conn.ReadMessage()
    if err != nil {
        return fmt.Errorf("failed to read registration response: %w", err)
    }

    tc.logger.Printf("Registration response size: %d bytes", len(message))

    var response TunnelMessage
    if err := json.Unmarshal(message, &response); err != nil {
        tc.logger.Printf("Failed to parse registration response. Raw message: %s", string(message))
        return fmt.Errorf("failed to parse registration response: %w", err)
    }

    if response.Type == "error" {
        return fmt.Errorf("registration failed: %s", response.Error)
    }

    if response.Type == "registered" {
        fmt.Printf("\n🚀 Tunnel active!\n")
        fmt.Printf("   Public URL: %s\n", response.PublicURL)
        fmt.Printf("   Local URL:  http://localhost:%d\n", tc.config.LocalPort)
        fmt.Printf("   Subdomain:  %s\n\n", tc.config.Subdomain)
        fmt.Printf("Press Ctrl+C to stop the tunnel\n\n")
        return nil
    }

    return fmt.Errorf("unexpected registration response: %s", response.Type)
}

func (tc *TunnelClient) handleMessages() {
    tc.logger.Printf("Starting message handler")
    defer tc.logger.Printf("Message handler stopped")
    
    for {
        select {
        case <-tc.ctx.Done():
            tc.logger.Printf("Message handler context cancelled")
            return
        default:
            // Remove any read deadline for ongoing message handling
            tc.conn.SetReadDeadline(time.Now().Add(60 * time.Second)) // 60 second read timeout
            
            msgType, message, err := tc.conn.ReadMessage()
            if err != nil {
                if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                    tc.logger.Printf("WebSocket error: %v", err)
                }
                return
            }

            tc.logger.Printf("Received WebSocket message - Type: %d, Size: %d bytes", msgType, len(message))
            
            // Log message preview for debugging
            if len(message) > 1000 {
                tc.logger.Printf("Message preview - First 500 chars: %s", string(message[:500]))
                tc.logger.Printf("Message preview - Last 500 chars: %s", string(message[len(message)-500:]))
            }

            var tunnelMsg TunnelMessage
            if err := json.Unmarshal(message, &tunnelMsg); err != nil {
                tc.logger.Printf("Failed to unmarshal message of size %d bytes: %v", len(message), err)
                
                // Check if message appears truncated
                messageStr := string(message)
                if !strings.HasSuffix(strings.TrimSpace(messageStr), "}") {
                    tc.logger.Printf("Message appears truncated - doesn't end with '}'")
                }
                
                continue
            }

            tc.logger.Printf("Received message type: %s", tunnelMsg.Type)

            switch tunnelMsg.Type {
            case "request":
                go tc.handleRequest(tunnelMsg)
            default:
                tc.logger.Printf("Unknown message type: %s", tunnelMsg.Type)
            }
        }
    }
}

func (tc *TunnelClient) handleRequest(msg TunnelMessage) {
    tc.logger.Printf("Handling request: %s %s", msg.Method, msg.Path)

    // Forward request to local service
    response, err := tc.forwardToLocal(msg)
    if err != nil {
        tc.logger.Printf("Error forwarding request: %v", err)
        response = &TunnelMessage{
            Type:       "response",
            RequestID:  msg.RequestID,
            StatusCode: 500,
            Headers:    map[string]string{"Content-Type": "text/plain"},
            Body:       base64.StdEncoding.EncodeToString([]byte("Internal server error")),
        }
    } else {
        response.Type = "response"
        response.RequestID = msg.RequestID
    }

    // Log response size before sending
    responseData, _ := json.Marshal(*response)
    tc.logger.Printf("Queuing response of size: %d bytes", len(responseData))

    // Send response back to server through write queue
    if err := tc.sendMessage(*response); err != nil {
        tc.logger.Printf("Failed to queue response: %v", err)
    }
}

func (tc *TunnelClient) forwardToLocal(msg TunnelMessage) (*TunnelMessage, error) {
    // Construct local URL - always use localhost
    var hostHeader string
    
    localURL := fmt.Sprintf("http://localhost:%d%s", tc.config.LocalPort, msg.Path)
    
    if tc.config.UseSubdomainLocalhost && tc.config.Subdomain != "" {
        hostHeader = fmt.Sprintf("%s.localhost:%d", tc.config.Subdomain, tc.config.LocalPort)
        tc.logger.Printf("Using localhost URL: %s with subdomain Host header: %s", localURL, hostHeader)
    } else {
        hostHeader = fmt.Sprintf("localhost:%d", tc.config.LocalPort)
        tc.logger.Printf("Using localhost URL: %s with standard Host header: %s", localURL, hostHeader)
    }
    
    // Decode body from base64 if present
    var bodyBytes []byte
    if msg.Body != "" {
        decoded, err := base64.StdEncoding.DecodeString(msg.Body)
        if err != nil {
            tc.logger.Printf("Failed to decode base64 body: %v", err)
            bodyBytes = []byte(msg.Body) // Fallback to treating as plain text
        } else {
            bodyBytes = decoded
        }
    }
    
    // Create request
    req, err := http.NewRequest(msg.Method, localURL, bytes.NewReader(bodyBytes))
    if err != nil {
        return nil, err
    }

    // Copy headers
    for key, value := range msg.Headers {
        // Skip host header as we'll set it explicitly below
        if strings.ToLower(key) != "host" {
            req.Header.Set(key, value)
        }
    }
    
    // Set the Host header appropriately
    req.Host = hostHeader

    // Set X-Forwarded-Host header with the same value
    req.Header.Set("X-Forwarded-Host", hostHeader)

    // Make request
    resp, err := tc.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Read response body
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    // Convert response headers
    headers := make(map[string]string)
    for key, values := range resp.Header {
        if len(values) > 0 {
            headers[key] = strings.Join(values, ", ")
        }
    }

    tc.logger.Printf("Local response: %d %s, Body size: %d bytes", 
                     resp.StatusCode, http.StatusText(resp.StatusCode), len(body))

    // Encode body as base64
    encodedBody := base64.StdEncoding.EncodeToString(body)
    tc.logger.Printf("Encoded body size: %d bytes", len(encodedBody))

    return &TunnelMessage{
        StatusCode: resp.StatusCode,
        Headers:    headers,
        Body:       encodedBody, // Now base64 encoded
    }, nil
}

func (tc *TunnelClient) sendMessage(msg TunnelMessage) error {
    select {
    case tc.writeQueue <- msg:
        tc.logger.Printf("Message queued for sending, type: %s", msg.Type)
        return nil
    case <-time.After(5 * time.Second):
        return fmt.Errorf("timeout queuing message for write")
    case <-tc.ctx.Done():
        return fmt.Errorf("client context cancelled")
    }
}

func (tc *TunnelClient) waitForInterrupt() {
    interrupt := make(chan os.Signal, 1)
    signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

    <-interrupt
    fmt.Printf("\n\n🛑 Shutting down tunnel...\n")

    // Cancel context to stop workers
    tc.cancel()

    // Send close message through the write queue
    closeMsg := TunnelMessage{Type: "close"}
    select {
    case tc.writeQueue <- closeMsg:
    case <-time.After(1 * time.Second):
        tc.logger.Printf("Timeout sending close message")
    }
    
    // Wait a bit for cleanup
    time.Sleep(time.Second)
    
    // Close WebSocket connection
    tc.conn.Close()
}