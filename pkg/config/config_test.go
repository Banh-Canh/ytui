package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestCreateDefaultConfigFile(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	tmpFilePath := filepath.Join(tmpDir, "config.yaml")
	// Act
	CreateDefaultConfigFile(tmpFilePath)
	if _, err := os.Stat(tmpFilePath); os.IsNotExist(err) {
		t.Fatalf("config file was not created: %v", err)
	}
}

func TestGetConfigPath(t *testing.T) {
	// Act
	configDir, err := GetConfigDirPath()
	if err != nil {
		os.Exit(1)
	}
	path := filepath.Join(configDir, "config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(path) == 0 {
		t.Error("path should not be empty")
	}
}

func TestReadConfig(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	tmpFilePath := filepath.Join(tmpDir, "config.yaml")

	// Create the config file
	CreateDefaultConfigFile(tmpFilePath)

	// Act
	err := ReadConfig(tmpFilePath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	// Verify if Viper has the expected values
	if !viper.IsSet("youtube.clientID") {
		t.Error("youtube.clientID should be set")
	}
	if !viper.IsSet("channels.subscribed") {
		t.Error("channels.subscribed should be set")
	}

	expectedChannels := []string{"UCTt2AnK--mnRmICnf-CCcrw", "UCutXfzLC5wrV3SInT_tdY0w"}
	actualChannels := viper.GetStringSlice("channels.subscribed")
	if len(expectedChannels) != len(actualChannels) {
		t.Errorf("expected %v, got %v", expectedChannels, actualChannels)
		return
	}
	for i, expected := range expectedChannels {
		if actualChannels[i] != expected {
			t.Errorf("expected %v, got %v", expectedChannels, actualChannels)
			break
		}
	}
}
