package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type AppConfig struct {
	AuthToken string `json:"authToken"`
}

const configFileName = ".kcx-config.json"

// getConfigPath returns the path to the config file in the user's home directory
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, configFileName), nil
}

// LoadConfig loads the configuration from the config file
func LoadConfig() (*AppConfig, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	// If config file doesn't exist, return empty config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &AppConfig{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveConfig saves the configuration to the config file
func SaveConfig(config *AppConfig) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetAuthToken retrieves the auth token from config, returns empty string if not found
func GetAuthToken() (string, error) {
	config, err := LoadConfig()
	if err != nil {
		return "", err
	}
	return config.AuthToken, nil
}

// SetAuthToken saves the auth token to config
func SetAuthToken(token string) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}
	
	config.AuthToken = token
	return SaveConfig(config)
}