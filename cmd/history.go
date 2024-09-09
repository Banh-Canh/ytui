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

// historyCmd represents the history command
var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Search for videos from your history",
	Long: `
Search for videos from your history. Due to Youtube Data APIv3 not allowing to retrieve user history,
ytui will feed and store its own history in a json file in the configuration directory. Any video watched with ytui
will be stored in there.`,
	Run: func(cmd *cobra.Command, args []string) {
		result, err := youtube.GetWatchedVideos(youtube.GetHistoryFilePath())
		if err != nil {
			log.Fatalf("Error loading history: %v", err)
		}
		if len(result) == 0 {
			log.Printf("No videos found.")
			os.Exit(0)
		}
		selectedVideo := youtube.YoutubeResultMenu(result)
		player.RunMPV("https://www.youtube.com/watch?v=" + selectedVideo.VideoID)
		youtube.FeedHistory(selectedVideo)
	},
}

func init() {
	queryCmd.AddCommand(historyCmd)
}
