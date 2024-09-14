/*
Copyright © 2024 Victor Hang
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/Banh-Canh/ytui/pkg/download"
	"github.com/Banh-Canh/ytui/pkg/player"
	"github.com/Banh-Canh/ytui/pkg/utils"
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
			utils.Logger.Error("Invalid number of arguments provided for 'search' command.")
			os.Exit(0)
		}
		query := args[0]
		utils.Logger.Info("Search command initiated.", zap.String("query", query))

		result, err := youtube.SearchVideos(query, false)
		if err != nil {
			utils.Logger.Fatal("Error searching for videos.", zap.Error(err))
			os.Exit(1)
		}

		if len(*result) == 0 {
			utils.Logger.Info("No videos found for the query.", zap.String("query", query))
			os.Exit(0)
		}

		utils.Logger.Info("Videos found.", zap.Int("video_count", len(*result)))
		selectedVideo, err := youtube.YoutubeResultMenu(*result)
		if err != nil {
			utils.Logger.Info("FZF menu closed.")
			os.Exit(0)
		}
		videoURL := "https://www.youtube.com/watch?v=" + selectedVideo.VideoID
		if downloadFlag {
			utils.Logger.Info("Downloading selected video with yt-dlp.", zap.String("video_url", videoURL))
			download.RunYTDLP(videoURL)
		} else {
			utils.Logger.Info("Playing selected video in MPV.", zap.String("video_url", videoURL))
			player.RunMPV(videoURL)
			youtube.FeedHistory(selectedVideo)
			utils.Logger.Info("Video added to watch history.", zap.String("video_id", selectedVideo.VideoID))
		}
	},
}

func init() {
	queryCmd.AddCommand(searchCmd)
}
