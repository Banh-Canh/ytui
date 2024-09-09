/*
Copyright © 2024 Victor Hang
*/
package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/Banh-Canh/ytui/pkg/player"
	"github.com/Banh-Canh/ytui/pkg/youtube"
)

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search <text>",
	Short: "Search for videos on Youtube/Invidious using keywords",
	Long: `
Search for videos on Youtube/Invidious with keywords.
Running this command will start a FZF menu with the search result.
Press enter to run any of the videos.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			cmd.Help() //nolint:all
			os.Exit(0)
		}
		query := args[0]
		result, err := youtube.SearchVideos(query, false)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		if len(*result) == 0 {
			log.Printf("No videos found.")
			os.Exit(0)
		}
		selectedVideo := youtube.YoutubeResultMenu(*result)
		player.RunMPV("https://www.youtube.com/watch?v=" + selectedVideo.VideoID)
		youtube.FeedHistory(selectedVideo)
	},
}

func init() {
	queryCmd.AddCommand(searchCmd)
}
