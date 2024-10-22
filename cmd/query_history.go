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
		fmt.Println("Fetching video history...")

		// Get the config directory
		configDir, err := config.GetConfigDirPath()
		if err != nil {
			utils.Logger.Error("Failed to get config path.", zap.Error(err))
			fmt.Println("Error: Unable to retrieve configuration path.")
			os.Exit(1)
		}
		utils.Logger.Debug("Configuration directory obtained.", zap.String("config_dir", configDir))

		// Get the history file path
		historyFile := filepath.Join(configDir, "watched_history.json")
		utils.Logger.Debug("Reading watched history from file.", zap.String("history_file", historyFile))

		// Fetch the watched videos from the history file
		result, err := youtube.GetWatchedVideos(historyFile)
		if err != nil {
			utils.Logger.Error("Failed to read history from file.", zap.Error(err))
			fmt.Println("Error: Unable to read video history.")
			os.Exit(1)
		}

		// Handle the case of no videos found
		if len(result) == 0 {
			utils.Logger.Info("No videos found in history.")
			fmt.Println("No videos found in history.")
			os.Exit(0)
		}

		utils.Logger.Info("Videos found in history.", zap.Int("video_count", len(result)))
		fmt.Printf("Found %d videos in your history.\n", len(result))

		// Show the FZF menu to select a video
		selectedVideo, err := youtube.YoutubeResultMenu(result, viper.GetString("invidious.instance"), viper.GetString("invidious.proxy"))
		if err != nil {
			utils.Logger.Info("FZF menu closed.")
			fmt.Println("History search cancelled.")
			os.Exit(0)
		}

		// Construct the video URL for the selected video
		videoURL := "https://www.youtube.com/watch?v=" + selectedVideo.VideoID
		fmt.Printf("Selected video: %s\n", selectedVideo.VideoID)

		// Handle download or playback
		if downloadFlag {
			var downloadDirStr string
			if downloadDirFlag != "" {
				downloadDirStr = downloadDirFlag // Use the flag if set
			} else {
				downloadDirStr = viper.GetString("download_dir") // Use config value if flag is not set
			}
			utils.Logger.Info("Downloading selected video with yt-dlp.", zap.String("video_url", videoURL))
			downloadDir := downloadDirStr
			fmt.Println("Downloading selected video...")
			download.RunYTDLP(videoURL, downloadDir)
			fmt.Println("Download completed.")
		} else {
			utils.Logger.Info("Playing selected video in MPV.", zap.String("video_url", videoURL))
			fmt.Println("Playing selected video in MPV...")

			player.RunMPV(videoURL)

			// Add to watch history if history is enabled
			if viper.GetBool("history.enable") {
				historyFilePath := filepath.Join(configDir, "watched_history.json")
				youtube.FeedHistory(selectedVideo, historyFilePath)
				utils.Logger.Info("Video added to watch history.", zap.String("video_id", selectedVideo.VideoID))
				fmt.Println("Video added to watch history.")
			}
		}
	},
}

func init() {
	queryCmd.AddCommand(historyCmd)
}
