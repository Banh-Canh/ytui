package youtube

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"

	"go.uber.org/zap"
	"golang.org/x/net/proxy"

	"github.com/Banh-Canh/ytui/internal/utils"
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

func SearchVideos(query, invidiousInstance, proxyURLString string, subscription bool) (*[]SearchResultItem, error) {
	if subscription {
		return searchSubscriptionVideos(query, invidiousInstance, proxyURLString)
	} else {
		return searchVideos(query, invidiousInstance, proxyURLString)
	}
}

func searchSubscriptionVideos(query, invidiousInstance, proxyURLString string) (*[]SearchResultItem, error) {
	baseURL := fmt.Sprintf("%s/api/v1/channels/%s/videos", invidiousInstance, query)
	utils.Logger.Debug("Subscription URL for channel videos constructed.", zap.String("subscription_url", baseURL))

	resp, err := makeRequest(baseURL, proxyURLString)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return processSubscribedVideoResponse(resp, baseURL)
}

func searchVideos(query, invidiousInstance, proxyURLString string) (*[]SearchResultItem, error) {
	baseURL := fmt.Sprintf("%s/api/v1/search", invidiousInstance)
	utils.Logger.Debug("Base URL for search constructed.", zap.String("base_url", baseURL))

	var aggregatedResults []SearchResultItem
	// Loop through the first 5 pages
	for page := 1; page <= 5; page++ {
		params := url.Values{}
		params.Set("page", fmt.Sprintf("%d", page))
		params.Set("type", "video")
		params.Set("q", query)
		params.Set("region", "US")

		fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
		utils.Logger.Debug("Constructed search URL.", zap.String("search_url", fullURL))

		resp, err := makeRequest(fullURL, proxyURLString)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		searchResponse, err := processResponse(resp, fullURL)
		if err != nil {
			return nil, err
		}

		// Append the current page results to the aggregated results
		aggregatedResults = append(aggregatedResults, *searchResponse...)

		// Log the number of results retrieved for the current page
		utils.Logger.Debug(
			"Page results added to aggregated results.",
			zap.Int("page", page),
			zap.Int("result_count", len(*searchResponse)),
		)
	}

	// Log the total number of aggregated results
	utils.Logger.Info("Search completed with total results.", zap.Int("total_result_count", len(aggregatedResults)))

	return &aggregatedResults, nil
}

func makeRequest(fullURL, proxyURLString string) (*http.Response, error) {
	if proxyURLString != "" {
		return makeRequestWithProxy(fullURL, proxyURLString)
	} else {
		return http.Get(fullURL)
	}
}

func makeRequestWithProxy(fullURL, proxyURLString string) (*http.Response, error) {
	proxyURL, err := url.Parse(proxyURLString)
	utils.Logger.Info("A proxy is configured and will be used.", zap.String("proxy_url", proxyURLString))
	if err != nil {
		return nil, fmt.Errorf("error parsing proxy URL: %v", err)
	}

	var transport *http.Transport
	switch proxyURL.Scheme {
	case "socks5":
		dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("error creating proxy dialer: %v", err)
		}
		transport = &http.Transport{Dial: dialer.Dial}

	case "http", "https":
		transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}

	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", proxyURL.Scheme)
	}

	// Create a client with the transport
	client := &http.Client{Transport: transport}

	// Make the request
	resp, err := client.Get(fullURL)
	if err != nil {
		utils.Logger.Error("Error fetching data from Invidious API.", zap.String("url", fullURL), zap.Error(err))
		return nil, fmt.Errorf("error creating the request: %v", err)
	}
	return resp, nil
}

func processResponse(resp *http.Response, fullURL string) (*[]SearchResultItem, error) {
	// Log the response status code
	utils.Logger.Debug("Received response from Invidious API.", zap.String("url", fullURL), zap.Int("status_code", resp.StatusCode))
	if resp.StatusCode != http.StatusOK {
		utils.Logger.Error(
			"Received non-200 response from Invidious API.",
			zap.String("url", fullURL),
			zap.Int("status_code", resp.StatusCode),
			zap.String("status", resp.Status),
		)
		return nil, fmt.Errorf("received non-200 response: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.Logger.Error("Error reading response body from Invidious API.", zap.String("url", fullURL), zap.Error(err))
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var searchResponse []SearchResultItem
	if err := json.Unmarshal(body, &searchResponse); err != nil {
		utils.Logger.Error("Error parsing 'videos' JSON from Invidious API response.", zap.String("url", fullURL), zap.Error(err))
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	// Log the number of videos retrieved
	utils.Logger.Info("Successfully retrieved videos.", zap.Int("video_count", len(searchResponse)))

	// Sort by Published date in descending order
	sort.Slice(searchResponse, func(i, j int) bool {
		return searchResponse[i].Published > searchResponse[j].Published
	})

	return &searchResponse, nil
}

func processSubscribedVideoResponse(resp *http.Response, fullURL string) (*[]SearchResultItem, error) {
	// Log the response status code
	utils.Logger.Debug("Received response from Invidious API.", zap.String("url", fullURL), zap.Int("status_code", resp.StatusCode))
	if resp.StatusCode != http.StatusOK {
		utils.Logger.Error(
			"Received non-200 response from Invidious API.",
			zap.String("url", fullURL),
			zap.Int("status_code", resp.StatusCode),
			zap.String("status", resp.Status),
		)
		return nil, fmt.Errorf("received non-200 response: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.Logger.Error("Error reading response body from Invidious API.", zap.String("url", fullURL), zap.Error(err))
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var temp map[string]json.RawMessage
	if err := json.Unmarshal(body, &temp); err != nil {
		utils.Logger.Error("Error parsing JSON from Invidious API response.", zap.String("url", fullURL), zap.Error(err))
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	var searchResponse []SearchResultItem
	if err := json.Unmarshal(temp["videos"], &searchResponse); err != nil {
		utils.Logger.Error("Error parsing 'videos' JSON from Invidious API response.", zap.String("url", fullURL), zap.Error(err))
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	// Log the number of videos retrieved
	utils.Logger.Info("Successfully retrieved videos.", zap.Int("video_count", len(searchResponse)))

	// Sort by Published date in descending order
	sort.Slice(searchResponse, func(i, j int) bool {
		return searchResponse[i].Published > searchResponse[j].Published
	})

	return &searchResponse, nil
}

func SearchVideoInfo(videoID, invidiousInstance, proxyURLString string) (SearchResultItem, error) {
	utils.Logger.Debug("Invidious instance retrieved.", zap.String("invidious_instance", invidiousInstance))

	baseURL := fmt.Sprintf("%s/api/v1/videos/%s", invidiousInstance, videoID)
	utils.Logger.Debug("Constructed URL for video info.", zap.String("url", baseURL))

	resp, err := makeRequest(baseURL, proxyURLString)
	if err != nil {
		return SearchResultItem{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.Logger.Error("Error reading response body from Invidious API.", zap.String("url", baseURL), zap.Error(err))
		return SearchResultItem{}, fmt.Errorf("error reading response body: %v", err)
	}

	var videoInfo VideoSnippet
	if err := json.Unmarshal(body, &videoInfo); err != nil {
		utils.Logger.Error("Error parsing JSON response from Invidious API.", zap.String("url", baseURL), zap.Error(err))
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

	utils.Logger.Debug("Successfully retrieved video info.", zap.String("video_id", videoID), zap.String("title", searchResultItem.Title))
	return searchResultItem, nil
}

func SearchAuthorInfo(channelId, invidiousInstance, proxyURLString string) (SearchChannelResult, error) {
	utils.Logger.Debug("Invidious instance retrieved.", zap.String("invidious_instance", invidiousInstance))

	baseURL := fmt.Sprintf("%s/api/v1/channels/%s", invidiousInstance, channelId)
	utils.Logger.Debug("Constructed URL for channel info.", zap.String("url", baseURL))

	// Perform the GET request to the Invidious API
	resp, err := makeRequest(baseURL, proxyURLString)
	if err != nil {
		return SearchChannelResult{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.Logger.Error("Error reading response body from Invidious API.", zap.String("url", baseURL), zap.Error(err))
		return SearchChannelResult{}, fmt.Errorf("error reading response body: %v", err)
	}

	var channelResponse SearchChannelResult
	err = json.Unmarshal(body, &channelResponse)
	if err != nil {
		utils.Logger.Error("Error unmarshalling response JSON from Invidious API.", zap.String("url", baseURL), zap.Error(err))
		return SearchChannelResult{}, fmt.Errorf("error unmarshalling response JSON: %v", err)
	}

	utils.Logger.Debug(
		"Successfully retrieved channel info.",
		zap.String("channel_id", channelId),
		zap.String("channel_name", channelResponse.Author),
	)
	return channelResponse, nil
}

// Function to fetch author information for a list of channel IDs
func GetAllChannelsInfo(channelIds []string, invidiousInstance, proxyURLString string) ([]SearchChannelResult, error) {
	utils.Logger.Debug("Starting to fetch info for multiple channels.", zap.Int("channel_count", len(channelIds)))

	var results []SearchChannelResult
	for _, channelId := range channelIds {
		utils.Logger.Debug("Fetching info for channel.", zap.String("channel_id", channelId))
		result, err := SearchAuthorInfo(channelId, invidiousInstance, proxyURLString)
		if err != nil {
			utils.Logger.Error("Failed to fetch info for channel ID.", zap.String("channel_id", channelId), zap.Error(err))
			return nil, fmt.Errorf("failed to fetch info for channel ID %s: %v", channelId, err)
		}
		results = append(results, result)
		utils.Logger.Info(
			"Successfully fetched info for channel.",
			zap.String("channel_id", channelId),
			zap.String("channel_name", result.Author),
		)
	}

	utils.Logger.Info("Completed fetching info for all channels.", zap.Int("total_channel_count", len(results)))
	return results, nil
}
