/*
Copyright © 2024 Victor Hang
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"

	"github.com/Banh-Canh/ytui/internal/config"
	"github.com/Banh-Canh/ytui/internal/utils"
)

var (
	versionFlag  bool
	version      string
	logLevelFlag string
)

// rootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "ytui",
	Short: "ytui TUI.",
	Long: `
ytui is a TUI tool that allows users to query videos on youtube and play them in their local player.
The configuration files is autogenerated on first run at *$HOME/.config/ytui/config.yaml*`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize configuration here
		initConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if versionFlag {
			fmt.Printf("%s", version)
		} else {
			cmd.Help() //nolint:all
		}
	},
}

func initConfig() {
	// Your configuration initialization logic
	configDir, err := config.GetConfigDirPath()
	if err != nil {
		os.Exit(1)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	viper.SetConfigFile(configPath)
	config.CreateDefaultConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "error, couldn't read config file: %v\n", err)
		os.Exit(1)
	}
	// Check if the logLevelFlag has been set, if not, fallback to config
	var logLevelStr string
	if logLevelFlag != "" {
		logLevelStr = logLevelFlag // Use the flag if set
	} else {
		logLevelStr = viper.GetString("loglevel") // Use config value if flag is not set
	}

	logLevel := zapcore.InfoLevel //nolint:all
	switch logLevelStr {
	case "debug":
		logLevel = zapcore.DebugLevel
	case "info":
		logLevel = zapcore.InfoLevel
	case "error":
		logLevel = zapcore.ErrorLevel
	default:
		fmt.Printf("Unknown log level %s, defaulting to info\n", logLevelStr)
		logLevel = zapcore.InfoLevel
	}
	utils.InitializeLogger(logLevel, filepath.Join(configDir, "ytui.log"))
	utils.Logger.Info("Initialized configuration.")
}

func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	RootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Display version information")
	RootCmd.PersistentFlags().StringVarP(&logLevelFlag, "log-level", "l", "", "Override log level (debug, info, error)")
}
