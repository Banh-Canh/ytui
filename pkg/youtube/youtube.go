// Package youtube provides a comprehensive YouTube API library for Go applications.
// It offers functionality for searching videos, managing subscriptions, handling authentication,
// managing watch history, and providing interactive user interfaces.
package youtube

import (
	"fmt"
)

// YouTube represents the main YouTube API client with all services
type YouTube struct {
	client *Client
}

// New creates a new YouTube API instance with the provided configuration
func New(config Config) *YouTube {
	client := NewClient(config)
	return &YouTube{
		client: client,
	}
}

// Client returns the underlying client for advanced usage
func (yt *YouTube) Client() *Client {
	return yt.client
}

// Search returns the search service
func (yt *YouTube) Search() *SearchService {
	return yt.client.Search()
}

// Subscriptions returns the subscriptions service
func (yt *YouTube) Subscriptions() *SubscriptionsService {
	return yt.client.Subscriptions()
}

// Auth returns the authentication service
func (yt *YouTube) Auth() *AuthService {
	return yt.client.Auth()
}

// Quick access methods for common operations

// SearchVideos is a convenience method for searching videos
func (yt *YouTube) SearchVideos(query string, maxPages int) ([]SearchResultItem, error) {
	options := DefaultSearchOptions()
	options.Query = query
	if maxPages > 0 {
		options.MaxPages = maxPages
	}
	return yt.Search().Videos(options)
}

// GetVideoInfo is a convenience method for getting video information
func (yt *YouTube) GetVideoInfo(videoID string) (SearchResultItem, error) {
	return yt.Search().VideoInfo(videoID)
}

// GetChannelInfo is a convenience method for getting channel information
func (yt *YouTube) GetChannelInfo(channelID string) (ChannelInfo, error) {
	return yt.Search().ChannelInfo(channelID)
}

// GetSubscribedChannels is a convenience method for getting subscribed channel IDs
func (yt *YouTube) GetSubscribedChannels() ([]string, error) {
	return yt.Subscriptions().GetChannelIDs()
}

// GetSubscriptionVideos is a convenience method for getting videos from subscribed channels
func (yt *YouTube) GetSubscriptionVideos() ([]SearchResultItem, error) {
	return yt.Subscriptions().GetAllVideos()
}

// Authenticate is a convenience method for OAuth2 authentication
func (yt *YouTube) Authenticate() error {
	client, err := yt.Auth().Authenticate()
	if err != nil {
		return err
	}
	yt.client.SetHTTPClient(client)
	return nil
}

// Version returns the library version
func Version() string {
	return "1.0.0"
}

// Example demonstrates basic usage of the library
func Example() {
	// Initialize the YouTube client
	config := Config{
		InvidiousURL: "https://invidious.example.com",
		ClientID:     "your-client-id",
		ClientSecret: "your-client-secret",
		RedirectURL:  "http://localhost:8080/oauth2callback",
	}
	
	_ = New(config) // Create client (unused in example)
	
	// Example usage (this would be in actual application code):
	fmt.Println("YouTube API Library Example")
	fmt.Printf("Library Version: %s\n", Version())
	fmt.Println("To use this library:")
	fmt.Println("1. Create a Config with your settings")
	fmt.Println("2. Call youtube.New(config) to create a client")
	fmt.Println("3. Use the various services for search, subscriptions, etc.")
	fmt.Printf("Client initialized with Invidious URL: %s\n", config.InvidiousURL)
}