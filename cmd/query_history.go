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
	"go.uber.org/zap"

	"github.com/Banh-Canh/ytui/pkg/config"
	"github.com/Banh-Canh/ytui/pkg/download"
	"github.com/Banh-Canh/ytui/pkg/player"
	"github.com/Banh-Canh/ytui/pkg/utils"
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
		utils.Logger.Info("History command initiated.")

		configDir, err := config.GetConfigDirPath()
		if err != nil {
			utils.Logger.Error("Failed to get config path.", zap.Error(err))
			os.Exit(1)
		}
		utils.Logger.Debug("Configuration directory obtained.", zap.String("config_dir", configDir))

		historyFile := filepath.Join(configDir, "watched_history.json")
		utils.Logger.Debug("Reading watched history from file.", zap.String("history_file", historyFile))

		result, err := youtube.GetWatchedVideos(historyFile)
		if err != nil {
			utils.Logger.Error("Failed to read history from file.", zap.Error(err))
			os.Exit(1)
		}
		if len(result) == 0 {
			utils.Logger.Info("No videos found in history.")
			fmt.Println("No videos found.")
			os.Exit(0)
		}
		utils.Logger.Info("Videos found in history.", zap.Int("video_count", len(result)))
		selectedVideo, err := youtube.YoutubeResultMenu(result, viper.GetString("invidious.proxy"))
		if err != nil {
			utils.Logger.Info("FZF menu closed.")
			os.Exit(0)
		}
		videoURL := "https://www.youtube.com/watch?v=" + selectedVideo.VideoID
		if downloadFlag {
			utils.Logger.Info("Downloading selected video with yt-dlp.", zap.String("video_url", videoURL))
			downloadDir := viper.GetString("download_dir")
			download.RunYTDLP(videoURL, downloadDir)
		} else {
			utils.Logger.Info("Playing selected video in MPV.", zap.String("video_url", videoURL))
			player.RunMPV(videoURL)
			if viper.GetBool("history.enable") {
				youtube.FeedHistory(selectedVideo)
				utils.Logger.Info("Video added to watch history.", zap.String("video_id", selectedVideo.VideoID))
			}
		}
	},
}

func init() {
	queryCmd.AddCommand(historyCmd)
}
