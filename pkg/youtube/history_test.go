package youtube

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// TestSaveHistory tests the SaveHistory function.
func TestSaveHistory(t *testing.T) {
	history := []SearchResultItem{
		{
			Type:            "video",
			Title:           "Test Video",
			VideoID:         "12345",
			Author:          "Test Author",
			AuthorID:        "author123",
			AuthorURL:       "http://testurl.com",
			VideoThumbnails: nil,
			Description:     "A test video description.",
			ViewCount:       1000,
			ViewCountText:   "1K",
			Published:       time.Now().Unix(),
			PublishedText:   "Just now",
			LengthSeconds:   120,
			ViewedDate:      time.Now().Unix(),
		},
	}

	tmpFile, err := os.CreateTemp("", "history*.json")
	if err != nil {
		t.Fatalf("Error creating temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	err = SaveHistory(&history, tmpFile.Name())
	if err != nil {
		t.Fatalf("Error saving history: %v", err)
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Error reading file: %v", err)
	}

	var savedHistory []SearchResultItem
	err = json.Unmarshal(data, &savedHistory)
	if err != nil {
		t.Fatalf("Error unmarshalling JSON: %v", err)
	}

	if len(savedHistory) != len(history) {
		t.Fatalf("Saved history length is incorrect. Got %d, want %d", len(savedHistory), len(history))
	}
}

// TestGetWatchedVideos tests the GetWatchedVideos function.
func TestGetWatchedVideos(t *testing.T) {
	history := []SearchResultItem{
		{
			Type:            "video",
			Title:           "Test Video",
			VideoID:         "12345",
			Author:          "Test Author",
			AuthorID:        "author123",
			AuthorURL:       "http://testurl.com",
			VideoThumbnails: nil,
			Description:     "A test video description.",
			ViewCount:       1000,
			ViewCountText:   "1K",
			Published:       time.Now().Unix(),
			PublishedText:   "Just now",
			LengthSeconds:   120,
			ViewedDate:      time.Now().Unix(),
		},
	}

	tmpFile, err := os.CreateTemp("", "history*.json")
	if err != nil {
		t.Fatalf("Error creating temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	err = SaveHistory(&history, tmpFile.Name())
	if err != nil {
		t.Fatalf("Error saving history: %v", err)
	}

	loadedHistory, err := GetWatchedVideos(tmpFile.Name())
	if err != nil {
		t.Fatalf("Error getting watched videos: %v", err)
	}

	if len(loadedHistory) != len(history) {
		t.Fatalf("Loaded history length is incorrect. Got %d, want %d", len(loadedHistory), len(history))
	}
}
