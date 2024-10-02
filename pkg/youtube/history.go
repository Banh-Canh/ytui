package youtube

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"github.com/Banh-Canh/ytui/pkg/utils"
)

func SaveHistory(history *[]SearchResultItem, filename string) error {
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
	utils.Logger.Info("Successfully saved history data.", zap.String("filename", filename), zap.Int("item_count", len(*history)))
	return nil
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
			utils.Logger.Error("Failed to marshal empty history.", zap.Error(err))
			return nil, err
		}
		err = os.WriteFile(filename, emptyHistoryData, 0o644)
		if err != nil {
			utils.Logger.Error("Failed to write empty history file.", zap.Error(err))
			return nil, err
		}
		utils.Logger.Info("Created new empty history file.", zap.String("filename", filename))
		return emptyHistory, nil // Return empty history
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

	// Invert the history index, for displaying most recent at the top.
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	utils.Logger.Info("Successfully retrieved and processed watched videos.", zap.Int("count", len(history)))
	return history, nil
}

func FeedHistory(selectedVideo SearchResultItem, historyFilePath string) {
	currentTime := time.Now().Unix()
	selectedVideo.ViewedDate = currentTime
	history, err := GetWatchedVideos(historyFilePath)
	if err != nil {
		utils.Logger.Error("Failed to read history.", zap.Error(err))
	}
	history = append(history, selectedVideo)
	// Save updated history
	err = SaveHistory(&history, historyFilePath)
	if err != nil {
		utils.Logger.Error("Failed to save history.", zap.Error(err))
	}
}
