package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"github.com/Banh-Canh/ytui/internal/utils"
	"github.com/Banh-Canh/ytui/pkg/youtube"
)

// Save saves the history to the specified file
func Save(history []youtube.SearchResultItem, filename string) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		utils.Logger.Error("Failed to create directory for history file.", zap.String("directory", dir), zap.Error(err))
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		utils.Logger.Error("Failed to create history file.", zap.String("filename", filename), zap.Error(err))
		return err
	}
	defer file.Close()

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		utils.Logger.Error("Failed to marshal history data.", zap.String("filename", filename), zap.Error(err))
		return err
	}

	err = os.WriteFile(filename, data, 0o644)
	if err != nil {
		utils.Logger.Error("Failed to write history data to file.", zap.String("filename", filename), zap.Error(err))
		return err
	}

	utils.Logger.Info("Successfully saved history data.", zap.String("filename", filename), zap.Int("item_count", len(history)))
	return nil
}

// Load loads the watch history from the specified file
func Load(filename string) ([]youtube.SearchResultItem, error) {
	var history []youtube.SearchResultItem

	data, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		// Create empty history file if it doesn't exist
		emptyHistory := []youtube.SearchResultItem{}
		emptyHistoryData, err := json.Marshal(emptyHistory)
		if err != nil {
			utils.Logger.Error("Failed to marshal empty history.", zap.Error(err))
			return nil, err
		}
		
		err = os.WriteFile(filename, emptyHistoryData, 0o644)
		if err != nil {
			utils.Logger.Error("Failed to write empty history file.", zap.Error(err))
			return nil, err
		}
		
		utils.Logger.Info("Created new empty history file.", zap.String("filename", filename))
		return emptyHistory, nil
	}
	
	if err != nil {
		utils.Logger.Error("Failed to read history file.", zap.Error(err))
		return nil, err
	}

	err = json.Unmarshal(data, &history)
	if err != nil {
		utils.Logger.Error("Failed to unmarshal history data.", zap.Error(err))
		return nil, err
	}

	// Reverse the history to show most recent at the top
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	utils.Logger.Info("Successfully retrieved and processed watched videos.", zap.Int("count", len(history)))
	return history, nil
}

// Add adds a video to the watch history
func Add(video youtube.SearchResultItem, historyFilePath string) error {
	currentTime := time.Now().Unix()
	video.ViewedDate = currentTime
	
	history, err := Load(historyFilePath)
	if err != nil {
		utils.Logger.Error("Failed to read history.", zap.Error(err))
		return err
	}
	
	history = append(history, video)
	return Save(history, historyFilePath)
}

// GetWatchedVideos is an alias for Load for backward compatibility
func GetWatchedVideos(filename string) ([]youtube.SearchResultItem, error) {
	return Load(filename)
}

// FeedHistory is an alias for Add for backward compatibility  
func FeedHistory(selectedVideo youtube.SearchResultItem, historyFilePath string) error {
	return Add(selectedVideo, historyFilePath)
}