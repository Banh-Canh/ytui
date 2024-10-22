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
	Short: "Show subscribed channels",
	Long: `
Show your subscribed channels.
If you set the configuration "local: false" in the configuration file, it will prompt you to Google login
to retrieve your user information. You must also configure your OAuth2 client in Google Dev Console first.

It will also only pick from the 50 most relevant subscribed channels in your Youtube account.`,
	Run: func(cmd *cobra.Command, args []string) {
		utils.Logger.Info("Command 'subscribed' executed.")
		fmt.Println("Retrieving your subscribed channels...")

		var channelList []string
		var err error

		// Get the config directory
		configDir, err := config.GetConfigDirPath()
		if err != nil {
			utils.Logger.Error("Failed to get config path.", zap.Error(err))
			fmt.Println("Error: Unable to retrieve configuration path.")
			os.Exit(1)
		}
		utils.Logger.Debug("Config directory retrieved.", zap.String("config_dir", configDir))

		// Read the config file
		configPath := filepath.Join(configDir, "config.yaml")
		if err := config.ReadConfig(configPath); err != nil {
			utils.Logger.Error("Failed to read config.", zap.Error(err))
			fmt.Println("Error: Unable to read configuration.")
			os.Exit(1)
		}
		utils.Logger.Debug("Config file read successfully.", zap.String("config_file", configPath))

		// Retrieve YouTube API credentials
		clientID := viper.GetString("youtube.clientid")
		secretID := viper.GetString("youtube.secretid")
		utils.Logger.Debug("YouTube API credentials retrieved.", zap.String("client_id", clientID))

		// Check if local or remote API should be used
		if !viper.GetBool("channels.local") {
			utils.Logger.Info("Using YouTube API to fetch subscribed channels.")
			fmt.Println("Authenticating with YouTube API...")

			// Authenticate with YouTube API
			apiChan, err := youtube.NewYouTubeAPI(clientID, secretID)
			if err != nil {
				utils.Logger.Error("Failed to authenticate to YouTube API.", zap.Error(err))
				fmt.Println("Error: Authentication with YouTube API failed.")
				os.Exit(1)
			}
			yt := <-apiChan
			utils.Logger.Info("YouTube API authenticated successfully.")

			// Get the list of subscribed channels
			channelList, err = yt.GetSubscribedChannels(YoutubeSubscriptionsURL)
			if err != nil {
				utils.Logger.Error("Failed to retrieve channels list.", zap.Error(err))
				fmt.Println("Error: Unable to retrieve subscribed channels.")
				os.Exit(1)
			}
			utils.Logger.Info("Retrieved subscribed channels list.", zap.Int("channel_count", len(channelList)))
		} else {
			utils.Logger.Info("Using local configuration for subscribed channels.")
			channelList = viper.GetStringSlice("channels.subscribed")
			utils.Logger.Info("Retrieved local subscribed channels.", zap.Int("channel_count", len(channelList)))
		}

		// Get detailed information on channels
		fmt.Println("Fetching subscribed channels information...")
		channels, err := youtube.GetAllChannelsInfo(channelList, viper.GetString("invidious.instance"), viper.GetString("invidious.proxy"))
		if err != nil {
			utils.Logger.Error("Failed to get channels data.", zap.Error(err))
			fmt.Println("Error: Unable to retrieve channel information.")
			os.Exit(1)
		}
		utils.Logger.Info("Retrieved all channels information.", zap.Int("channel_count", len(channels)))

		// Display the list of channels
		for _, channel := range channels {
			fmt.Printf("\n")
			fmt.Printf("Author: %s\n", channel.Author)
			fmt.Printf("Author URL: %s\n", channel.AuthorUrl)
			fmt.Println(strings.Repeat("-", 30))
		}
		fmt.Println("Successfully retrieved and displayed your subscribed channels.")
	},
}

func init() {
	showCmd.AddCommand(showSubscribedCmd)
}
