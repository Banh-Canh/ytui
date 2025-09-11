/*
Copyright © 2024 Victor Hang
*/
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/Banh-Canh/ytui/internal/ui"
)

var browseCmd = &cobra.Command{
	Use:   "browse",
	Short: "Browse YouTube content",
	Long: `
Browse YouTube content using an interactive TUI.

Navigate through search results, subscribed channels, and watch history.
Use arrow keys or hjkl to navigate, Enter to open, Space/p to play.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Start TUI
		ui.Menu()
	},
}

func init() {
	RootCmd.AddCommand(browseCmd)
}