package youtube

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ktr0731/go-fuzzyfinder"
	"go.uber.org/zap"

	"github.com/Banh-Canh/ytui/internal/utils"
)

// Fetches video descriptions in the background and caches them.
func fetchDescriptionsInBackground(
	videoData []SearchResultItem,
	descriptionCache map[string]string,
	cacheLock *sync.RWMutex,
	invidiousInstance,
	proxyURLString string,
) {
	go func() {
		for {
			for _, video := range videoData {
				utils.Logger.Debug("Fetching video description...", zap.String("videoTitle", video.Title))
				videoID := video.VideoID
				cacheLock.RLock()
				_, found := descriptionCache[videoID]
				cacheLock.RUnlock()
				if found {
					utils.Logger.Debug("Video description found in cache, skipping...", zap.String("videoTitle", video.Title))
					continue
				}

				if video.Description != "" {
					utils.Logger.Debug("Video description already exists in query, using it...", zap.String("videoTitle", video.Title))
					cacheLock.Lock()
					descriptionCache[videoID] = cleanDescription(video.Description)
					cacheLock.Unlock()
					continue
				}

				// Fetch the video description with retries
				if err := fetchAndCacheDescription(video, descriptionCache, cacheLock, invidiousInstance, proxyURLString); err != nil {
					utils.Logger.Error("Failed to fetch description.", zap.Error(err))
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
func fetchAndCacheDescription(
	video SearchResultItem,
	descriptionCache map[string]string,
	cacheLock *sync.RWMutex,
	invidiousInstance,
	proxyURLString string,
) error {
	for {
		videoInfo, err := SearchVideoInfo(video.VideoID, invidiousInstance, proxyURLString)
		if err != nil {
			utils.Logger.Info("Fetching description failed. Retrying...", zap.String("videoTitle", video.Title), zap.Error(err))
		}
		description := cleanDescription(videoInfo.Description)
		// Store the cleaned description in the cache
		cacheLock.Lock()
		descriptionCache[video.VideoID] = description
		cacheLock.Unlock()
		if descriptionCache[video.VideoID] != "" || description == "" {
			break
		}
	}
	utils.Logger.Debug("Fetched description successfully.", zap.String("videoTitle", video.Title))
	return nil
}

func allDescriptionsFetched(videoData []SearchResultItem, descriptionCache map[string]string) bool {
	for _, video := range videoData {
		if _, found := descriptionCache[video.VideoID]; !found {
			return false
		}
	}
	utils.Logger.Info("Finished fetching all query's video descriptions.")
	return true
}

// Provides descriptions of the video.
func getVideoPreview(video SearchResultItem, descriptionCache map[string]string, cacheLock *sync.RWMutex) string {
	videoID := video.VideoID

	// Check if description is cached
	cacheLock.RLock()
	description, found := descriptionCache[videoID]
	cacheLock.RUnlock()
	duration := time.Duration(video.LengthSeconds) * time.Second

	if !found {
		// If not cached, show a "fetching" message
		return fmt.Sprintf(
			"=========\n\nTitle: %s\nAuthor: %s\nPublished: %s\nDuration: %s\nViews: %s\nURL: %s\n\n=========\n\nDescription: Loading...",
			video.Title,
			video.Author,
			video.PublishedText,
			duration.String(),
			video.ViewCountText,
			"https://www.youtube.com/watch?v="+videoID,
		)
	}

	// Show cached description
	return fmt.Sprintf(
		"=========\n\nTitle: %s\nAuthor: %s\nPublished: %s\nDuration: %s\nViews: %s\nURL: %s\n\n=========\n\nDescription: \n\n%s",
		video.Title,
		video.Author,
		video.PublishedText,
		duration.String(),
		video.ViewCountText,
		"https://www.youtube.com/watch?v="+videoID,
		description,
	)
}

// Handles the interactive menu for video selection. Powered by fzf-like
func YoutubeResultMenu(videoData []SearchResultItem, invidiousInstance, proxyURLString string) (SearchResultItem, error) {
	// Cache to store video descriptions
	descriptionCache := make(map[string]string)
	cacheLock := sync.RWMutex{} // For thread-safe cache access

	// Start background fetching of descriptions
	fetchDescriptionsInBackground(videoData, descriptionCache, &cacheLock, invidiousInstance, proxyURLString)

	utils.Logger.Info("Opening search menu.")
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
		utils.Logger.Info("Closing search menu.")
		return SearchResultItem{}, err
	}
	invertedIdx := len(videoData) - 1 - idx
	return videoData[invertedIdx], nil
}
