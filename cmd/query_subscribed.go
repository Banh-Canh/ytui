/*
Copyright © 2024 Victor Hang
*/
package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Banh-Canh/ytui/pkg/config"
	"github.com/Banh-Canh/ytui/pkg/player"
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
		// Create a new instance of YouTubeAPI
		// Read the config file
		var result *[]youtube.SearchResultItem
		configPath, err := config.GetConfigPath()
		if err != nil {
			log.Fatalf("Failed to get config path: %v", err)
			os.Exit(1)
		}
		if err := config.ReadConfig(configPath); err != nil {
			log.Fatalf("Failed to read config: %v", err)
		}
		clientID := viper.GetString("youtube.clientid")
		secretID := viper.GetString("youtube.secretid")

		if !viper.GetBool("channels.local") {
			apiChan, err := youtube.NewYouTubeAPI(clientID, secretID)
			if err != nil {
				log.Fatalf("Failed to authenticate to Youtube API")
				os.Exit(1)
			}
			yt := <-apiChan
			result, _ = yt.GetAllSubscribedChannelsVideos()
		} else {
			result, _ = youtube.GetLocalSubscribedChannelsVideos()
		}

		selectedVideo := youtube.YoutubeResultMenu(*result)
		player.RunMPV("https://www.youtube.com/watch?v=" + selectedVideo.VideoID)
		youtube.FeedHistory(selectedVideo)
	},
}

func init() {
	queryCmd.AddCommand(subscribedCmd)
}
