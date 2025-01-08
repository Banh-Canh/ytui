package player

import (
	"os/exec"

	"go.uber.org/zap"

	"github.com/Banh-Canh/ytui/internal/utils"
)

func RunMPV(videoPath string) {
	utils.Logger.Debug("Starting the video with mpv...")
	args := []string{
		"--ytdl-format=bestvideo[ext=mp4][height<=?2160]+bestaudio[ext=m4a]",
		"--ytdl-raw-options=mark-watched=,cookies-from-browser=firefox",
		videoPath, // Path to the video file
	}
	cmd := exec.Command("mpv", args...)
	err := cmd.Start()
	if err != nil {
		utils.Logger.Error("Failed to start mpv.", zap.Error(err))
	} else {
		utils.Logger.Info("Mpv started.")
	}
}
