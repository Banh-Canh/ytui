package youtube

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"
)

func GetHistoryFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting user home directory: %v", err)
	}
	watched_history := filepath.Join(homeDir, ".config", "ytui", "watched_history.json")
	return watched_history
}

func SaveHistory(history *[]SearchResultItem, filename string) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func GetWatchedVideos(filename string) ([]SearchResultItem, error) {
	var history []SearchResultItem // history should not be a pointer

	// Check if the file exists
	data, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		// Create an empty history file if it doesn't exist
		emptyHistory := []SearchResultItem{}
		emptyHistoryData, err := json.Marshal(emptyHistory)
		if err != nil {
			return nil, err
		}
		err = os.WriteFile(filename, emptyHistoryData, 0644)
		if err != nil {
			return nil, err
		}
		return emptyHistory, nil // Return empty history
	}
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &history)
	if err != nil {
		return nil, err
	}

	// Invert the history index, for displaying most recent at the top.
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	return history, nil
}

func FeedHistory(selectedVideo SearchResultItem) {
	currentTime := time.Now().Unix()
	selectedVideo.ViewedDate = currentTime
	// Add the selected video to history
	historyFilePath := GetHistoryFilePath()
	history, err := GetWatchedVideos(historyFilePath)
	if err != nil {
		log.Fatalf("Error loading history: %v", err)
	}
	history = append(history, selectedVideo)
	// Save updated history
	err = SaveHistory(&history, historyFilePath)
	if err != nil {
		log.Fatalf("Error saving history: %v", err)
	}
}
