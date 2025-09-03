package youtube

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Setup a temporary directory for testing
func setupTempDir(t *testing.T) string {
	return t.TempDir()
}

func TestSaveHistory(t *testing.T) {
	tempDir := setupTempDir(t)
	historyFile := filepath.Join(tempDir, "watched_history.json")

	// Create test data
	history := []SearchResultItem{
		{
			Type:      "video",
			Title:     "Test Video",
			VideoID:   "abc123",
			Author:    "Test Author",
			AuthorID:  "author123",
			AuthorURL: "https://youtube.com/author123",
			VideoThumbnails: []VideoThumbnail{
				{Quality: "default", URL: "https://img.youtube.com/vi/abc123/default.jpg", Width: 120, Height: 90},
				{Quality: "medium", URL: "https://img.youtube.com/vi/abc123/mqdefault.jpg", Width: 320, Height: 180},
				{Quality: "high", URL: "https://img.youtube.com/vi/abc123/hqdefault.jpg", Width: 480, Height: 360},
			},
			Description:   "This is a test video",
			ViewCount:     1000,
			ViewCountText: "1K views",
			Published:     time.Now().Unix(),
			PublishedText: "1 day ago",
			LengthSeconds: 300,
			ViewedDate:    time.Now().Unix(),
		},
	}

	err := SaveHistory(&history, historyFile)
	assert.NoError(t, err)

	// Check that the file is created
	_, err = os.Stat(historyFile)
	assert.NoError(t, err)

	// Check that the file contents are correct
	data, err := os.ReadFile(historyFile)
	assert.NoError(t, err)

	var savedHistory []SearchResultItem
	err = json.Unmarshal(data, &savedHistory)
	assert.NoError(t, err)

	assert.Equal(t, history, savedHistory)
}

func TestGetWatchedVideos_FileNotExist(t *testing.T) {
	tempDir := setupTempDir(t)
	historyFile := filepath.Join(tempDir, "watched_history.json")

	history, err := GetWatchedVideos(historyFile)
	assert.NoError(t, err)

	// Since the file didn't exist, we should get an empty history
	assert.Equal(t, 0, len(history))

	// Check that the file was created
	_, err = os.Stat(historyFile)
	assert.NoError(t, err)
}

func TestGetWatchedVideos_FileExists(t *testing.T) {
	tempDir := setupTempDir(t)
	historyFile := filepath.Join(tempDir, "watched_history.json")

	// Create a file with initial history data
	initialHistory := []SearchResultItem{
		{
			Type:      "video",
			Title:     "Test Video 1",
			VideoID:   "abc123",
			Author:    "Author 1",
			AuthorID:  "author1",
			AuthorURL: "https://youtube.com/author1",
			VideoThumbnails: []VideoThumbnail{
				{Quality: "default", URL: "https://img.youtube.com/vi/abc123/default.jpg", Width: 120, Height: 90},
			},
			Description:   "Description 1",
			ViewCount:     500,
			ViewCountText: "500 views",
			Published:     time.Now().Unix(),
			PublishedText: "2 days ago",
			LengthSeconds: 200,
			ViewedDate:    time.Now().Unix(),
		},
		{
			Type:      "video",
			Title:     "Test Video 2",
			VideoID:   "def456",
			Author:    "Author 2",
			AuthorID:  "author2",
			AuthorURL: "https://youtube.com/author2",
			VideoThumbnails: []VideoThumbnail{
				{Quality: "default", URL: "https://img.youtube.com/vi/def456/default.jpg", Width: 120, Height: 90},
			},
			Description:   "Description 2",
			ViewCount:     1000,
			ViewCountText: "1K views",
			Published:     time.Now().Unix(),
			PublishedText: "3 days ago",
			LengthSeconds: 300,
			ViewedDate:    time.Now().Unix(),
		},
	}
	data, err := json.Marshal(initialHistory)
	assert.NoError(t, err)

	err = os.WriteFile(historyFile, data, 0o644)
	assert.NoError(t, err)

	// Get the watched videos from the file
	history, err := GetWatchedVideos(historyFile)
	assert.NoError(t, err)

	// Since the list is inverted, the most recent video should appear first
	assert.Equal(t, "Test Video 2", history[0].Title)
	assert.Equal(t, "Test Video 1", history[1].Title)
}

func TestFeedHistory(t *testing.T) {
	tempDir := setupTempDir(t)

	// Mock config directory path
	historyFile := filepath.Join(tempDir, "watched_history.json")

	// Initial video in history
	initialHistory := []SearchResultItem{
		{
			Type:      "video",
			Title:     "Old Video",
			VideoID:   "xyz789",
			Author:    "Old Author",
			AuthorID:  "old_author",
			AuthorURL: "https://youtube.com/old_author",
			VideoThumbnails: []VideoThumbnail{
				{Quality: "default", URL: "https://img.youtube.com/vi/xyz789/default.jpg", Width: 120, Height: 90},
			},
			Description:   "An old video description",
			ViewCount:     800,
			ViewCountText: "800 views",
			Published:     time.Now().Unix(),
			PublishedText: "5 days ago",
			LengthSeconds: 600,
			ViewedDate:    time.Now().Unix(),
		},
	}
	data, err := json.Marshal(initialHistory)
	assert.NoError(t, err)
	err = os.WriteFile(historyFile, data, 0o644)
	assert.NoError(t, err)

	// Mock a new video to be added to history
	newVideo := SearchResultItem{
		Type:      "video",
		Title:     "New Video",
		VideoID:   "new123",
		Author:    "New Author",
		AuthorID:  "new_author",
		AuthorURL: "https://youtube.com/new_author",
		VideoThumbnails: []VideoThumbnail{
			{Quality: "default", URL: "https://img.youtube.com/vi/new123/default.jpg", Width: 120, Height: 90},
		},
		Description:   "A brand new video description",
		ViewCount:     1500,
		ViewCountText: "1.5K views",
		Published:     time.Now().Unix(),
		PublishedText: "1 hour ago",
		LengthSeconds: 360,
	}

	// Feed history with the new video
	FeedHistory(newVideo, historyFile)

	// Verify that the video was added to the history file
	history, err := GetWatchedVideos(historyFile)
	assert.NoError(t, err)

	// The new video should be at the end of the list
	assert.Equal(t, "New Video", history[0].Title) // The most recent should be first after feed
	assert.Equal(t, "Old Video", history[1].Title) // The older video should follow
	assert.Equal(t, 2, len(history))               // Check that the length of history is now 2
}
