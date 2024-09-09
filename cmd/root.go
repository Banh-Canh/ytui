/*
Copyright © 2024 Victor Hang
*/
package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/Banh-Canh/ytui/pkg/config"
)

var (
	versionFlag bool
	version     string
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
	configPath, err := config.GetConfigPath()
	if err != nil {
		log.Fatalf("Failed to get config path")
		os.Exit(1)
	}
	config.CreateDefaultConfigFile(configPath)
}

func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	RootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Display version information")
}
