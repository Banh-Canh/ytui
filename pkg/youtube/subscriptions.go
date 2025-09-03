package youtube

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
)

const YoutubeSubscriptionsURL = "https://www.googleapis.com/youtube/v3/subscriptions"

// SubscriptionsService handles subscription-related operations
type SubscriptionsService struct {
	client *Client
}

// NewSubscriptionsService creates a new subscriptions service
func (c *Client) Subscriptions() *SubscriptionsService {
	return &SubscriptionsService{client: c}
}

// GetChannelIDs retrieves the channel IDs of all subscribed channels for the authenticated user
func (s *SubscriptionsService) GetChannelIDs() ([]string, error) {
	params := url.Values{}
	params.Set("part", "snippet")
	params.Set("mine", "true")
	params.Set("maxResults", "50")

	fullURL := fmt.Sprintf("%s?%s", YoutubeSubscriptionsURL, params.Encode())
	
	resp, err := s.client.httpClient.Get(fullURL)
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

	var subscriptionsResponse SubscriptionsResponse
	if err := json.Unmarshal(body, &subscriptionsResponse); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	var channelIDs []string
	for _, item := range subscriptionsResponse.Items {
		channelIDs = append(channelIDs, item.Snippet.ResourceID.ChannelID)
	}

	return channelIDs, nil
}

// GetAllVideos retrieves videos from all subscribed channels
func (s *SubscriptionsService) GetAllVideos() ([]SearchResultItem, error) {
	channelIDs, err := s.GetChannelIDs()
	if err != nil {
		return nil, err
	}

	return s.GetVideosFromChannels(channelIDs)
}

// GetVideosFromChannels retrieves videos from the specified channel IDs
func (s *SubscriptionsService) GetVideosFromChannels(channelIDs []string) ([]SearchResultItem, error) {
	var aggregatedResponse []SearchResultItem
	searchService := s.client.Search()

	for _, channelID := range channelIDs {
		options := SearchOptions{
			Query:        channelID,
			Subscription: true,
		}
		
		videosResponse, err := searchService.Videos(options)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch videos for channel %s: %v", channelID, err)
		}

		aggregatedResponse = append(aggregatedResponse, videosResponse...)
	}

	// Sort by Published date in descending order
	if len(aggregatedResponse) > 0 {
		sort.Slice(aggregatedResponse, func(i, j int) bool {
			return aggregatedResponse[i].Published > aggregatedResponse[j].Published
		})
	}

	return aggregatedResponse, nil
}