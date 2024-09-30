package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/Banh-Canh/ytui/pkg/utils"
)

type Config struct {
	Channels []string `yaml:"channels"`
}

// Creates the YAML config file
func CreateDefaultConfigFile(filePath string) {
	// Struct with empty channels list
	// Get user's home directory
	downloadDir := xdg.UserDirs.Videos
	viper.SetDefault("download_dir", downloadDir)
	viper.SetDefault("logLevel", "info")
	viper.SetDefault("invidious", map[string]interface{}{
		"proxy":    "",
		"instance": "invidious.jing.rocks",
	})
	viper.SetDefault("history", map[string]interface{}{
		"enable": true,
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

func GetConfigDirPath() (string, error) {
	// Construct the directory path to the config directory
	configDirPath := filepath.Join(xdg.ConfigHome, "ytui")
	if err := os.MkdirAll(configDirPath, os.ModePerm); err != nil {
		panic(fmt.Sprintf("Failed to create config directory: %v", err))
	}
	return configDirPath, nil
}

func ReadConfig(filePath string) error {
	// Set up Viper to read from the config file
	viper.SetConfigFile(filePath)
	utils.Logger.Debug("Reading config file...", zap.String("filePath", filePath))
	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		utils.Logger.Error("Failed to read config file.", zap.Error(err))
		return fmt.Errorf("failed to read config file: %v", err)
	}
	return nil
}
