/*
Copyright © 2024 Victor Hang
*/
package cmd

import (
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
		var result *[]youtube.SearchResultItem
		var err error
		// Read the config file
		configDir, err := config.GetConfigDirPath()
		if err != nil {
			utils.Logger.Fatal("Failed to get config path.", zap.Error(err))
			os.Exit(1)
		}
		utils.Logger.Debug("Config directory retrieved.", zap.String("config_dir", configDir))
		configPath := filepath.Join(configDir, "config.yaml")
		if err := config.ReadConfig(configPath); err != nil {
			utils.Logger.Fatal("Failed to read config.", zap.Error(err))
		}
		utils.Logger.Debug("Config file read successfully.", zap.String("config_file", configPath))
		clientID := viper.GetString("youtube.clientid")
		secretID := viper.GetString("youtube.secretid")
		utils.Logger.Debug("Retrieved YouTube API credentials.", zap.String("client_id", clientID))
		if !viper.GetBool("channels.local") {
			utils.Logger.Info("Local configuration is false. Starting YouTube API authentication.")
			apiChan, err := youtube.NewYouTubeAPI(clientID, secretID)
			if err != nil {
				utils.Logger.Fatal("Failed to authenticate to YouTube API.", zap.Error(err))
				os.Exit(1)
			}
			yt := <-apiChan
			utils.Logger.Info("YouTube API authenticated successfully.")
			result, err = yt.GetAllSubscribedChannelsVideos()
			if err != nil {
				utils.Logger.Fatal("Failed to get all subscribed channels videos.", zap.Error(err))
				os.Exit(1)
			}
		} else {
			utils.Logger.Info("Using local configuration for subscribed channels.")
			result, err = youtube.GetLocalSubscribedChannelsVideos()
			if err != nil {
				utils.Logger.Fatal("Failed to get local subscribed channels videos.", zap.Error(err))
				os.Exit(1)
			}
		}
		utils.Logger.Info("Retrieved videos from subscribed channels.", zap.Int("video_count", len(*result)))
		selectedVideo, err := youtube.YoutubeResultMenu(*result)
		if err != nil {
			utils.Logger.Info("FZF menu closed.")
			os.Exit(0)
		}
		utils.Logger.Info("Selected video for playback.", zap.String("video_id", selectedVideo.VideoID))
		videoURL := "https://www.youtube.com/watch?v=" + selectedVideo.VideoID
		if downloadFlag {
			utils.Logger.Info("Downloading selected video with yt-dlp.", zap.String("video_url", videoURL))
			downloadDir := viper.GetString("download_dir")
			download.RunYTDLP(videoURL, downloadDir)
		} else {
			utils.Logger.Info("Playing selected video in MPV.", zap.String("video_url", videoURL))
			player.RunMPV(videoURL)
			youtube.FeedHistory(selectedVideo)
			utils.Logger.Info("Video added to watch history.", zap.String("video_id", selectedVideo.VideoID))
		}
	},
}

func init() {
	queryCmd.AddCommand(subscribedCmd)
}
