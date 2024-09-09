package main

import (
	"log"

	"github.com/spf13/cobra/doc"

	cmd "github.com/Banh-Canh/ytui/cmd"
)

func main() {
	rootCmd := cmd.RootCmd
	err := doc.GenMarkdownTree(rootCmd, "./docs")
	if err != nil {
		log.Fatal(err)
	}
}
