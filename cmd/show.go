/*
Copyright © 2024 Victor Hang
*/
package cmd

import (
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "show items through different patterns",
	Long: `
show through different patterns.

Run one of the available subcommands.`,
}

func init() {
	RootCmd.AddCommand(showCmd)
}
