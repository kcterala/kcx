/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strconv"

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
		if len(args) < 1 {
            fmt.Println("Error: port is required. Usage: cli http <port>")
            os.Exit(1)
        }

        port, err := strconv.Atoi(args[0])
        if err != nil {
            fmt.Printf("Invalid port: %v\n", args[0])
            os.Exit(1)
        }
        localPort = port
    
		// No token provided via command line, get from config or prompt
		token, err := config.GetOrPromptAuthToken()
		if err != nil {
			fmt.Printf("Error getting auth token: %v\n", err)
			os.Exit(1)
		}

        clientConfig := &client.Config{
            ServerURL:   serverURL,
            LocalPort:   localPort,
            Subdomain:   subdomain,
            AuthToken:   token,
            Verbose:     verbose,
        }
        
        tunnelClient := client.NewTunnelClient(clientConfig)
        tunnelClient.Start()
    },
}

func init() {
	rootCmd.AddCommand(httpCmd)

	httpCmd.Flags().StringVarP(&serverURL, "server", "s", "ws://localhost:8080/tunnel", "Tunnel server WebSocket URL")
    httpCmd.Flags().StringVarP(&subdomain, "subdomain", "d", "", "Custom subdomain (optional)")
    httpCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
}
