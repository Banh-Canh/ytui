package ui

import (
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"github.com/blacktop/go-termimg"
	"github.com/nfnt/resize"

	"github.com/Banh-Canh/ytui/internal/config"
	"github.com/Banh-Canh/ytui/internal/download"
	"github.com/Banh-Canh/ytui/internal/history"
	"github.com/Banh-Canh/ytui/internal/utils"
	"github.com/Banh-Canh/ytui/pkg/youtube"
)

type ViewType int

const (
	MainMenuView ViewType = iota
	SearchResultsView
	SubscribedView
	HistoryView
	SearchInputView
)

type menuItem struct {
	name        string
	id          string
	description string
}

type model struct {
	yt              *youtube.YouTube
	currentView     ViewType
	items           []interface{} // Can be menuItem or youtube.SearchResultItem
	videoItems      []youtube.SearchResultItem
	cursor          int
	currentDetails  *youtube.SearchResultItem
	loading         bool
	err             error
	searchQuery     string
	width           int
	height          int
	viewport        int
	viewportOffset  int
	selectedVideo   *youtube.SearchResultItem
	sortByDate      bool              // Whether to sort by date (newest first)
	// Video playback status
	currentPlayingItem  *youtube.SearchResultItem
	currentPlayPosition float64 // Current position in seconds
	currentPlayDuration float64 // Total duration in seconds
	isVideoPlaying      bool
}

// Messages
type videosLoadedMsg struct {
	items []youtube.SearchResultItem
}

type videoDetailsLoadedMsg struct {
	details *youtube.SearchResultItem
}

type searchResultsMsg struct {
	items []youtube.SearchResultItem
}

type errMsg struct {
	err error
}

func (e errMsg) Error() string { return e.err.Error() }


type playbackProgressMsg struct {
	position  float64
	duration  float64
	isPlaying bool
	itemName  string
}

type playbackStoppedMsg struct{}

type stopPlaybackMsg struct{}

type togglePauseMsg struct{}

type cycleSubtitleMsg struct{}

type cycleAudioMsg struct{}

// Styles
var (
	// Modern header styles
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E8E3F3")).
			Background(lipgloss.Color("#1a1b26")).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#3b4261")).
			Padding(0, 2).
			Margin(0).
			Bold(true)

	headerTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f7768e")).
			Bold(true)

	headerStatusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9ece6a"))
			
	headerOfflineStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f7768e")).
			Bold(true)

	headerDividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3b4261"))

	// Panel title styles (updated for consistency)
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f7768e")).
			Background(lipgloss.Color("#1f2335")).
			Padding(0, 1).
			Bold(true)

	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c0caf5"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1a1b26")).
			Background(lipgloss.Color("#f7768e")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565f89"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c0caf5"))

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3b4261")).
			Padding(1)
)

func Menu() {
	setupCleanupHandlers()
	
	// Clean up old thumbnail cache files
	go cleanupOldThumbnails()

	p := tea.NewProgram(initialModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		CleanupMpvProcesses()
		os.Exit(1)
	}
	CleanupMpvProcesses()
}

func initialModel() model {
	// Initialize YouTube client
	config := youtube.Config{
		InvidiousURL: viper.GetString("invidious.instance"),
		ProxyURL:     viper.GetString("invidious.proxy"),
		ClientID:     viper.GetString("youtube.clientid"),
		ClientSecret: viper.GetString("youtube.secretid"),
		RedirectURL:  "http://localhost:8080/oauth2callback",
	}
	yt := youtube.New(config)

	// Create main menu items
	mainMenuItems := []interface{}{
		menuItem{
			name:        "Search Videos",
			id:          "search",
			description: "Search for videos on YouTube/Invidious",
		},
		menuItem{
			name:        "Subscribed Channels",
			id:          "subscribed",
			description: "Browse videos from your subscribed channels",
		},
		menuItem{
			name:        "Watch History",
			id:          "history",
			description: "View your watch history",
		},
	}

	return model{
		yt:             yt,
		currentView:    MainMenuView,
		items:          mainMenuItems,
		cursor:         0,
		currentDetails: nil,
		loading:        false,
		width:          80,
		height:         24,
		viewport:       15,
		viewportOffset: 0,
		sortByDate:     true, // Default to sorting by date (newest first)
		currentPlayingItem:  nil,
		currentPlayPosition: 0,
		currentPlayDuration: 0,
		isVideoPlaying:      false,
	}
}

func (m model) Init() tea.Cmd {
	if m.err != nil {
		return nil
	}
	return tea.Batch(
		createProgressUpdateCmd(),
	)
}

// sortVideosByDate sorts videos by publication date (newest first if sortByDate is true)
// For search results, always preserve relevance order regardless of sortByDate setting
func (m *model) sortVideosByDate(videos []youtube.SearchResultItem) []youtube.SearchResultItem {
	// Never sort search results - preserve relevance order
	if m.currentView == SearchResultsView {
		return videos
	}
	
	// Only sort subscriptions and history if sortByDate is enabled
	if !m.sortByDate {
		return videos
	}
	
	// Create a copy to avoid modifying the original slice
	sorted := make([]youtube.SearchResultItem, len(videos))
	copy(sorted, videos)
	
	sort.Slice(sorted, func(i, j int) bool {
		// Sort by Published timestamp (newest first)
		return sorted[i].Published > sorted[j].Published
	})
	
	return sorted
}

func loadSearchResults(yt *youtube.YouTube, query string) tea.Cmd {
	return func() tea.Msg {
		results, err := yt.SearchVideos(query, 2)
		if err != nil {
			return errMsg{err}
		}
		return searchResultsMsg{results}
	}
}

func loadSubscribedVideos(yt *youtube.YouTube) tea.Cmd {
	return func() tea.Msg {
		// Check if using local subscriptions
		if viper.GetBool("channels.local") {
			results, err := yt.Subscriptions().GetVideosFromChannels(viper.GetStringSlice("channels.subscribed"))
			if err != nil {
				return errMsg{err}
			}
			return videosLoadedMsg{results}
		} else {
			// Authenticate and get subscription videos
			err := yt.Authenticate()
			if err != nil {
				return errMsg{err}
			}
			results, err := yt.GetSubscriptionVideos()
			if err != nil {
				return errMsg{err}
			}
			return videosLoadedMsg{results}
		}
	}
}

func loadHistoryVideos() tea.Cmd {
	return func() tea.Msg {
		configDir, err := config.GetConfigDirPath()
		if err != nil {
			return errMsg{err}
		}
		historyFilePath := filepath.Join(configDir, "watched_history.json")
		
		historyItems, err := history.Load(historyFilePath)
		if err != nil {
			return errMsg{err}
		}
		return videosLoadedMsg{historyItems}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.err != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateViewport()
		return m, nil

	case videosLoadedMsg:
		m.loading = false
		// Sort videos by date if enabled
		sortedVideos := m.sortVideosByDate(msg.items)
		// Convert to interface{} slice
		m.items = make([]interface{}, len(sortedVideos))
		for i, item := range sortedVideos {
			m.items[i] = item
		}
		m.videoItems = sortedVideos
		m.cursor = 0
		m.viewportOffset = 0
		m.updateViewport()
		m.updateCurrentDetails()
		
		
		return m, nil

	case searchResultsMsg:
		m.loading = false
		m.currentView = SearchResultsView
		// Sort videos by date if enabled
		sortedVideos := m.sortVideosByDate(msg.items)
		// Convert to interface{} slice
		m.items = make([]interface{}, len(sortedVideos))
		for i, item := range sortedVideos {
			m.items[i] = item
		}
		m.videoItems = sortedVideos
		m.cursor = 0
		m.viewportOffset = 0
		m.updateViewport()
		m.updateCurrentDetails()
		return m, nil


	case playbackProgressMsg:
		m.currentPlayPosition = msg.position
		m.currentPlayDuration = msg.duration
		wasPlaying := m.isVideoPlaying
		m.isVideoPlaying = msg.isPlaying

		if !msg.isPlaying && wasPlaying {
			// Video stopped, clear the current playing item
			m.currentPlayingItem = nil
			m.currentPlayPosition = 0
			m.currentPlayDuration = 0
		}
		// Continue updating progress only if something is playing or we just detected playback
		if msg.isPlaying || wasPlaying {
			return m, createProgressUpdateCmd()
		}
		return m, nil

	case playbackStoppedMsg:
		m.isVideoPlaying = false
		m.currentPlayingItem = nil
		m.currentPlayPosition = 0
		m.currentPlayDuration = 0
		return m, nil

	case stopPlaybackMsg:
		m.isVideoPlaying = false
		m.currentPlayingItem = nil
		m.currentPlayPosition = 0
		m.currentPlayDuration = 0
		return m, nil

	case togglePauseMsg:
		// The pause state will be updated automatically by the next progress update
		// No need to change m.isVideoPlaying here as it will be reflected in checkMpvStatus
		return m, nil

	case cycleSubtitleMsg:
		// Subtitle track cycling handled, no UI state changes needed
		return m, nil

	case cycleAudioMsg:
		// Audio track cycling handled, no UI state changes needed
		return m, nil

	case errMsg:
		m.err = msg.err
		m.loading = false
		return m, nil

	case tea.KeyMsg:
		// Handle search input first
		if m.currentView == SearchInputView {
			switch msg.String() {
			case "enter":
				if m.searchQuery != "" {
					m.loading = true
					return m, loadSearchResults(m.yt, m.searchQuery)
				}
			case "backspace":
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
				} else {
					m.currentView = MainMenuView
					m.items = []interface{}{
						menuItem{name: "Search Videos", id: "search", description: "Search for videos on YouTube/Invidious"},
						menuItem{name: "Subscribed Channels", id: "subscribed", description: "Browse videos from your subscribed channels"},
						menuItem{name: "Watch History", id: "history", description: "View your watch history"},
					}
					m.cursor = 0
					m.currentDetails = nil
				}
			case "escape":
				m.currentView = MainMenuView
				m.items = []interface{}{
					menuItem{name: "Search Videos", id: "search", description: "Search for videos on YouTube/Invidious"},
					menuItem{name: "Subscribed Channels", id: "subscribed", description: "Browse videos from your subscribed channels"},
					menuItem{name: "Watch History", id: "history", description: "View your watch history"},
				}
				m.cursor = 0
				m.searchQuery = ""
				m.currentDetails = nil
			case "ctrl+c":
				return m, tea.Quit
			default:
				if len(msg.String()) == 1 {
					m.searchQuery += msg.String()
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.updateViewport() // Ensure viewport is current before navigation
				m.cursor--
				m.updateViewport()
				m.updateCurrentDetails()
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.updateViewport() // Ensure viewport is current before navigation
				m.cursor++
				m.updateViewport()
				m.updateCurrentDetails()
			}
		case "g":
			if len(m.items) > 0 {
				m.cursor = 0
				m.viewportOffset = 0
				m.updateViewport()
				m.updateCurrentDetails()
			}
		case "G":
			if len(m.items) > 0 {
				m.cursor = len(m.items) - 1
				m.updateViewportForBottom()
				m.updateCurrentDetails()
			}
		case "pageup", "left":
			if len(m.items) > 0 {
				// Recalculate viewport size to ensure it's current
				m.updateViewport()
				pageSize := m.viewport
				newCursor := m.cursor - pageSize
				if newCursor < 0 {
					newCursor = 0
				}
				m.cursor = newCursor
				m.updateViewport()
				m.updateCurrentDetails()
			}
		case "pagedown", "right":
			if len(m.items) > 0 {
				// Recalculate viewport size to ensure it's current
				m.updateViewport()
				pageSize := m.viewport
				newCursor := m.cursor + pageSize
				if newCursor >= len(m.items) {
					newCursor = len(m.items) - 1
				}
				m.cursor = newCursor
				m.updateViewport()
				m.updateCurrentDetails()
			}
		case "enter":
			if len(m.items) > 0 {
				return m.selectItem()
			}
		case "backspace", "h":
			return m.goBack()
		case "p", " ":
			// If video is playing, toggle pause/play; otherwise play video
			if m.isVideoPlaying {
				return m, togglePause()
			} else if m.currentDetails != nil && (m.currentView == SearchResultsView || m.currentView == SubscribedView || m.currentView == HistoryView) {
				m.currentPlayingItem = m.currentDetails
				return m, tea.Batch(
					playVideo(*m.currentDetails, false),
					createProgressUpdateCmd(),
				)
			}
		case "d":
			if m.currentDetails != nil && (m.currentView == SearchResultsView || m.currentView == SubscribedView || m.currentView == HistoryView) {
				return m, downloadVideo(*m.currentDetails)
			}
		case "t":
			if m.currentDetails != nil && (m.currentView == SearchResultsView || m.currentView == SubscribedView || m.currentView == HistoryView) {
				return m, openThumbnail(*m.currentDetails)
			}
		case "s":
			// Stop video playback if playing, otherwise toggle sort
			if m.isVideoPlaying {
				return m, stopPlayback()
			} else if m.currentView == SubscribedView || m.currentView == HistoryView {
				// Toggle sort by date (only for subscriptions and history, not search results)
				m.sortByDate = !m.sortByDate
				// Re-sort current items
				if len(m.videoItems) > 0 {
					sortedVideos := m.sortVideosByDate(m.videoItems)
					m.items = make([]interface{}, len(sortedVideos))
					for i, item := range sortedVideos {
						m.items[i] = item
					}
					m.videoItems = sortedVideos
					m.cursor = 0
					m.viewportOffset = 0
					m.updateViewport()
					m.updateCurrentDetails()
				}
				return m, nil
			}
		case "/":
			// Allow search from any view
			m.currentView = SearchInputView
			m.searchQuery = ""
			return m, nil
		}
	}

	return m, nil
}

func (m *model) updateCurrentDetails() {
	previousDetails := m.currentDetails
	
	if len(m.items) > 0 && m.cursor < len(m.items) {
		switch item := m.items[m.cursor].(type) {
		case youtube.SearchResultItem:
			m.currentDetails = &item
		case menuItem:
			m.currentDetails = nil
		}
	}
	
	// Clear Kitty image if we changed to a different item or no item
	if previousDetails != m.currentDetails {
		if m.currentDetails == nil || 
		   (previousDetails != nil && previousDetails.VideoID != m.currentDetails.VideoID) {
			clearKittyImage()
		}
	}
}

func (m model) selectItem() (model, tea.Cmd) {
	item := m.items[m.cursor]
	
	switch v := item.(type) {
	case menuItem:
		switch v.id {
		case "search":
			m.currentView = SearchInputView
			m.searchQuery = ""
			return m, nil
		case "subscribed":
			m.currentView = SubscribedView
			m.loading = true
			return m, loadSubscribedVideos(m.yt)
		case "history":
			m.currentView = HistoryView
			m.loading = true
			return m, loadHistoryVideos()
		}
	case youtube.SearchResultItem:
		return m, playVideo(v, true)
	}
	
	return m, nil
}

func playVideo(video youtube.SearchResultItem, addToHistory bool) tea.Cmd {
	return func() tea.Msg {
		videoURL := "https://www.youtube.com/watch?v=" + video.VideoID
		utils.Logger.Info("Playing selected video in MPV.", zap.String("video_url", videoURL))

		// Close any existing ytui-launched videos before starting a new one
		if len(runningMpvProcesses) > 0 {
			CleanupMpvProcesses()
		}

		// Prepare mpv command with JSON IPC enabled for position tracking
		// Use a unique socket path to identify ytui-launched processes
		args := []string{"--input-ipc-server=/tmp/ytui-mpvsocket", "--title=ytui-player", videoURL}
		cmd := exec.Command("mpv", args...)

		// Add to global tracking so we can kill it on exit
		runningMpvProcesses = append(runningMpvProcesses, cmd)

		// Start playback tracking in background
		go func() {
			// Run mpv and wait for completion
			cmd.Run()

			// Remove from tracking list when mpv exits
			for i, p := range runningMpvProcesses {
				if p == cmd {
					runningMpvProcesses = append(runningMpvProcesses[:i], runningMpvProcesses[i+1:]...)
					break
				}
			}

			// Add to history if enabled and requested
			if addToHistory && viper.GetBool("history.enable") {
				configDir, err := config.GetConfigDirPath()
				if err == nil {
					historyFilePath := filepath.Join(configDir, "watched_history.json")
					history.Add(video, historyFilePath)
				}
			}
		}()

		return nil
	}
}

// getMpvPosition retrieves current playback position from mpv via JSON IPC
func getMpvPosition() float64 {
	conn, err := net.Dial("unix", "/tmp/ytui-mpvsocket")
	if err != nil {
		return 0
	}
	defer conn.Close()

	// Send command to get playback position
	command := map[string]interface{}{
		"command": []string{"get_property", "time-pos"},
	}

	jsonData, err := json.Marshal(command)
	if err != nil {
		return 0
	}

	conn.Write(append(jsonData, '\n'))

	// Read response
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return 0
	}

	var response map[string]interface{}
	if err := json.Unmarshal(buffer[:n], &response); err != nil {
		return 0
	}

	if data, ok := response["data"].(float64); ok {
		return data
	}

	return 0
}

// getMpvDuration retrieves total duration from mpv via JSON IPC
func getMpvDuration() float64 {
	conn, err := net.Dial("unix", "/tmp/ytui-mpvsocket")
	if err != nil {
		return 0
	}
	defer conn.Close()

	// Send command to get duration
	command := map[string]interface{}{
		"command": []string{"get_property", "duration"},
	}

	jsonData, err := json.Marshal(command)
	if err != nil {
		return 0
	}

	conn.Write(append(jsonData, '\n'))

	// Read response
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return 0
	}

	var response map[string]interface{}
	if err := json.Unmarshal(buffer[:n], &response); err != nil {
		return 0
	}

	if data, ok := response["data"].(float64); ok {
		return data
	}

	return 0
}

// stopMpv sends a quit command to mpv via JSON IPC
func stopMpv() error {
	conn, err := net.Dial("unix", "/tmp/ytui-mpvsocket")
	if err != nil {
		return err
	}
	defer conn.Close()

	// Send quit command
	command := map[string]interface{}{
		"command": []string{"quit"},
	}

	jsonData, err := json.Marshal(command)
	if err != nil {
		return err
	}

	conn.Write(append(jsonData, '\n'))
	return nil
}

// togglePauseMpv toggles pause/play state in mpv via JSON IPC
func togglePauseMpv() error {
	conn, err := net.Dial("unix", "/tmp/ytui-mpvsocket")
	if err != nil {
		return err
	}
	defer conn.Close()

	// Send pause toggle command
	command := map[string]interface{}{
		"command": []string{"cycle", "pause"},
	}

	jsonData, err := json.Marshal(command)
	if err != nil {
		return err
	}

	conn.Write(append(jsonData, '\n'))
	return nil
}

// checkMpvStatus checks if mpv is running and returns current status
func checkMpvStatus() (position, duration float64, isPlaying bool) {
	conn, err := net.Dial("unix", "/tmp/ytui-mpvsocket")
	if err != nil {
		return 0, 0, false
	}
	defer conn.Close()

	// First check if mpv is paused
	pauseCommand := map[string]interface{}{
		"command": []string{"get_property", "pause"},
	}

	jsonData, err := json.Marshal(pauseCommand)
	if err != nil {
		return 0, 0, false
	}

	conn.Write(append(jsonData, '\n'))

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return 0, 0, false
	}

	var pauseResponse map[string]interface{}
	if err := json.Unmarshal(buffer[:n], &pauseResponse); err != nil {
		return 0, 0, false
	}

	isPaused, _ := pauseResponse["data"].(bool)
	position = getMpvPosition()
	duration = getMpvDuration()

	return position, duration, !isPaused && position >= 0
}

// createProgressUpdateCmd creates a command that periodically updates playback progress
func createProgressUpdateCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		position, duration, isPlaying := checkMpvStatus()
		return playbackProgressMsg{
			position:  position,
			duration:  duration,
			isPlaying: isPlaying,
		}
	})
}

// stopPlayback stops the currently playing video
func stopPlayback() tea.Cmd {
	return func() tea.Msg {
		err := stopMpv()
		if err != nil {
			return errMsg{fmt.Errorf("failed to stop playback: %w", err)}
		}
		return stopPlaybackMsg{}
	}
}

// togglePause toggles pause/play state of the currently playing video
func togglePause() tea.Cmd {
	return func() tea.Msg {
		err := togglePauseMpv()
		if err != nil {
			return errMsg{fmt.Errorf("failed to toggle pause: %w", err)}
		}
		return togglePauseMsg{}
	}
}

func downloadVideo(video youtube.SearchResultItem) tea.Cmd {
	return func() tea.Msg {
		videoURL := "https://www.youtube.com/watch?v=" + video.VideoID
		downloadDir := viper.GetString("download_dir")
		utils.Logger.Info("Downloading selected video with yt-dlp.", zap.String("video_url", videoURL))
		
		go download.RunYTDLP(videoURL, downloadDir)
		
		return nil
	}
}

// Global variable to track running mpv processes
var runningMpvProcesses []*exec.Cmd

// Global variable to track current image for cleanup
var globalImageArea *imageArea

// Global variable to track current Kitty image for cleanup
var globalKittyImageArea *imageArea

// Shared HTTP client for image downloads to improve performance
var imageDownloadClient *http.Client

type imageArea struct {
	x      int
	y      int
	width  int
	height int
	itemID string
}

func init() {
	// Initialize shared HTTP client with optimized settings
	transport := &http.Transport{
		MaxIdleConns:          20,
		MaxIdleConnsPerHost:   5,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     false,
	}
	imageDownloadClient = &http.Client{
		Timeout:   8 * time.Second,
		Transport: transport,
	}
}

func (m model) goBack() (model, tea.Cmd) {
	switch m.currentView {
	case SearchResultsView, SubscribedView, HistoryView, SearchInputView:
		// Back to main menu
		m.currentView = MainMenuView
		m.items = []interface{}{
			menuItem{name: "Search Videos", id: "search", description: "Search for videos on YouTube/Invidious"},
			menuItem{name: "Subscribed Channels", id: "subscribed", description: "Browse videos from your subscribed channels"},
			menuItem{name: "Watch History", id: "history", description: "View your watch history"},
		}
		m.cursor = 0
		m.viewportOffset = 0
		m.currentDetails = nil
		m.updateViewport()
		return m, nil
	case MainMenuView:
		return m, tea.Quit
	}
	return m, nil
}

func (m *model) updateViewport() {
	m.viewport = m.height - 8
	if m.viewport < 5 {
		m.viewport = 5
	}
	
	if m.cursor < m.viewportOffset {
		m.viewportOffset = m.cursor
	} else if m.cursor >= m.viewportOffset+m.viewport {
		m.viewportOffset = m.cursor - m.viewport + 1
	}
}

func (m *model) updateViewportForBottom() {
	m.viewport = m.height - 8
	if m.viewport < 5 {
		m.viewport = 5
	}
	
	if len(m.items) > m.viewport {
		m.viewportOffset = len(m.items) - m.viewport
	} else {
		m.viewportOffset = 0
	}
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress 'q' to quit.", m.err)
	}

	if m.loading {
		return "Loading..."
	}

	// Use cached viewport calculations for better performance
	viewport := m.viewport
	if viewport < 5 {
		viewport = 5
	}
	viewportOffset := m.viewportOffset

	// Render modern header
	header := m.renderHeader()

	// Calculate exact dimensions to fit in terminal (account for header)
	leftWidth := (m.width / 2) - 2
	rightWidth := m.width - leftWidth - 2
	contentHeight := m.height - 4 // Leave space for header, help, and spacing

	// Create panels with exact sizing
	leftPane := m.renderItemList(leftWidth, contentHeight, viewport, viewportOffset)
	rightPane := m.renderDetails(rightWidth, contentHeight)

	// Style panels with modern theme
	leftStyle := lipgloss.NewStyle().
		Width(leftWidth).
		Height(contentHeight).
		Border(lipgloss.RoundedBorder(), false, true, false, false).
		BorderForeground(lipgloss.Color("#3b4261"))

	rightStyle := lipgloss.NewStyle().
		Width(rightWidth).
		Height(contentHeight).
		Border(lipgloss.RoundedBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("#3b4261"))

	leftPanel := leftStyle.Render(leftPane)
	rightPanel := rightStyle.Render(rightPane)

	// Create help footer
	help := m.renderHelp()

	// Join horizontally and add help at bottom
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	finalContent := lipgloss.JoinVertical(lipgloss.Left, header, content, help)
	
	// Render Kitty image overlay after lipgloss content (separate layer)
	// Note: This is done in the background to not interfere with lipgloss rendering
	m.renderKittyImageOverlay(leftWidth, rightWidth, contentHeight)
	
	return finalContent
}

func (m model) renderItemList(width, height, viewport, viewportOffset int) string {
	var content strings.Builder
	
	// Title based on current view
	title := ""
	switch m.currentView {
	case MainMenuView:
		title = "YouTube TUI"
	case SearchInputView:
		title = fmt.Sprintf("Search: %s", m.searchQuery)
		if len(title) > width-4 {
			title = title[:width-7] + "..."
		}
	case SearchResultsView:
		title = "Search Results (by relevance)"
	case SubscribedView:
		title = "Subscribed Channels"
		if m.sortByDate {
			title += " (sorted by date)"
		}
	case HistoryView:
		title = "Watch History"
		if m.sortByDate {
			title += " (sorted by date)"
		}
	}
	
	content.WriteString(titleStyle.Width(width-4).Render(title))
	content.WriteString("\n")

	if len(m.items) == 0 {
		content.WriteString(dimStyle.Render("No items found"))
		return content.String()
	}

	// Use passed viewport parameters
	availableLines := viewport
	if availableLines < 1 {
		availableLines = 1
	}

	// Show only items within viewport
	start := viewportOffset
	end := start + availableLines
	if end > len(m.items) {
		end = len(m.items)
	}

	// Render visible items
	for i := start; i < end; i++ {
		var itemText string
		
		switch item := m.items[i].(type) {
		case menuItem:
			itemText = item.name
		case youtube.SearchResultItem:
			itemText = item.Title
			if len(itemText) > width-10 {
				itemText = itemText[:width-13] + "..."
			}
			itemText += " - " + item.Author
		}
		
		// Truncate if too long
		maxItemWidth := width - 6
		if maxItemWidth < 10 {
			maxItemWidth = 10
		}
		if len(itemText) > maxItemWidth {
			itemText = lipgloss.NewStyle().Width(maxItemWidth).Render(itemText)
		}
		
		if i == m.cursor {
			content.WriteString(selectedStyle.Render(" ▶ " + itemText + " "))
		} else {
			content.WriteString(itemStyle.Render("   " + itemText))
		}
		
		if i < end-1 {
			content.WriteString("\n")
		}
	}
	
	// Add scroll indicators
	if viewportOffset > 0 {
		content.WriteString("\n" + dimStyle.Render("  ↑ more items above"))
	}
	if end < len(m.items) {
		content.WriteString("\n" + dimStyle.Render("  ↓ more items below"))
	}
	
	return content.String()
}

func (m model) renderDetails(width, height int) string {
	if m.currentDetails == nil {
		// Show menu item descriptions or general info
		if len(m.items) > 0 && m.cursor < len(m.items) {
			if item, ok := m.items[m.cursor].(menuItem); ok {
				return infoStyle.Render(item.description)
			}
		}
		if m.currentView == SearchInputView {
			return infoStyle.Render("Type your search query and press Enter")
		}
		return dimStyle.Render("Select an item to view details")
	}
	
	var details strings.Builder
	linesUsed := 0
	maxLines := height - 2
	
	// Title
	details.WriteString(titleStyle.Width(width-4).Render("Video Details"))
	details.WriteString("\n")
	linesUsed++
	
	if linesUsed >= maxLines {
		return details.String()
	}

	
	// Video title
	title := m.currentDetails.Title
	if len(title) > width-8 {
		title = title[:width-11] + "..."
	}
	details.WriteString(infoStyle.Render(fmt.Sprintf("Title: %s", title)))
	details.WriteString("\n")
	linesUsed++
	if linesUsed >= maxLines {
		return details.String()
	}
	
	// Author
	details.WriteString(infoStyle.Render(fmt.Sprintf("Author: %s", m.currentDetails.Author)))
	details.WriteString("\n")
	linesUsed++
	if linesUsed >= maxLines {
		return details.String()
	}
	
	// Duration
	duration := time.Duration(m.currentDetails.LengthSeconds) * time.Second
	details.WriteString(infoStyle.Render(fmt.Sprintf("Duration: %s", duration.String())))
	details.WriteString("\n")
	linesUsed++
	if linesUsed >= maxLines {
		return details.String()
	}
	
	// Views
	if m.currentDetails.ViewCountText != "" {
		details.WriteString(infoStyle.Render(fmt.Sprintf("Views: %s", m.currentDetails.ViewCountText)))
		details.WriteString("\n")
		linesUsed++
		if linesUsed >= maxLines {
			return details.String()
		}
	}
	
	// Published
	if m.currentDetails.PublishedText != "" {
		details.WriteString(infoStyle.Render(fmt.Sprintf("Published: %s", m.currentDetails.PublishedText)))
		details.WriteString("\n")
		linesUsed++
		if linesUsed >= maxLines {
			return details.String()
		}
	}
	
	// URL
	videoURL := "https://www.youtube.com/watch?v=" + m.currentDetails.VideoID
	if len(videoURL) > width-6 {
		videoURL = videoURL[:width-9] + "..."
	}
	details.WriteString(dimStyle.Render(fmt.Sprintf("URL: %s", videoURL)))
	details.WriteString("\n")
	linesUsed++
	if linesUsed >= maxLines {
		return details.String()
	}
	
	// Description with word wrapping
	if m.currentDetails.Description != "" && linesUsed < maxLines-2 {
		details.WriteString("\n")
		details.WriteString(infoStyle.Render("Description:"))
		details.WriteString("\n")
		linesUsed += 2
		
		description := m.currentDetails.Description
		words := strings.Fields(description)
		line := ""
		lineWidth := width - 4
		if lineWidth < 20 {
			lineWidth = 20
		}
		
		for _, word := range words {
			if linesUsed >= maxLines {
				break
			}
			
			if len(line)+len(word)+1 > lineWidth {
				details.WriteString(dimStyle.Render(line))
				details.WriteString("\n")
				linesUsed++
				line = word
			} else {
				if line != "" {
					line += " "
				}
				line += word
			}
		}
		
		if line != "" && linesUsed < maxLines {
			details.WriteString(dimStyle.Render(line))
		}
	}
	
	return details.String()
}

// calculateDetailsLines estimates how many lines the video details will use
func (m model) calculateDetailsLines(rightWidth int) int {
	if m.currentDetails == nil {
		return 0
	}
	
	lines := 1 // Title line
	lines += 6 // Title, Author, Duration, Views, Published, URL (basic info)
	
	// Account for description if present and space allows
	if m.currentDetails.Description != "" {
		lines += 2 // Description header + at least one line
		// Rough estimate of description lines (word wrap)
		words := len(strings.Fields(m.currentDetails.Description))
		lineWidth := rightWidth - 4
		if lineWidth < 20 {
			lineWidth = 20
		}
		avgWordsPerLine := lineWidth / 6 // Rough estimate
		if avgWordsPerLine > 0 {
			descLines := (words / avgWordsPerLine) + 1
			if descLines > 4 {
				descLines = 4 // Limit description lines
			}
			lines += descLines
		}
	}
	
	return lines + 1 // Add buffer line
}

// renderKittyImageOverlay renders a Kitty protocol image overlay in the right panel
func (m model) renderKittyImageOverlay(leftWidth, rightWidth, contentHeight int) {
	if m.currentDetails == nil {
		// Clear any existing Kitty image if no details
		clearKittyImage()
		return
	}

	// Calculate position in right panel
	rightPanelX := leftWidth + 2  // Account for left border
	
	// Calculate Y position based on rendered content to avoid overlap
	// Account for: panel title (1) + basic video info (6-7 lines) + buffer
	detailsLines := m.calculateDetailsLines(rightWidth)
	rightPanelY := 4 + detailsLines  // Start image below the text content

	// Check if we have enough space for image after the details
	maxLines := contentHeight - 2
	availableLines := maxLines - detailsLines
	if availableLines <= 8 {
		// Not enough space for meaningful image, clear any existing image
		clearKittyImage()
		return
	}

	// Dynamic image sizing based on available space after details
	thumbWidth := rightWidth - 4  // Account for borders
	thumbHeight := (availableLines * 9) / 20  // Proportional to available space after details

	// Constrain dimensions (similar to jtui constraints)
	if thumbWidth > 40 {
		thumbWidth = 40
	}
	if thumbHeight > 15 {
		thumbHeight = 15
	}
	if thumbWidth < 20 {
		thumbWidth = 20
	}
	if thumbHeight < 8 {
		thumbHeight = 8
	}

	// For YouTube videos, construct thumbnail URL
	imageURL := fmt.Sprintf("https://img.youtube.com/vi/%s/maxresdefault.jpg", m.currentDetails.VideoID)
	currentItemID := m.currentDetails.VideoID

	// Only render if different from current image or no current image
	if globalKittyImageArea == nil || globalKittyImageArea.itemID != currentItemID {
		// Attempt to render Kitty image
		err := renderKittyImageAt(imageURL, rightPanelX, rightPanelY, thumbWidth, thumbHeight, currentItemID)
		if err != nil {
			// Kitty protocol failed, clear any existing image
			clearKittyImage()
		}
	}
}

// CleanupMpvProcesses kills any running ytui-launched mpv processes
func CleanupMpvProcesses() {
	for _, cmd := range runningMpvProcesses {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}
	// Clear the tracking list
	runningMpvProcesses = nil

	// Also clean up the socket file to prevent conflicts
	os.Remove("/tmp/ytui-mpvsocket")
	
	// Clear any Kitty images
	clearKittyImage()
}

// setupCleanupHandlers sets up signal handlers to cleanup mpv processes on exit
func setupCleanupHandlers() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-c
		CleanupMpvProcesses()
		os.Exit(0)
	}()
}


// yaziThumbnailConfig holds configuration for Yazi-style image processing
type yaziThumbnailConfig struct {
	filter      resize.InterpolationFunction
	quality     int
	maxWidth    int
	maxHeight   int
	minWidth    int
	minHeight   int
}

// getYaziConfig returns Yazi-inspired configuration for high-quality thumbnails
func getYaziConfig() yaziThumbnailConfig {
	// Read user configuration for image filter (like Yazi's image_filter setting)
	filterStr := viper.GetString("image_filter")
	var filter resize.InterpolationFunction
	switch filterStr {
	case "nearest":
		filter = resize.NearestNeighbor
	case "bilinear", "triangle":
		filter = resize.Bilinear
	case "bicubic", "catmull-rom":
		filter = resize.Bicubic
	case "lanczos2":
		filter = resize.Lanczos2
	case "lanczos3":
		filter = resize.Lanczos3
	default:
		filter = resize.Lanczos3 // Default to highest quality like Yazi
	}

	// Read quality setting (like Yazi's image_quality setting)
	quality := viper.GetInt("image_quality")
	if quality <= 0 || quality > 100 {
		quality = 85 // High default quality
	}

	return yaziThumbnailConfig{
		filter:    filter,
		quality:   quality,
		maxWidth:  70,
		maxHeight: 30,
		minWidth:  20,
		minHeight: 8,
	}
}


// renderKittyImageAt renders a Kitty protocol image at specific terminal coordinates
func renderKittyImageAt(imageURL string, x, y, width, height int, itemID string) error {
	if imageURL == "" {
		return fmt.Errorf("no image URL provided")
	}

	config := getYaziConfig()

	// Validate and constrain dimensions
	if width > config.maxWidth {
		width = config.maxWidth
	}
	if height > config.maxHeight {
		height = config.maxHeight
	}
	if width < config.minWidth {
		width = config.minWidth
	}
	if height < config.minHeight {
		height = config.minHeight
	}

	// Check for processed file first
	processedFile := fmt.Sprintf("/tmp/ytui_kitty_%s_%dx%d.jpg", itemID, width, height)

	// Check if processed image already exists for this exact size
	if _, err := os.Stat(processedFile); os.IsNotExist(err) {
		if err := downloadAndProcessImageForTerminal(imageURL, processedFile, width, height, config); err != nil {
			return fmt.Errorf("failed to process image: %w", err)
		}
	}

	// Clear previous Kitty image if different item
	if globalKittyImageArea != nil && globalKittyImageArea.itemID != itemID {
		clearKittyImage()
	}

	// Render using Kitty protocol
	img, err := termimg.Open(processedFile)
	if err != nil {
		os.Remove(processedFile) // Clean up on error
		return fmt.Errorf("failed to open processed image: %w", err)
	}

	// Use Kitty protocol for high-quality rendering
	kittyData, err := img.Width(width).Height(height).Protocol(termimg.Kitty).Render()
	if err != nil {
		return fmt.Errorf("failed to render Kitty image: %w", err)
	}

	// Position cursor and render
	fmt.Printf("\x1b[%d;%dH", y+1, x+1) // Move to position (1-indexed)
	fmt.Print(kittyData)

	// Track the image area for cleanup
	globalKittyImageArea = &imageArea{
		x:      x,
		y:      y,
		width:  width,
		height: height,
		itemID: itemID,
	}

	return nil
}

// clearKittyImage clears the current Kitty protocol image
func clearKittyImage() {
	if globalKittyImageArea == nil {
		return
	}

	// Send Kitty protocol delete command to clear all images
	fmt.Print("\x1b_Gq=2,a=d,d=A\x1b\\")

	// Also clear with spaces for safety
	for row := 0; row < globalKittyImageArea.height; row++ {
		fmt.Printf("\x1b[%d;%dH%s", 
			globalKittyImageArea.y+row+1, 
			globalKittyImageArea.x+1, 
			strings.Repeat(" ", globalKittyImageArea.width))
	}

	globalKittyImageArea = nil
}

// downloadAndProcessImageForTerminal downloads and scales image perfectly for terminal display
func downloadAndProcessImageForTerminal(imageURL, outputPath string, termWidth, termHeight int, config yaziThumbnailConfig) error {
	// Use shared HTTP client for better performance and connection reuse
	resp, err := imageDownloadClient.Get(imageURL)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned HTTP %d for image", resp.StatusCode)
	}

	// Decode image
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Get original dimensions
	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	// Calculate optimal pixel dimensions for terminal characters
	// Use more accurate character dimensions for better image quality
	targetPixelWidth := termWidth * 9   // More accurate character width
	targetPixelHeight := termHeight * 18 // More accurate character height (halfblock)
	
	// Calculate target dimensions while preserving aspect ratio (Yazi approach)
	targetWidth, targetHeight := calculateYaziDimensions(origWidth, origHeight, targetPixelWidth, targetPixelHeight)

	// Apply high-quality resampling with performance optimization
	var resized image.Image
	if targetWidth != origWidth || targetHeight != origHeight {
		// Only use high-quality filter for significant resize operations
		if float64(targetWidth)/float64(origWidth) < 0.5 || float64(targetHeight)/float64(origHeight) < 0.5 {
			resized = resize.Resize(uint(targetWidth), uint(targetHeight), img, config.filter)
		} else {
			// Use faster bilinear for minor resizes to improve performance
			resized = resize.Resize(uint(targetWidth), uint(targetHeight), img, resize.Bilinear)
		}
	} else {
		resized = img
	}

	// Save processed image with high quality
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Use JPEG with high quality (like Yazi's quality setting)
	return jpeg.Encode(file, resized, &jpeg.Options{Quality: config.quality})
}

// calculateYaziDimensions calculates optimal dimensions while preserving aspect ratio
func calculateYaziDimensions(origWidth, origHeight, maxWidth, maxHeight int) (int, int) {
	if origWidth <= maxWidth && origHeight <= maxHeight {
		return origWidth, origHeight
	}

	// Calculate scaling ratios
	widthRatio := float64(maxWidth) / float64(origWidth)
	heightRatio := float64(maxHeight) / float64(origHeight)

	// Use the smaller ratio to ensure both dimensions fit
	ratio := widthRatio
	if heightRatio < widthRatio {
		ratio = heightRatio
	}

	return int(float64(origWidth) * ratio), int(float64(origHeight) * ratio)
}

// cleanupOldThumbnails removes old thumbnail cache files to prevent disk space issues
func cleanupOldThumbnails() {

	// Clean up old cache directory completely to force fresh generation
	oldCacheDir := "/tmp/ytui_thumbs"
	if _, err := os.Stat(oldCacheDir); err == nil {
		os.RemoveAll(oldCacheDir) // Remove all old cache files
	}

	// Also clean up processed images older than 2 hours (extended for better performance)
	imageCutoff := time.Now().Add(-2 * time.Hour)
	
	// Use more efficient cleanup for /tmp directory
	tmpEntries, err := os.ReadDir("/tmp")
	if err == nil {
		for _, entry := range tmpEntries {
			name := entry.Name()
			if !entry.IsDir() && 
				(strings.HasPrefix(name, "ytui_img_") || strings.HasPrefix(name, "ytui_kitty_")) && 
				strings.HasSuffix(name, ".jpg") {
				if info, err := entry.Info(); err == nil && info.ModTime().Before(imageCutoff) {
					os.Remove(filepath.Join("/tmp", name))
				}
			}
		}
	}
}

func openThumbnail(video youtube.SearchResultItem) tea.Cmd {
	return func() tea.Msg {
		// For YouTube videos, construct thumbnail URL
		imageURL := fmt.Sprintf("https://img.youtube.com/vi/%s/maxresdefault.jpg", video.VideoID)
		
		thumbnailPath := fmt.Sprintf("/tmp/ytui_thumb_%s.jpg", video.VideoID)
		
		// Download thumbnail if it doesn't exist
		if _, err := os.Stat(thumbnailPath); os.IsNotExist(err) {
			resp, err := http.Get(imageURL)
			if err != nil {
				return nil
			}
			defer resp.Body.Close()

			file, err := os.Create(thumbnailPath)
			if err != nil {
				return nil
			}
			defer file.Close()

			_, err = io.Copy(file, resp.Body)
			if err != nil {
				return nil
			}
		}

		// Get configured image viewer or use default
		imageViewer := viper.GetString("image_viewer")
		if imageViewer == "" {
			imageViewer = "xdg-open" // Default for Linux
		}

		// Open thumbnail with configured tool
		cmd := exec.Command(imageViewer, thumbnailPath)
		cmd.Start()

		return nil
	}
}

// Help text
var helpText = strings.Join([]string{
	"↑↓/jk: navigate",
	"←→/PgUp/PgDn: page",
	"g/G: top/bottom",
	"Enter: select/play",
	"h/Bksp: back",
	"t: thumbnail",
	"s: sort by date/stop",
	"p/Space: play/pause",
	"d: download",
	"/: search",
	"q: quit",
}, " • ")

func (m model) renderHeader() string {
	// App title with modern styling
	appName := headerTitleStyle.Render("📺 YTUI")
	
	// Current view indicator
	var currentLocation string
	switch m.currentView {
	case MainMenuView:
		currentLocation = "🏠 Main Menu"
	case SearchInputView:
		currentLocation = "🔍 Search: " + m.searchQuery
	case SearchResultsView:
		currentLocation = "📋 Search Results"
	case SubscribedView:
		currentLocation = "📺 Subscribed Channels"
	case HistoryView:
		currentLocation = "🕰️ Watch History"
	default:
		currentLocation = "📋 Content"
	}
	
	// Connection status (simplified for YouTube)
	status := headerStatusStyle.Render("🌐 ONLINE")
	
	// Create dividers
	divider := headerDividerStyle.Render(" │ ")
	
	// Build header content
	leftSide := appName + divider + currentLocation
	rightSide := status
	
	// Calculate spacing
	usedSpace := lipgloss.Width(leftSide) + lipgloss.Width(rightSide)
	spacer := strings.Repeat(" ", max(1, m.width-usedSpace-4)) // -4 for padding
	
	headerContent := leftSide + spacer + rightSide
	
	return headerStyle.Width(m.width).Render(headerContent)
}

func (m model) renderHelp() string {
	if len(helpText) > m.width-2 {
		return dimStyle.Render(lipgloss.NewStyle().Width(m.width-2).Render(helpText))
	}
	
	return dimStyle.Render(helpText)
}

// Helper function for max calculation
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}