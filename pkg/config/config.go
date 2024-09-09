package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Channels []string `yaml:"channels"`
}

// Creates the YAML config file
func CreateDefaultConfigFile(filePath string) {
	// Struct with empty channels list

	viper.SetDefault("invidious", map[string]interface{}{
		"instance": "invidious.jing.rocks",
	})
	viper.SetDefault("youtube", map[string]interface{}{
		"clientID": "CREATE_IN_YOUTUBE_API_CONSOLE",
		"secretID": "CREATE_IN_YOUTUBE_API_CONSOLE",
	})
	viper.SetDefault("channels", map[string]interface{}{
		"local":      true,
		"subscribed": []string{"UCTt2AnK--mnRmICnf-CCcrw", "UCutXfzLC5wrV3SInT_tdY0w"},
	})
	viper.SetConfigType("yaml")
	viper.SafeWriteConfigAs(filePath) // nolint:all
}

func GetConfigPath() (string, error) {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get home directory: %v", err)
	}
	// Construct the file path to the YAML file
	filePath := filepath.Join(homeDir, ".config", "ytui", "config.yaml")
	return filePath, nil
}

func ReadConfig(filePath string) error {
	// Set up Viper to read from the config file
	viper.SetConfigFile(filePath)

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}
	return nil
}
