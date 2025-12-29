package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	configDir  = ".config/lazyl2m-tui"
	configFile = "config.json"
)

// GetConfigPath returns the full path to the config file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, configDir, configFile), nil
}

// LoadConfig loads configuration from file or returns default
func LoadConfig() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return NewDefaultConfig(), err
	}

	// If config file doesn't exist, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return NewDefaultConfig(), nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return NewDefaultConfig(), err
	}

	// Parse JSON
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return NewDefaultConfig(), err
	}

	return &config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	configDirPath := filepath.Dir(configPath)
	if err := os.MkdirAll(configDirPath, 0755); err != nil {
		return err
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(configPath, data, 0644)
}
