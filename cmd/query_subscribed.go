package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/Banh-Canh/ytui/internal/config"
	"github.com/Banh-Canh/ytui/internal/download"
	"github.com/Banh-Canh/ytui/internal/history"
	"github.com/Banh-Canh/ytui/internal/player"
	"github.com/Banh-Canh/ytui/internal/ui"
	"github.com/Banh-Canh/ytui/internal/utils"
	"github.com/Banh-Canh/ytui/pkg/youtube"
)

// subscribedCmd represents the subscribed command
var subscribedCmd = &cobra.Command{
	Use:   "subscribed",
	Short: "Search for videos from your subscribed channels",
	Long: `
Search videos on Youtube from your subscribed channels.
If you set the configuration "local: false" in the configuration file, it will prompt you to google login,
to retrieve your user informations. You must also configure your OAuth2 client in Google Dev Console first.

It will also only pick from the 50 most relevants subscribed channels in your Youtube account.`,
	Run: func(cmd *cobra.Command, args []string) {
		utils.Logger.Info("Command 'subscribed' executed.")
		fmt.Println("Executing 'subscribed' command...") // Print a simple status message.

		var result []youtube.SearchResultItem
		var err error

		// Read the config file
		configDir, err := config.GetConfigDirPath()
		if err != nil {
			utils.Logger.Fatal("Failed to get config path.", zap.Error(err))
			fmt.Println("Error: Failed to get config path.")
			os.Exit(1)
		}
		utils.Logger.Debug("Config directory retrieved.", zap.String("config_dir", configDir))
		configPath := filepath.Join(configDir, "config.yaml")
		if err := config.ReadConfig(configPath); err != nil {
			utils.Logger.Fatal("Failed to read config.", zap.Error(err))
			fmt.Println("Error: Failed to read config.")
			os.Exit(1)
		}
		utils.Logger.Debug("Config file read successfully.", zap.String("config_file", configPath))

		clientID := viper.GetString("youtube.clientid")
		secretID := viper.GetString("youtube.secretid")
		utils.Logger.Debug("Retrieved YouTube API credentials.", zap.String("client_id", clientID))

		if !viper.GetBool("channels.local") {
			utils.Logger.Info("Local configuration is false. Starting YouTube API authentication.")
			fmt.Println("Authenticating with YouTube API...")

			// Initialize YouTube client with OAuth2
			config := youtube.Config{
				InvidiousURL: viper.GetString("invidious.instance"),
				ProxyURL:     viper.GetString("invidious.proxy"),
				ClientID:     clientID,
				ClientSecret: secretID,
				RedirectURL:  "http://localhost:8080/oauth2callback",
			}
			yt := youtube.New(config)
			
			// Authenticate
			err = yt.Authenticate()
			if err != nil {
				utils.Logger.Fatal("Failed to authenticate to YouTube API.", zap.Error(err))
				fmt.Println("Error: Failed to authenticate with YouTube API.")
				os.Exit(1)
			}
			utils.Logger.Info("YouTube API authenticated successfully.")
			fmt.Println("YouTube API authenticated successfully.")

			result, err = yt.GetSubscriptionVideos()
			if err != nil {
				utils.Logger.Fatal("Failed to get all subscribed channels videos.", zap.Error(err))
				fmt.Println("Error: Failed to retrieve subscribed channels videos.")
				os.Exit(1)
			}
		} else {
			utils.Logger.Info("Using local configuration for subscribed channels.")
			fmt.Println("Retrieving subscribed channels from local configuration...")

			// Initialize YouTube client for local subscriptions
			config := youtube.Config{
				InvidiousURL: viper.GetString("invidious.instance"),
				ProxyURL:     viper.GetString("invidious.proxy"),
			}
			yt := youtube.New(config)
			
			result, err = yt.Subscriptions().GetVideosFromChannels(viper.GetStringSlice("channels.subscribed"))
			if err != nil {
				utils.Logger.Fatal("Failed to get local subscribed channels videos.", zap.Error(err))
				fmt.Println("Error: Failed to retrieve local subscribed channels videos.")
				os.Exit(1)
			}
		}
		for {
			utils.Logger.Info("Retrieved videos from subscribed channels.", zap.Int("video_count", len(result)))
			fmt.Printf("Found %d videos from subscribed channels.\n", len(result))

			selectedVideo, err := ui.VideoSelectionMenu(result, viper.GetString("invidious.instance"), viper.GetString("invidious.proxy"))
			if err != nil {
				utils.Logger.Info("FZF menu closed.")
				fmt.Println("Video selection cancelled.")
				os.Exit(0)
			}
			utils.Logger.Info("Selected video for playback.", zap.String("video_id", selectedVideo.VideoID))
			fmt.Printf("Selected video: %s\n", selectedVideo.VideoID)

			videoURL := "https://www.youtube.com/watch?v=" + selectedVideo.VideoID
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

				if viper.GetBool("history.enable") {
					historyFilePath := filepath.Join(configDir, "watched_history.json")
					err := history.Add(selectedVideo, historyFilePath)
					if err != nil {
						utils.Logger.Error("Failed to add video to history.", zap.Error(err))
					} else {
						utils.Logger.Info("Video added to watch history.", zap.String("video_id", selectedVideo.VideoID))
						fmt.Println("Video added to watch history.")
					}
				}

			}
			if !keepOpenFlag {
				break
			}
		}
	},
}

func init() {
	queryCmd.AddCommand(subscribedCmd)
}
