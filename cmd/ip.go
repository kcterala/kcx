/*
Copyright © 2025 NAME HERE kcterala@gmail.com
*/
package cmd

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

var (
	baseUrl = "https://1.1.1.1/cdn-cgi/trace"
)

// ipCmd represents the ip command
var ipCmd = &cobra.Command{
	Use:   "ip",
	Short: "shows ip and copies to the clipboard",
	Long: `Same as short`,
	Run: func(cmd *cobra.Command, args []string) {
		printAndCopyIpAddress()
	},
}

func printAndCopyIpAddress() {
	ipAddress, err  := getIpAddress()
	if err != nil {
		fmt.Println("error fetching ip address.")
		return
	}

	if ipAddress == "" {
		fmt.Println("Couldn't find the ip address. Are you sure you are connected to network?")
		return
	}

	fmt.Println("ip address:", ipAddress)

	copyToClipboard(ipAddress)
}

func getIpAddress() (string, error) {

	req, err := http.NewRequest("GET", baseUrl, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	
	body, err  := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	responeString := string(body)
	lines := strings.Split(responeString, "\n")
	var ipAddress string
	for _, line := range lines {
		if strings.HasPrefix(line, "ip=") {
			ipAddress = strings.TrimPrefix(line, "ip=")
			return ipAddress, nil
		}
	}

	return "", nil
}

func copyToClipboard(ipAddress string) {
	err := clipboard.WriteAll(ipAddress)
	if err != nil {
		fmt.Println("Error copying to clipboard:", err)
		return
	}

	fmt.Println("IP address copied to clipboard!")
}

func init() {
	rootCmd.AddCommand(ipCmd)
}
