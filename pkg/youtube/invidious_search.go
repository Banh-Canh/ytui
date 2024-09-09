package youtube

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"

	"github.com/spf13/viper"

	"github.com/Banh-Canh/ytui/pkg/config"
)

type YouTubeAPI struct {
	Client *http.Client
}

type VideoSnippet struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	ChannelTitle string `json:"channelTitle"`
	PublishedAt  string `json:"publishedAt"`
	VideoURL     string `json:"videoUrl"`
}

// SearchResultItem represents a single search result
type SearchResultItem struct {
	Type            string           `json:"type"`
	Title           string           `json:"title"`
	VideoID         string           `json:"videoId"`
	Author          string           `json:"author"`
	AuthorID        string           `json:"authorId"`
	AuthorURL       string           `json:"authorUrl"`
	VideoThumbnails []VideoThumbnail `json:"videoThumbnails"`
	Description     string           `json:"description"`
	ViewCount       int64            `json:"viewCount"`
	ViewCountText   string           `json:"viewCountText"`
	Published       int64            `json:"published"`
	PublishedText   string           `json:"publishedText"`
	LengthSeconds   int32            `json:"lengthSeconds"`
	ViewedDate      int64            `json:"vieweddate"`
}
type VideoThumbnail struct {
	Quality string `json:"quality"`
	URL     string `json:"url"`
	Width   int32  `json:"width"`
	Height  int32  `json:"height"`
}

// Struct to parse the JSON response from the Invidious API
type SearchChannelResult struct {
	Author    string `json:"author"`
	AuthorUrl string `json:"authorUrl"`
}

// YouTubeSearchResponse represents the response from a YouTube search
type YouTubeSearchResponse []SearchResultItem

func getInvidiousInstance() (string, error) {
	// Read config for instance
	filepath, err := config.GetConfigPath()
	if err != nil {
		return "", fmt.Errorf("failed to get config path: %v", err)
	}
	viper.SetConfigFile(filepath)
	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		return "", fmt.Errorf("failed to read config file: %v", err)
	}
	invidiousInstance := viper.GetString("invidious.instance")
	return invidiousInstance, nil
}

func SearchVideos(query string, subscription bool) (*[]SearchResultItem, error) {
	var baseURL string
	invidiousInstance, err := getInvidiousInstance()
	if err != nil {
		return nil, err
	}
	baseURL = fmt.Sprintf("https://%s/api/v1/search", invidiousInstance)

	if subscription {
		baseURL = fmt.Sprintf("https://%s/api/v1/channels/%s/videos", invidiousInstance, query)

		resp, err := http.Get(baseURL)
		if err != nil {
			return nil, fmt.Errorf("error fetching data from YouTube API: %v", err)
		}
		defer resp.Body.Close()

		// Log the response status code to ensure we're getting a response
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("received non-200 response: %d %s", resp.StatusCode, resp.Status)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading response body: %v", err)
		}

		// Temporary map to unmarshal the "videos" part of the response
		var temp map[string]json.RawMessage
		if err := json.Unmarshal(body, &temp); err != nil {
			return nil, fmt.Errorf("error parsing JSON: %v", err)
		}

		var searchResponse []SearchResultItem
		if err := json.Unmarshal(temp["videos"], &searchResponse); err != nil {
			return nil, fmt.Errorf("error parsing JSON: %v", err)
		}

		// Sort by Published date in descending order
		sort.Slice(searchResponse, func(i, j int) bool {
			return searchResponse[i].Published > searchResponse[j].Published
		})

		return &searchResponse, nil
	}

	var aggregatedResults []SearchResultItem
	// Loop through the first 5 pages
	for page := 1; page <= 5; page++ {
		params := url.Values{}
		params.Set("page", fmt.Sprintf("%d", page))
		params.Set("type", "video") // We want video results only
		params.Set("q", query)      // The query string to search for
		params.Set("region", "US")

		fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

		resp, err := http.Get(fullURL)
		if err != nil {
			return nil, fmt.Errorf("error fetching data from YouTube API: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("received non-200 response: %d %s", resp.StatusCode, resp.Status)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading response body: %v", err)
		}

		var searchResponse []SearchResultItem
		if err := json.Unmarshal(body, &searchResponse); err != nil {
			return nil, fmt.Errorf("error parsing JSON: %v", err)
		}

		// Append the current page results to the aggregated results
		aggregatedResults = append(aggregatedResults, searchResponse...)
	}

	return &aggregatedResults, nil
}

func SearchVideoInfo(videoID string) (SearchResultItem, error) {
	invidiousInstance, err := getInvidiousInstance()
	if err != nil {
		return SearchResultItem{}, err
	}

	baseURL := fmt.Sprintf("https://%s/api/v1/videos/%s", invidiousInstance, videoID)

	resp, err := http.Get(baseURL)
	if err != nil {
		return SearchResultItem{}, fmt.Errorf("error fetching data from YouTube API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return SearchResultItem{}, fmt.Errorf("received non-200 response: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SearchResultItem{}, fmt.Errorf("error reading response body: %v", err)
	}

	var videoInfo VideoSnippet
	if err := json.Unmarshal(body, &videoInfo); err != nil {
		return SearchResultItem{}, fmt.Errorf("error parsing JSON: %v", err)
	}

	// Fill out the SearchResultItem with its own fields
	searchResultItem := SearchResultItem{
		VideoID:       videoID,
		Title:         videoInfo.Title,
		Description:   videoInfo.Description,
		Author:        videoInfo.ChannelTitle,
		PublishedText: videoInfo.PublishedAt,
	}

	return searchResultItem, nil
}

func SearchAuthorInfo(channelId string) (SearchChannelResult, error) {
	invidiousInstance, err := getInvidiousInstance()
	if err != nil {
		return SearchChannelResult{}, err
	}

	baseURL := fmt.Sprintf("https://%s/api/v1/channels/%s", invidiousInstance, channelId)

	// Perform the GET request to the Invidious API
	resp, err := http.Get(baseURL)
	if err != nil {
		return SearchChannelResult{}, fmt.Errorf("error fetching data from Invidious API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return SearchChannelResult{}, fmt.Errorf("received non-200 response: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SearchChannelResult{}, fmt.Errorf("error reading response body: %v", err)
	}

	var channelResponse SearchChannelResult
	err = json.Unmarshal(body, &channelResponse)
	if err != nil {
		return SearchChannelResult{}, fmt.Errorf("error unmarshalling response JSON: %v", err)
	}

	// Return the author name
	return channelResponse, nil
}

// Function to fetch author information for a list of channel IDs
func GetAllChannelsInfo(channelIds []string) ([]SearchChannelResult, error) {
	var results []SearchChannelResult
	for _, channelId := range channelIds {
		result, err := SearchAuthorInfo(channelId)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch info for channel ID %s: %v", channelId, err)
		}
		results = append(results, result)
	}
	return results, nil
}
