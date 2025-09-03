# YouTube API Library

A focused Go library for interacting with YouTube data through Invidious APIs and official YouTube APIs.

## Features

- **Search**: Search for videos with flexible options
- **Subscriptions**: Manage and retrieve videos from subscribed channels
- **Authentication**: OAuth2 authentication with YouTube API
- **Proxy Support**: HTTP/SOCKS5 proxy support
- **Clean API**: Simple, focused interface for YouTube data access

## Installation

```bash
go get github.com/Banh-Canh/ytui/pkg/youtube
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/Banh-Canh/ytui/pkg/youtube"
)

func main() {
    // Initialize the YouTube client
    config := youtube.Config{
        InvidiousURL: "https://invidious.snopyta.org",
        ClientID:     "your-google-client-id",
        ClientSecret: "your-google-client-secret",
        RedirectURL:  "http://localhost:8080/oauth2callback",
    }
    
    yt := youtube.New(config)
    
    // Search for videos
    videos, err := yt.SearchVideos("golang tutorial", 3)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d videos\n", len(videos))
    for _, video := range videos {
        fmt.Printf("- %s by %s\n", video.Title, video.Author)
    }
}
```

## Services

### Search Service

```go
// Search for videos
searchService := yt.Search()
videos, err := searchService.Videos(youtube.SearchOptions{
    Query: "golang",
    MaxPages: 5,
    Region: "US",
})

// Get video information
videoInfo, err := searchService.VideoInfo("dQw4w9WgXcQ")

// Get channel information
channelInfo, err := searchService.ChannelInfo("UC_x5XG1OV2P6uZZ5FSM9Ttw")
```

### Subscriptions Service

```go
// Authenticate first for subscriptions
err := yt.Authenticate()
if err != nil {
    log.Fatal(err)
}

// Get subscribed channels
subscriptions := yt.Subscriptions()
channelIDs, err := subscriptions.GetChannelIDs()

// Get videos from subscribed channels
videos, err := subscriptions.GetAllVideos()

// Get videos from specific channels
videos, err := subscriptions.GetVideosFromChannels([]string{"channel1", "channel2"})
```

### Authentication Service

```go
auth := yt.Auth()

// Authenticate synchronously
client, err := auth.Authenticate()
yt.Client().SetHTTPClient(client)

// Authenticate asynchronously
clientChan, err := auth.AuthenticateAsync()
client := <-clientChan
```


## Configuration

The `Config` struct allows you to customize the client:

```go
config := youtube.Config{
    InvidiousURL: "https://your-invidious-instance.com", // Required for search
    ProxyURL:     "socks5://127.0.0.1:9050",            // Optional proxy
    ClientID:     "your-google-client-id",              // Required for auth
    ClientSecret: "your-google-client-secret",          // Required for auth
    RedirectURL:  "http://localhost:8080/oauth2callback", // OAuth callback
}
```

## Data Types

### SearchResultItem

```go
type SearchResultItem struct {
    VideoID       string `json:"videoId"`
    Title         string `json:"title"`
    Author        string `json:"author"`
    Description   string `json:"description"`
    ViewCount     int64  `json:"viewCount"`
    Published     int64  `json:"published"`
    LengthSeconds int32  `json:"lengthSeconds"`
    // ... more fields
}
```

### SearchOptions

```go
type SearchOptions struct {
    Query        string
    MaxPages     int    // Default: 5
    Region       string // Default: "US"
    Type         string // Default: "video"
    Subscription bool   // Search in subscriptions
}
```

## Error Handling

All methods return errors that should be checked:

```go
videos, err := yt.SearchVideos("query", 1)
if err != nil {
    log.Printf("Search failed: %v", err)
    return
}
```

## Advanced Usage

### Custom HTTP Client

```go
import "net/http"

customClient := &http.Client{
    Timeout: time.Second * 30,
}

yt.Client().SetHTTPClient(customClient)
```

### Direct Service Access

```go
// Access services directly
client := yt.Client()
searchService := client.Search()
authService := client.Auth()
```

## Development

### Building

```bash
go build ./...
```

### Testing

```bash
go test ./...
```

### Dependencies

- `golang.org/x/oauth2` - OAuth2 authentication
- `github.com/ktr0731/go-fuzzyfinder` - Interactive UI

## License

This library is part of the ytui project. See the main project for license information.

## Contributing

Contributions are welcome! Please ensure your code follows the existing patterns and includes appropriate error handling.