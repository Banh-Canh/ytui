/*
Copyright © 2024 Victor Hang
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/Banh-Canh/ytui/pkg/config"
	"github.com/Banh-Canh/ytui/pkg/utils"
	"github.com/Banh-Canh/ytui/pkg/youtube"
)

const (
	YoutubeSubscriptionsURL = "https://www.googleapis.com/youtube/v3/subscriptions"
)

// subscribedCmd represents the subscribed command
var showSubscribedCmd = &cobra.Command{
	Use:   "subscribed",
	Short: "show subscribed channels",
	Long: `
show your subscribed channels.
If you set the configuration "local: false" in the configuration file, it will prompt you to google login,
to retrieve your user informations. You must also configure your OAuth2 client in Google Dev Console first.

It will also only pick from the 50 most relevants subscribed channels in your Youtube account.`,
	Run: func(cmd *cobra.Command, args []string) {
		utils.Logger.Info("Command 'subscribed' executed.")
		var channelList []string
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
			apiChan, err := youtube.NewYouTubeAPI(clientID, secretID)
			if err != nil {
				utils.Logger.Fatal("Failed to authenticate to YouTube API.", zap.Error(err))
				os.Exit(1)
			}
			yt := <-apiChan
			utils.Logger.Info("YouTube API authenticated successfully.")

			channelList, err = yt.GetSubscribedChannels(YoutubeSubscriptionsURL)
			if err != nil {
				utils.Logger.Fatal("Failed to retrieve channels list.", zap.Error(err))
				os.Exit(1)
			}
			utils.Logger.Info("Retrieved subscribed channels list.", zap.Int("channel_count", len(channelList)))
		} else {
			channelList = viper.GetStringSlice("channels.subscribed")
			utils.Logger.Info("Retrieved local subscribed channels.", zap.Int("channel_count", len(channelList)))
		}
		channels, err := youtube.GetAllChannelsInfo(channelList, viper.GetString("invidious.instance"), viper.GetString("invidious.proxy"))
		if err != nil {
			utils.Logger.Fatal("Failed to get all channels data.", zap.Error(err))
			os.Exit(1)
		}
		utils.Logger.Info("Retrieved all channels information.", zap.Int("channel_count", len(channels)))
		for _, channel := range channels {
			fmt.Printf("\n")
			fmt.Printf("Author: %s\n", channel.Author)
			fmt.Printf("Author URL: %s\n", channel.AuthorUrl)
			fmt.Println(strings.Repeat("-", 30))
		}
	},
}

func init() {
	showCmd.AddCommand(showSubscribedCmd)
}
