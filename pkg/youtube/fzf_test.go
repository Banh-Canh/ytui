package youtube

import (
	"sync"
	"testing"
)

// Mocking the SearchVideoInfo function for testing
type MockVideoInfo struct {
	// Add a map to store the mock results
	MockResults map[string]SearchResultItem
	MockErrors  map[string]error
}

func (m *MockVideoInfo) SearchVideoInfo(videoID string) (SearchResultItem, error) {
	result, ok := m.MockResults[videoID]
	if !ok {
		return SearchResultItem{}, m.MockErrors[videoID]
	}
	return result, nil
}

// Test cleanDescription
func TestCleanDescription(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"\n\nSome text\n\nAnother line\n", "Some text\nAnother line"},
		{"A long description that exceeds 1000 characters...\n", "A long description that exceeds 1000 characters..."},
	}

	for _, test := range tests {
		result := cleanDescription(test.input)
		if result != test.expected {
			t.Errorf("cleanDescription(%q) = %q; expected %q", test.input, result, test.expected)
		}
	}
}

// Test getVideoPreview
func TestGetVideoPreview(t *testing.T) {
	video := SearchResultItem{
		VideoID:       "videoID",
		Title:         "Test Video",
		Author:        "Test Author",
		PublishedText: "Today",
		ViewCountText: "12345",
	}

	descriptionCache := map[string]string{
		"videoID": "Test description",
	}
	cacheLock := &sync.RWMutex{}

	preview := getVideoPreview(video, descriptionCache, cacheLock)
	expected := "=========\n\nTitle: Test Video\n\n=========\n\nAuthor: Test Author\n\n=========\n\nPublished: Today\n\n=========\n\nViews: 12345\n\n=========\n\nURL: https://www.youtube.com/watch?v=videoID\n\n=========\n\nDescription: \n\nTest description"
	if preview != expected {
		t.Errorf("getVideoPreview() = %q; expected %q", preview, expected)
	}
}
