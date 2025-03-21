package youtube

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"

	"go.uber.org/zap"

	"github.com/Banh-Canh/ytui/internal/utils"
)

const YoutubeSubscriptionsURL = "https://www.googleapis.com/youtube/v3/subscriptions"

type YouTubeSubscriptionsResponse struct {
	Items []struct {
		Snippet struct {
			ResourceId struct {
				ChannelId string `json:"channelId"`
			} `json:"resourceId"`
		} `json:"snippet"`
	} `json:"items"`
}

func (yt *YouTubeAPI) GetSubscribedChannels(baseURL string) ([]string, error) {
	params := url.Values{}
	params.Set("part", "snippet")
	params.Set("mine", "true") // Only get subscriptions for the authenticated user
	params.Set("maxResults", "50")

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	utils.Logger.Debug("Constructed URL for fetching subscriptions.", zap.String("url", fullURL))

	resp, err := yt.Client.Get(fullURL)
	if err != nil {
		utils.Logger.Error("Error fetching subscriptions from YouTube API.", zap.String("url", fullURL), zap.Error(err))
		return nil, fmt.Errorf("error fetching subscriptions from YouTube API: %v", err)
	}
	defer resp.Body.Close()

	utils.Logger.Debug("Received response from YouTube API.", zap.String("url", fullURL), zap.Int("status_code", resp.StatusCode))
	if resp.StatusCode != http.StatusOK {
		utils.Logger.Error(
			"Received non-200 response from YouTube API.",
			zap.String("url", fullURL),
			zap.Int("status_code", resp.StatusCode),
			zap.String("status", resp.Status),
		)
		return nil, fmt.Errorf("received non-200 response: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.Logger.Error("Error reading response body from YouTube API.", zap.String("url", fullURL), zap.Error(err))
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var subscriptionsResponse YouTubeSubscriptionsResponse
	if err := json.Unmarshal(body, &subscriptionsResponse); err != nil {
		utils.Logger.Error("Error parsing JSON response from YouTube API.", zap.String("url", fullURL), zap.Error(err))
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	var channelIds []string
	for _, item := range subscriptionsResponse.Items {
		channelIds = append(channelIds, item.Snippet.ResourceId.ChannelId)
	}

	utils.Logger.Debug("Successfully retrieved channel IDs from subscriptions.", zap.Int("channel_count", len(channelIds)))
	return channelIds, nil
}

func (yt *YouTubeAPI) GetAllSubscribedChannelsVideos(invidiousInstance, proxyURLString string) (*[]SearchResultItem, error) {
	utils.Logger.Info("Starting to fetch videos for all subscribed channels.")

	channelIds, err := yt.GetSubscribedChannels(YoutubeSubscriptionsURL)
	if err != nil {
		utils.Logger.Error("Failed to get subscribed channels.", zap.Error(err))
		return nil, err
	}
	utils.Logger.Info("Retrieved subscribed channels.", zap.Int("channel_count", len(channelIds)))

	var aggregatedResponse *[]SearchResultItem

	for _, channelId := range channelIds {
		utils.Logger.Info("Fetching videos for channel.", zap.String("channel_id", channelId))
		videosResponse, err := SearchVideos(channelId, invidiousInstance, proxyURLString, true)
		if err != nil {
			utils.Logger.Error("Failed to fetch videos for channel.", zap.String("channel_id", channelId), zap.Error(err))
			return nil, err
		}
		// If the aggregatedResponse is nil, initialize it with the first response
		if aggregatedResponse == nil {
			aggregatedResponse = videosResponse
			utils.Logger.Info(
				"Initialized aggregated response with videos from channel.",
				zap.String("channel_id", channelId),
				zap.Int("video_count", len(*videosResponse)),
			)
		} else {
			// Aggregate the results by appending video items to the common list
			*aggregatedResponse = append(*aggregatedResponse, *videosResponse...)
			utils.Logger.Info("Appended videos to aggregated response.", zap.String("channel_id", channelId), zap.Int("appended_video_count", len(*videosResponse)))
		}
	}
	// Check if there are videos to sort
	if aggregatedResponse != nil && len(*aggregatedResponse) > 0 {
		// Sort the aggregatedResponse by Published in ascending order (oldest first)
		utils.Logger.Info("Sorting aggregated videos by published date.")
		sort.Slice(*aggregatedResponse, func(i, j int) bool {
			return (*aggregatedResponse)[i].Published > (*aggregatedResponse)[j].Published
		})
		utils.Logger.Info("Sorted aggregated videos.", zap.Int("sorted_video_count", len(*aggregatedResponse)))
	} else {
		utils.Logger.Info("No videos to sort.")
	}
	utils.Logger.Info("Completed fetching and aggregating videos for all subscribed channels.")
	return aggregatedResponse, nil
}

func GetLocalSubscribedChannelsVideos(invidiousInstance, proxyURLString string, subscribed []string) (*[]SearchResultItem, error) {
	utils.Logger.Info("Starting to fetch videos for local subscribed channels.")

	var aggregatedResponse *[]SearchResultItem

	for _, channelId := range subscribed {
		utils.Logger.Debug("Fetching videos for channel.", zap.String("channel_id", channelId))
		videosResponse, err := SearchVideos(channelId, invidiousInstance, proxyURLString, true)
		if err != nil {
			utils.Logger.Error("Failed to fetch videos for channel.", zap.String("channel_id", channelId), zap.Error(err))
			return nil, err
		}

		if aggregatedResponse == nil {
			aggregatedResponse = videosResponse
			utils.Logger.Debug(
				"Initialized aggregated response with videos from channel.",
				zap.String("channel_id", channelId),
				zap.Int("video_count", len(*videosResponse)),
			)
		} else {
			*aggregatedResponse = append(*aggregatedResponse, *videosResponse...)
			utils.Logger.Debug("Appended videos to aggregated response.", zap.String("channel_id", channelId), zap.Int("appended_video_count", len(*videosResponse)))
		}
	}

	if aggregatedResponse != nil && len(*aggregatedResponse) > 0 {
		// Sort the aggregatedResponse by Published in ascending order (oldest first)
		utils.Logger.Debug("Sorting aggregated videos by published date.")
		sort.Slice(*aggregatedResponse, func(i, j int) bool {
			return (*aggregatedResponse)[i].Published > (*aggregatedResponse)[j].Published
		})
		utils.Logger.Info("Sorted aggregated videos.", zap.Int("sorted_video_count", len(*aggregatedResponse)))
	} else {
		utils.Logger.Info("No videos to sort.")
	}

	utils.Logger.Info("Completed fetching and aggregating videos for local subscribed channels.")
	return aggregatedResponse, nil
}
