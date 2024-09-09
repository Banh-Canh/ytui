/*
Copyright © 2024 Victor Hang
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Banh-Canh/ytui/pkg/config"
	"github.com/Banh-Canh/ytui/pkg/youtube"
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
		var channelList []string
		var err error
		// Read the config file
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
				log.Fatalf("Failed to authenticate to Youtube API: %v", err)
				os.Exit(1)
			}
			yt := <-apiChan
			channelList, err = yt.GetSubscribedChannels()
			if err != nil {
				log.Fatalf("Failed to retrieve channels list: %v", err)
			}

		} else {
			channelList = viper.GetStringSlice("channels.subscribed")
		}
		channels, err := youtube.GetAllChannelsInfo(channelList)
		if err != nil {
			log.Fatalf("Failed to get all channels data: %v", err)
		}
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
