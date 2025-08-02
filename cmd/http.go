/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/kcterala/kcx/internal/client"
	"github.com/kcterala/kcx/internal/config"
	"github.com/spf13/cobra"
)

var (
    serverURL    string
    localPort    int
    subdomain    string
    authToken    string
    verbose      bool
)

// httpCmd represents the http command
var httpCmd = &cobra.Command{
	Use:   "http [port]",
	Short: "Expose a local HTTP server to the internet",
	Run: func(cmd *cobra.Command, args []string) {
        // Determine auth token: use command line flag if provided, otherwise get from config or prompt
        var finalAuthToken string
        if authToken != "" && authToken != "default-auth-token-12345" {
            // User provided a token via command line, use it
            finalAuthToken = authToken
        } else {
            // No token provided via command line, get from config or prompt
            token, err := config.GetOrPromptAuthToken()
            if err != nil {
                fmt.Printf("Error getting auth token: %v\n", err)
                os.Exit(1)
            }
            finalAuthToken = token
        }

        clientConfig := &client.Config{
            ServerURL:   serverURL,
            LocalPort:   localPort,
            Subdomain:   subdomain,
            AuthToken:   finalAuthToken,
            Verbose:     verbose,
        }
        
        tunnelClient := client.NewTunnelClient(clientConfig)
        tunnelClient.Start()
    },
}

func init() {
	rootCmd.AddCommand(httpCmd)

	httpCmd.Flags().StringVarP(&serverURL, "server", "s", "ws://localhost:8080/tunnel", "Tunnel server WebSocket URL")
    httpCmd.Flags().IntVarP(&localPort, "port", "p", 3000, "Local port to tunnel")
    httpCmd.Flags().StringVarP(&subdomain, "subdomain", "d", "", "Custom subdomain (optional)")
    httpCmd.Flags().StringVarP(&authToken, "token", "t", "", "Authentication token (optional, will prompt if not provided and not in config)")
    httpCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
}
