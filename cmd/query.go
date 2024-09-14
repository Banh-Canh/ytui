/*
Copyright © 2024 Victor Hang
*/
package cmd

import (
	"github.com/spf13/cobra"
)

var downloadFlag bool

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Run queries for videos through different patterns",
	Long: `
Run queries for videos through different patterns.

Run one of the available subcommands.`,
}

func init() {
	RootCmd.AddCommand(queryCmd)
	queryCmd.PersistentFlags().BoolVarP(&downloadFlag, "download", "d", false, "Download the selected video instead of watching it")
}
