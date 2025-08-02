package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// PromptForAuthToken prompts the user to enter an auth token
func PromptForAuthToken() (string, error) {
	fmt.Print("Enter your authentication token: ")
	
	reader := bufio.NewReader(os.Stdin)
	token, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	
	// Trim whitespace and newline characters
	token = strings.TrimSpace(token)
	
	if token == "" {
		return "", fmt.Errorf("auth token cannot be empty")
	}
	
	return token, nil
}

// GetOrPromptAuthToken gets auth token from config or prompts user if not found
func GetOrPromptAuthToken() (string, error) {
	// First try to get from config
	token, err := GetAuthToken()
	if err != nil {
		return "", fmt.Errorf("failed to load auth token from config: %w", err)
	}
	
	// If token exists in config, use it
	if token != "" {
		fmt.Printf("Using stored auth token: %s...\n", token[:min(len(token), 8)])
		return token, nil
	}
	
	// If no token in config, prompt user
	fmt.Println("No auth token found in config.")
	token, err = PromptForAuthToken()
	if err != nil {
		return "", err
	}
	
	// Save the token to config for future use
	if err := SetAuthToken(token); err != nil {
		fmt.Printf("Warning: Failed to save auth token to config: %v\n", err)
		// Continue anyway, we still have the token for this session
	} else {
		fmt.Println("Auth token saved to config for future use.")
	}
	
	return token, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}