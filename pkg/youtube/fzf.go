package youtube

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/ktr0731/go-fuzzyfinder"
)

// Fetches video descriptions in the background and caches them.
func fetchDescriptionsInBackground(
	videoData []SearchResultItem,
	descriptionCache map[string]string,
	cacheLock *sync.RWMutex,
) {
	go func() {
		for {
			for _, video := range videoData {
				videoID := video.VideoID

				// Check if the description is already in the SearchResultItem
				cacheLock.RLock()
				_, found := descriptionCache[videoID]
				cacheLock.RUnlock()

				if found {
					// Skip fetching if the description is already in the cache
					continue
				}

				if video.Description != "" {
					// If the description is already filled in the SearchResultItem, use it
					cacheLock.Lock()
					descriptionCache[videoID] = cleanDescription(video.Description)
					cacheLock.Unlock()
					continue
				}

				// Fetch the video description with retries
				if err := fetchAndCacheDescription(videoID, descriptionCache, cacheLock); err != nil {
					log.Printf("Failed to fetch video %s description, retrying...", videoID)
					continue
				}
			}

			// Check if all descriptions are fetched
			if allDescriptionsFetched(videoData, descriptionCache) {
				break
			}
		}
	}()
}

func cleanDescription(description string) string {
	// Remove blank lines from the description
	lines := strings.Split(description, "\n")
	nonEmptyLines := []string{}
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine != "" {
			nonEmptyLines = append(nonEmptyLines, trimmedLine)
		}
	}
	description = strings.Join(nonEmptyLines, "\n")

	// Trim the description to 1000 characters
	if len(description) > 1000 {
		description = description[:1000] + "..."
	}
	return strings.TrimSpace(description)
}

// Fetches a single video description and stores it in cache.
func fetchAndCacheDescription(videoID string, descriptionCache map[string]string, cacheLock *sync.RWMutex) error {
	for {
		videoInfo, err := SearchVideoInfo(videoID)
		if err != nil {
			return err
		}
		description := cleanDescription(videoInfo.Description)

		// Store the cleaned description in the cache
		cacheLock.Lock()
		descriptionCache[videoID] = description
		cacheLock.Unlock()

		if descriptionCache[videoID] != "" {
			return nil
		}
	}
}

func allDescriptionsFetched(videoData []SearchResultItem, descriptionCache map[string]string) bool {
	for _, video := range videoData {
		if _, found := descriptionCache[video.VideoID]; !found {
			return false
		}
	}
	return true
}

// Provides descriptions of the video.
func getVideoPreview(video SearchResultItem, descriptionCache map[string]string, cacheLock *sync.RWMutex) string {
	videoID := video.VideoID

	// Check if description is cached
	cacheLock.RLock()
	description, found := descriptionCache[videoID]
	cacheLock.RUnlock()

	if !found {
		// If not cached, show a "fetching" message
		return fmt.Sprintf(
			"=========\n\nTitle: %s\n\n=========\n\nAuthor: %s\n\n=========\n\nPublished: %s\n\n=========\n\nViews: %s\n\n=========\n\nURL: %s\n\n=========\n\nDescription: Loading...",
			video.Title,
			video.Author,
			video.PublishedText,
			video.ViewCountText,
			"https://www.youtube.com/watch?v="+videoID,
		)
	}

	// Show cached description
	return fmt.Sprintf(
		"=========\n\nTitle: %s\n\n=========\n\nAuthor: %s\n\n=========\n\nPublished: %s\n\n=========\n\nViews: %s\n\n=========\n\nURL: %s\n\n=========\n\nDescription: \n\n%s",
		video.Title,
		video.Author,
		video.PublishedText,
		video.ViewCountText,
		"https://www.youtube.com/watch?v="+videoID,
		description,
	)
}

// Handles the interactive menu for video selection. Powered by fzf-like
func YoutubeResultMenu(videoData []SearchResultItem) SearchResultItem {
	// Cache to store video descriptions
	descriptionCache := make(map[string]string)
	cacheLock := sync.RWMutex{} // For thread-safe cache access

	// Start background fetching of descriptions
	fetchDescriptionsInBackground(videoData, descriptionCache, &cacheLock)

	// Invert the indexing order of videoData by adjusting the index in the callback
	idx, err := fuzzyfinder.Find(
		videoData,
		func(i int) string {
			// Invert the index by calculating it from the end of the slice
			// Since we put the cursor at the top, it's more intuitive
			invertedIndex := len(videoData) - 1 - i
			title := videoData[invertedIndex].Title
			if len(title) > 70 {
				return title[:70] + "..." // Trim the title
			}
			author := videoData[invertedIndex].Author

			return title + " - " + author
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			// Invert the index for the preview window as well
			invertedIndex := len(videoData) - 1 - i
			return getVideoPreview(videoData[invertedIndex], descriptionCache, &cacheLock)
		}),
		fuzzyfinder.WithCursorPosition(fuzzyfinder.CursorPositionTop))
	if err != nil {
		log.Fatal(err)
	}

	invertedIdx := len(videoData) - 1 - idx
	return videoData[invertedIdx]
}
