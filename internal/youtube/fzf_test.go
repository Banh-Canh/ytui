package youtube

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFetchDescriptionsInBackground(t *testing.T) {
	// Mock video data
	videos := []SearchResultItem{
		{VideoID: "123", Title: "Video 1", Description: "Description 1"},
		{VideoID: "456", Title: "Video 2", Description: "Description 2"},
	}

	descriptionCache := make(map[string]string)
	cacheLock := sync.RWMutex{}

	// Call the function
	fetchDescriptionsInBackground(videos, descriptionCache, &cacheLock, "mockInstance", "mockProxy")

	// Wait for the background process to finish
	time.Sleep(1 * time.Second)

	// Verify that the descriptions have been cached
	cacheLock.RLock()
	defer cacheLock.RUnlock()
	assert.Equal(t, "Description 1", descriptionCache["123"])
	assert.Equal(t, "Description 2", descriptionCache["456"])
}
