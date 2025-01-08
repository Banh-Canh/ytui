package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/Banh-Canh/ytui/internal/utils"
)

func TestMain(m *testing.M) {
	// Mock zap.Logger to avoid breaking tests.
	utils.Logger = zap.NewNop()

	// Run the tests
	m.Run()
}

func TestCreateDefaultConfigFile(t *testing.T) {
	// Define a temporary file path for testing
	tempFilePath := filepath.Join(os.TempDir(), "test_config.yaml")
	defer os.Remove(tempFilePath) // Clean up the file after the test

	// Create the default config file
	CreateDefaultConfigFile(tempFilePath)

	// Check if the file was created
	_, err := os.Stat(tempFilePath)
	require.NoError(t, err, "Expected config file to be created")

	// Check if the config file can be read correctly
	viper.SetConfigFile(tempFilePath)
	err = viper.ReadInConfig()
	require.NoError(t, err, "Failed to read the config file")

	// Validate the defaults set in the config
	assert.Equal(t, "info", viper.GetString("logLevel"))
	assert.Equal(t, "https://invidious.jing.rocks", viper.GetString("invidious.instance"))
	assert.Equal(t, true, viper.GetBool("history.enable"))
	assert.Equal(t, "CREATE_IN_YOUTUBE_API_CONSOLE", viper.GetString("youtube.clientID"))
}

func TestReadConfig(t *testing.T) {
	// Create a temporary config file
	tempFilePath := filepath.Join(os.TempDir(), "test_read_config.yaml")
	defer os.Remove(tempFilePath)

	// Create default config
	CreateDefaultConfigFile(tempFilePath)

	// Read the config
	err := ReadConfig(tempFilePath)
	require.NoError(t, err, "Expected to read config without error")

	// Validate a value from the config
	assert.Equal(t, "info", viper.GetString("logLevel"))
}

func TestReadConfig_FileNotFound(t *testing.T) {
	// Attempt to read a non-existent config file
	nonExistentFilePath := filepath.Join(os.TempDir(), "non_existent_config.yaml")
	err := ReadConfig(nonExistentFilePath)

	// Expect an error because the file does not exist
	require.Error(t, err, "Expected an error when reading non-existent config file")
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestReadConfig_InvalidConfigFile(t *testing.T) {
	// Create a temporary invalid config file
	tempFilePath := filepath.Join(os.TempDir(), "invalid_config.yaml")
	defer os.Remove(tempFilePath)

	// Write invalid YAML content to the file
	err := os.WriteFile(tempFilePath, []byte("invalid_yaml: : :"), 0o644)
	require.NoError(t, err, "Failed to write invalid config file")

	// Attempt to read the invalid config file
	err = ReadConfig(tempFilePath)
	require.Error(t, err, "Expected an error when reading invalid config file")
	assert.Contains(t, err.Error(), "failed to read config file")
}
