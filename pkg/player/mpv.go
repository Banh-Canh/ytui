package player

import (
	"fmt"
	"os/exec"
)

func RunMPV(videoPath string) {
	args := []string{
		"--ytdl-format=bestvideo[ext=mp4][height<=?2160]+bestaudio[ext=m4a]",
		"--ytdl-raw-options=mark-watched=,cookies-from-browser=firefox",
		videoPath, // Path to the video file
	}
	cmd := exec.Command("mpv", args...)
	err := cmd.Start()
	if err != nil {
		fmt.Printf("Error running MPV: %v\n", err)
	}
}
