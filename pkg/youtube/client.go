package youtube

import (
	"context"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Client represents the YouTube API client with all necessary functionality
type Client struct {
	httpClient   *http.Client
	invidiousURL string
	proxyURL     string
	oauth2Config *oauth2.Config
}

// Config holds the configuration for the YouTube client
type Config struct {
	InvidiousURL string
	ProxyURL     string
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// NewClient creates a new YouTube API client with the provided configuration
func NewClient(config Config) *Client {
	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       []string{"https://www.googleapis.com/auth/youtube.readonly"},
		Endpoint:     google.Endpoint,
	}

	return &Client{
		httpClient:   &http.Client{},
		invidiousURL: config.InvidiousURL,
		proxyURL:     config.ProxyURL,
		oauth2Config: oauth2Config,
	}
}

// SetHTTPClient sets a custom HTTP client (useful for authentication)
func (c *Client) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

// SetOAuth2Token sets the OAuth2 token for authenticated requests
func (c *Client) SetOAuth2Token(token *oauth2.Token) {
	c.httpClient = c.oauth2Config.Client(context.Background(), token)
}

// GetHTTPClient returns the underlying HTTP client
func (c *Client) GetHTTPClient() *http.Client {
	return c.httpClient
}

// GetOAuth2Config returns the OAuth2 configuration
func (c *Client) GetOAuth2Config() *oauth2.Config {
	return c.oauth2Config
}