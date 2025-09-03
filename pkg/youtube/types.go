package youtube

// SearchResultItem represents a single video search result
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

// VideoThumbnail represents a video thumbnail
type VideoThumbnail struct {
	Quality string `json:"quality"`
	URL     string `json:"url"`
	Width   int32  `json:"width"`
	Height  int32  `json:"height"`
}

// VideoSnippet represents detailed video information
type VideoSnippet struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	ChannelTitle string `json:"channelTitle"`
	PublishedAt  string `json:"publishedAt"`
	VideoURL     string `json:"videoUrl"`
}

// ChannelInfo represents channel information
type ChannelInfo struct {
	Author    string `json:"author"`
	AuthorURL string `json:"authorUrl"`
}

// SubscriptionsResponse represents the YouTube API subscriptions response
type SubscriptionsResponse struct {
	Items []struct {
		Snippet struct {
			ResourceID struct {
				ChannelID string `json:"channelId"`
			} `json:"resourceId"`
		} `json:"snippet"`
	} `json:"items"`
}

// SearchOptions contains options for video search
type SearchOptions struct {
	Query        string
	MaxPages     int
	Region       string
	Type         string
	Subscription bool
}

// DefaultSearchOptions returns default search options
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		MaxPages: 5,
		Region:   "US",
		Type:     "video",
	}
}