package youtube

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
)

// SearchService handles video search operations
type SearchService struct {
	client *Client
}

// NewSearchService creates a new search service
func (c *Client) Search() *SearchService {
	return &SearchService{client: c}
}

// Videos searches for videos using the provided options
func (s *SearchService) Videos(options SearchOptions) ([]SearchResultItem, error) {
	if options.Subscription {
		return s.searchSubscriptionVideos(options.Query)
	}
	return s.searchVideos(options)
}

// VideoInfo retrieves detailed information about a specific video
func (s *SearchService) VideoInfo(videoID string) (SearchResultItem, error) {
	baseURL := fmt.Sprintf("%s/api/v1/videos/%s", s.client.invidiousURL, videoID)
	
	resp, err := s.makeRequest(baseURL)
	if err != nil {
		return SearchResultItem{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SearchResultItem{}, fmt.Errorf("error reading response body: %v", err)
	}

	var videoInfo VideoSnippet
	if err := json.Unmarshal(body, &videoInfo); err != nil {
		return SearchResultItem{}, fmt.Errorf("error parsing JSON: %v", err)
	}

	return SearchResultItem{
		VideoID:       videoID,
		Title:         videoInfo.Title,
		Description:   videoInfo.Description,
		Author:        videoInfo.ChannelTitle,
		PublishedText: videoInfo.PublishedAt,
	}, nil
}

// ChannelInfo retrieves information about a specific channel
func (s *SearchService) ChannelInfo(channelID string) (ChannelInfo, error) {
	baseURL := fmt.Sprintf("%s/api/v1/channels/%s", s.client.invidiousURL, channelID)
	
	resp, err := s.makeRequest(baseURL)
	if err != nil {
		return ChannelInfo{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ChannelInfo{}, fmt.Errorf("error reading response body: %v", err)
	}

	var channelInfo ChannelInfo
	if err := json.Unmarshal(body, &channelInfo); err != nil {
		return ChannelInfo{}, fmt.Errorf("error parsing JSON: %v", err)
	}

	return channelInfo, nil
}

// MultipleChannelsInfo retrieves information for multiple channels
func (s *SearchService) MultipleChannelsInfo(channelIDs []string) ([]ChannelInfo, error) {
	var results []ChannelInfo
	for _, channelID := range channelIDs {
		result, err := s.ChannelInfo(channelID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch info for channel ID %s: %v", channelID, err)
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *SearchService) searchSubscriptionVideos(channelID string) ([]SearchResultItem, error) {
	baseURL := fmt.Sprintf("%s/api/v1/channels/%s/videos", s.client.invidiousURL, channelID)
	
	resp, err := s.makeRequest(baseURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return s.processSubscribedVideoResponse(resp)
}

func (s *SearchService) searchVideos(options SearchOptions) ([]SearchResultItem, error) {
	baseURL := fmt.Sprintf("%s/api/v1/search", s.client.invidiousURL)
	
	var aggregatedResults []SearchResultItem
	maxPages := options.MaxPages
	if maxPages == 0 {
		maxPages = 5
	}

	for page := 1; page <= maxPages; page++ {
		params := url.Values{}
		params.Set("page", fmt.Sprintf("%d", page))
		params.Set("type", options.Type)
		params.Set("q", options.Query)
		params.Set("region", options.Region)

		fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
		
		resp, err := s.makeRequest(fullURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		searchResponse, err := s.processResponse(resp)
		if err != nil {
			return nil, err
		}

		aggregatedResults = append(aggregatedResults, searchResponse...)
	}

	return aggregatedResults, nil
}

func (s *SearchService) makeRequest(fullURL string) (*http.Response, error) {
	if s.client.proxyURL != "" {
		return s.makeRequestWithProxy(fullURL)
	}
	return s.client.httpClient.Get(fullURL)
}

func (s *SearchService) makeRequestWithProxy(fullURL string) (*http.Response, error) {
	// Proxy implementation would go here
	// For now, just use regular client
	return s.client.httpClient.Get(fullURL)
}

func (s *SearchService) processResponse(resp *http.Response) ([]SearchResultItem, error) {
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

	// Sort by Published date in descending order
	sort.Slice(searchResponse, func(i, j int) bool {
		return searchResponse[i].Published > searchResponse[j].Published
	})

	return searchResponse, nil
}

func (s *SearchService) processSubscribedVideoResponse(resp *http.Response) ([]SearchResultItem, error) {
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var temp map[string]json.RawMessage
	if err := json.Unmarshal(body, &temp); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	var searchResponse []SearchResultItem
	if err := json.Unmarshal(temp["videos"], &searchResponse); err != nil {
		return nil, fmt.Errorf("error parsing videos JSON: %v", err)
	}

	// Sort by Published date in descending order
	sort.Slice(searchResponse, func(i, j int) bool {
		return searchResponse[i].Published > searchResponse[j].Published
	})

	return searchResponse, nil
}