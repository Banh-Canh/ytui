package download

import (
	"os"
	"os/exec"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/Banh-Canh/ytui/pkg/utils"
)

func RunYTDLP(videoPath, outputPath string) {
	utils.Logger.Debug("Downloading the video with yt-dlp...")

	err := os.MkdirAll(outputPath, os.ModePerm)
	if err != nil {
		utils.Logger.Error("Failed to create output directory.", zap.Error(err))
		return
	}
	args := []string{
		"--format=bestvideo[ext=mp4][height<=?2160]+bestaudio[ext=m4a]",
		"--mark-watched",
		"--cookies-from-browser=firefox",
		"-o", filepath.Join(outputPath, "%(title)s.%(ext)s"), // Set output path dynamically
		videoPath, // URL or video path
	}
	cmd := exec.Command("yt-dlp", args...)

	// Set stdout and stderr to os.Stdout and os.Stderr so we can see the output in the terminal
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		utils.Logger.Error("Failed to start yt-dlp.", zap.Error(err))
	} else {
		utils.Logger.Info("yt-dlp started.")
	}
}
