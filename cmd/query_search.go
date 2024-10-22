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
			fmt.Println("Error: Please provide a single search query.")
			os.Exit(0)
		}

		query := args[0]
		utils.Logger.Info("Search command initiated.", zap.String("query", query))
		fmt.Printf("Searching for videos with query: %s\n", query)

		configDir, err := config.GetConfigDirPath()
		if err != nil {
			utils.Logger.Fatal("Failed to get config path.", zap.Error(err))
			fmt.Println("Error: Failed to retrieve configuration path.")
			os.Exit(1)
		}
		utils.Logger.Debug("Config directory retrieved.", zap.String("config_dir", configDir))

		configPath := filepath.Join(configDir, "config.yaml")
		if err := config.ReadConfig(configPath); err != nil {
			utils.Logger.Fatal("Failed to read config.", zap.Error(err))
			fmt.Println("Error: Failed to read configuration file.")
			os.Exit(1)
		}
		utils.Logger.Debug("Config file read successfully.", zap.String("config_file", configPath))

		// Search for videos
		result, err := youtube.SearchVideos(query, viper.GetString("invidious.instance"), viper.GetString("invidious.proxy"), false)
		if err != nil {
			utils.Logger.Fatal("Error searching for videos.", zap.Error(err))
			fmt.Println("Error: Failed to search for videos.")
			os.Exit(1)
		}

		if len(*result) == 0 {
			utils.Logger.Info("No videos found for the query.", zap.String("query", query))
			fmt.Printf("No videos found for query: %s\n", query)
			os.Exit(0)
		}

		utils.Logger.Info("Videos found.", zap.Int("video_count", len(*result)))
		fmt.Printf("Found %d videos for query: %s\n", len(*result), query)

		// Display search results in FZF menu
		selectedVideo, err := youtube.YoutubeResultMenu(*result, viper.GetString("invidious.instance"), viper.GetString("invidious.proxy"))
		if err != nil {
			utils.Logger.Info("FZF menu closed.")
			fmt.Println("Search cancelled.")
			os.Exit(0)
		}

		// Build the video URL
		videoURL := "https://www.youtube.com/watch?v=" + selectedVideo.VideoID
		fmt.Printf("Selected video: %s\n", selectedVideo.VideoID)

		// Handle video playback or download
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

			// Add to watch history if enabled
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
	queryCmd.AddCommand(searchCmd)
}
