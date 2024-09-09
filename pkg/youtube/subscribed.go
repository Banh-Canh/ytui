package youtube

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"

	"github.com/spf13/viper"

	"github.com/Banh-Canh/ytui/pkg/config"
)

type YouTubeSubscriptionsResponse struct {
	Items []struct {
		Snippet struct {
			ResourceId struct {
				ChannelId string `json:"channelId"`
			} `json:"resourceId"`
		} `json:"snippet"`
	} `json:"items"`
}

func (yt *YouTubeAPI) GetSubscribedChannels() ([]string, error) {
	baseURL := "https://www.googleapis.com/youtube/v3/subscriptions"

	params := url.Values{}
	params.Set("part", "snippet")
	params.Set("mine", "true") // Only get subscriptions for the authenticated user
	params.Set("maxResults", "50")

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	resp, err := yt.Client.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("error fetching subscriptions from YouTube API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var subscriptionsResponse YouTubeSubscriptionsResponse
	if err := json.Unmarshal(body, &subscriptionsResponse); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	var channelIds []string
	for _, item := range subscriptionsResponse.Items {
		channelIds = append(channelIds, item.Snippet.ResourceId.ChannelId)
	}

	return channelIds, nil
}

func GetLocalSubscribedChannels() ([]string, error) {
	filepath, err := config.GetConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %v", err)
	}

	viper.SetConfigFile(filepath)
	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// Retrieve the 'subscribed' channels from the 'channels' section
	subscribed := viper.GetStringSlice("channels.subscribed")

	if len(subscribed) == 0 {
		log.Printf("No videos found.")
		os.Exit(0)
	}

	return subscribed, nil
}

func (yt *YouTubeAPI) GetAllSubscribedChannelsVideos() (*[]SearchResultItem, error) {
	channelIds, err := yt.GetSubscribedChannels()
	if err != nil {
		return nil, err
	}

	var aggregatedResponse *[]SearchResultItem

	for _, channelId := range channelIds {
		videosResponse, err := SearchVideos(channelId, true)
		if err != nil {
			return nil, err
		}

		// If the aggregatedResponse is nil, initialize it with the first response
		if aggregatedResponse == nil {
			aggregatedResponse = videosResponse
		} else {
			// Aggregate the results by appending video items to the common list
			*aggregatedResponse = append(*aggregatedResponse, *videosResponse...)
		}
	}

	// Check if there are videos to sort
	if aggregatedResponse != nil && len(*aggregatedResponse) > 0 {
		// Sort the aggregatedResponse by Published in ascending order (oldest first)
		sort.Slice(*aggregatedResponse, func(i, j int) bool {
			return (*aggregatedResponse)[i].Published > (*aggregatedResponse)[j].Published
		})
	}

	return aggregatedResponse, nil
}

func GetLocalSubscribedChannelsVideos() (*[]SearchResultItem, error) {
	channelIds, err := GetLocalSubscribedChannels()
	if err != nil {
		return nil, err
	}

	var aggregatedResponse *[]SearchResultItem

	for _, channelId := range channelIds {
		videosResponse, err := SearchVideos(channelId, true)
		if err != nil {
			return nil, err
		}

		if aggregatedResponse == nil {
			aggregatedResponse = videosResponse
		} else {
			*aggregatedResponse = append(*aggregatedResponse, *videosResponse...)
		}
	}

	if aggregatedResponse != nil && len(*aggregatedResponse) > 0 {
		// Sort the aggregatedResponse by Published in ascending order (oldest first)
		sort.Slice(*aggregatedResponse, func(i, j int) bool {
			return (*aggregatedResponse)[i].Published > (*aggregatedResponse)[j].Published
		})
	}

	return aggregatedResponse, nil
}
