package client

import (
	"bytes"
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
    Body       []byte            `json:"body,omitempty"`
    StatusCode int               `json:"statusCode,omitempty"`
}

func NewTunnelClient(config *Config) *TunnelClient {
    logger := log.New(os.Stdout, "[TUNNEL] ", log.LstdFlags)
    if !config.Verbose {
        logger.SetOutput(io.Discard)
    }

    return &TunnelClient{
        config: config,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
        logger: logger,
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
    
    conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
    if err != nil {
        return err
    }

    tc.conn = conn
    tc.logger.Printf("Connected to tunnel server")
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

    // Wait for registration response
    _, message, err := tc.conn.ReadMessage()
    if err != nil {
        return fmt.Errorf("failed to read registration response: %w", err)
    }

    var response TunnelMessage
    if err := json.Unmarshal(message, &response); err != nil {
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
    for {
        _, message, err := tc.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                tc.logger.Printf("WebSocket error: %v", err)
            }
            break
        }

        var tunnelMsg TunnelMessage
        if err := json.Unmarshal(message, &tunnelMsg); err != nil {
            tc.logger.Printf("Failed to unmarshal message: %v", err)
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
            Body:       []byte("Internal server error"),
        }
    } else {
        response.Type = "response"
        response.RequestID = msg.RequestID
    }

    // Send response back to server
    if err := tc.sendMessage(*response); err != nil {
        tc.logger.Printf("Failed to send response: %v", err)
    }
}

func (tc *TunnelClient) forwardToLocal(msg TunnelMessage) (*TunnelMessage, error) {
    // Construct local URL
    localURL := fmt.Sprintf("http://localhost:%d%s", tc.config.LocalPort, msg.Path)
    
    // Create request
    req, err := http.NewRequest(msg.Method, localURL, bytes.NewReader(msg.Body))
    if err != nil {
        return nil, err
    }

    // Copy headers
    for key, value := range msg.Headers {
        // Skip host header as it should be localhost
        if strings.ToLower(key) != "host" {
            req.Header.Set(key, value)
        }
    }

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
            headers[key] = values[0]
        }
    }

    tc.logger.Printf("Local response: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))

    return &TunnelMessage{
        StatusCode: resp.StatusCode,
        Headers:    headers,
        Body:       body,
    }, nil
}

func (tc *TunnelClient) sendMessage(msg TunnelMessage) error {
    data, err := json.Marshal(msg)
    if err != nil {
        return err
    }

    return tc.conn.WriteMessage(websocket.TextMessage, data)
}

func (tc *TunnelClient) waitForInterrupt() {
    interrupt := make(chan os.Signal, 1)
    signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

    <-interrupt
    fmt.Printf("\n\n🛑 Shutting down tunnel...\n")

    // Close connection gracefully
    tc.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
    time.Sleep(time.Second)
}
