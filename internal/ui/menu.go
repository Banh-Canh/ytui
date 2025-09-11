package ui

import (
	"fmt"
	"io"
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

	"github.com/Banh-Canh/ytui/internal/config"
	"github.com/Banh-Canh/ytui/internal/download"
	"github.com/Banh-Canh/ytui/internal/history"
	"github.com/Banh-Canh/ytui/internal/player"
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
	thumbnailCache  map[string]string // Cache for rendered thumbnails
	sortByDate      bool              // Whether to sort by date (newest first)
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

type thumbnailLoadedMsg struct {
	itemID    string
	cacheKey  string
	thumbnail string
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#FF0000")).
		Padding(0, 1).
		Bold(true)

	itemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA"))

	selectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#FF0000")).
		Bold(true)

	dimStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))

	infoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA"))

	panelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Padding(1)
)

func Menu() {
	setupCleanupHandlers()
	
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
		thumbnailCache: make(map[string]string),
		sortByDate:     true, // Default to sorting by date (newest first)
	}
}

func (m model) Init() tea.Cmd {
	if m.err != nil {
		return nil
	}
	return nil
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
		
		// Limit cache size to prevent memory growth
		if len(m.thumbnailCache) > 20 {
			// Clear old entries
			m.thumbnailCache = make(map[string]string)
		}
		
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

	case thumbnailLoadedMsg:
		// Store thumbnail in cache
		m.thumbnailCache[msg.cacheKey] = msg.thumbnail
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
			if m.currentDetails != nil && (m.currentView == SearchResultsView || m.currentView == SubscribedView || m.currentView == HistoryView) {
				return m, playVideo(*m.currentDetails, false)
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
			// Toggle sort by date (only for subscriptions and history, not search results)
			if m.currentView == SubscribedView || m.currentView == HistoryView {
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
	if len(m.items) > 0 && m.cursor < len(m.items) {
		switch item := m.items[m.cursor].(type) {
		case youtube.SearchResultItem:
			m.currentDetails = &item
		case menuItem:
			m.currentDetails = nil
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
		
		cmd := exec.Command("mpv", videoURL)
		runningMpvProcesses = append(runningMpvProcesses, cmd)
		
		go func() {
			player.RunMPV(videoURL)
			
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

	// Calculate exact dimensions
	leftWidth := (m.width / 2) - 2
	rightWidth := m.width - leftWidth - 2
	contentHeight := m.height - 2

	// Ensure viewport is calculated correctly for display
	viewport := m.height - 8 
	if viewport < 5 {
		viewport = 5
	}
	
	// Create panels using current model state
	leftPane := m.renderItemList(leftWidth, contentHeight, viewport, m.viewportOffset)
	rightPane := m.renderDetails(rightWidth, contentHeight)

	// Style panels
	leftStyle := lipgloss.NewStyle().
		Width(leftWidth).
		Height(contentHeight).
		Border(lipgloss.RoundedBorder(), false, true, false, false).
		BorderForeground(lipgloss.Color("#333"))
	
	rightStyle := lipgloss.NewStyle().
		Width(rightWidth).
		Height(contentHeight).
		Border(lipgloss.RoundedBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("#333"))

	leftPanel := leftStyle.Render(leftPane)
	rightPanel := rightStyle.Render(rightPane)

	// Create help footer
	help := m.renderHelp()

	// Join horizontally and add help at bottom
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	finalContent := lipgloss.JoinVertical(lipgloss.Left, content, help)
	
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

	// Render thumbnail if available and there's space (need at least 12 lines total)
	if maxLines > 12 {
		// For YouTube videos, construct thumbnail URL
		imageURL := fmt.Sprintf("https://img.youtube.com/vi/%s/maxresdefault.jpg", m.currentDetails.VideoID)
		if imageURL != "" {
			// Calculate dimensions - make it slightly bigger
			thumbWidth := width - 2
			if thumbWidth > 50 {
				thumbWidth = 50
			}
			if thumbWidth < 30 {
				thumbWidth = 30
			}
			// Use more space for height - about 45% of available space
			thumbHeight := (maxLines * 9) / 20 
			if thumbHeight > 18 {
				thumbHeight = 18 // Cap at 18 lines
			}
			if thumbHeight < 10 {
				thumbHeight = 10 // Minimum 10 lines
			}
			
			// Check cache first - ensure it's for the current item
			currentItemID := m.currentDetails.VideoID
			cacheKey := fmt.Sprintf("%s_%d_%d", currentItemID, thumbWidth, thumbHeight)
			
			if cachedThumbnail, exists := m.thumbnailCache[cacheKey]; exists {
				// Use cached thumbnail for this specific item
				thumbnailLines := strings.Count(cachedThumbnail, "\n")
				if thumbnailLines == 0 && cachedThumbnail != "" {
					thumbnailLines = 1
				}
				
				if linesUsed + thumbnailLines + 3 < maxLines {
					details.WriteString(cachedThumbnail)
					details.WriteString("\n\n")
					linesUsed += thumbnailLines + 2
				}
			} else {
				// Render and cache immediately (but with optimizations)
				if thumbnail, err := renderThumbnailInlineOptimized(imageURL, thumbWidth, thumbHeight, currentItemID); err == nil {
					// Cache the result for this specific item
					m.thumbnailCache[cacheKey] = thumbnail
					
					// Count actual lines in the rendered thumbnail
					thumbnailLines := strings.Count(thumbnail, "\n")
					if thumbnailLines == 0 && thumbnail != "" {
						thumbnailLines = 1
					}
					
					// Check if we have space for the thumbnail plus some detail text
					if linesUsed + thumbnailLines + 3 < maxLines {
						details.WriteString(thumbnail)
						details.WriteString("\n\n")
						linesUsed += thumbnailLines + 2
					}
				}
			}
			
			if linesUsed >= maxLines {
				return details.String()
			}
		}
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

// CleanupMpvProcesses kills any running mpv processes when ytui exits
func CleanupMpvProcesses() {
	for _, cmd := range runningMpvProcesses {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}
	runningMpvProcesses = nil
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

// renderThumbnailInline renders thumbnail image inline in terminal using halfblock renderer
func renderThumbnailInline(imageURL string, width, height int) (string, error) {
	if imageURL == "" {
		return "", fmt.Errorf("no image URL provided")
	}

	// Validate dimensions to prevent excessive sizes
	if width > 60 {
		width = 60
	}
	if height > 25 {
		height = 25
	}
	if width < 15 {
		width = 15
	}
	if height < 6 {
		height = 6
	}

	// Download image to temporary file with timeout
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(imageURL)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "ytui_thumb_*.jpg")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Copy image data to temp file with size limit
	limitedReader := io.LimitReader(resp.Body, 10*1024*1024) // 10MB limit
	_, err = io.Copy(tmpFile, limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Render image using go-termimg with forced halfblock renderer
	img, err := termimg.Open(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to open image: %w", err)
	}

	// Configure dimensions and explicitly force halfblock renderer
	rendered, err := img.Width(width).Height(height).Protocol(termimg.Halfblocks).Render()
	if err != nil {
		return "", fmt.Errorf("failed to render image: %w", err)
	}

	// Validate the output doesn't exceed expected dimensions
	lines := strings.Split(rendered, "\n")
	if len(lines) > height+2 { // Allow some tolerance
		// Truncate if too many lines
		rendered = strings.Join(lines[:height], "\n")
	}

	return rendered, nil
}

// renderThumbnailInlineOptimized renders thumbnail with disk caching for better performance
func renderThumbnailInlineOptimized(imageURL string, width, height int, itemID string) (string, error) {
	if imageURL == "" {
		return "", fmt.Errorf("no image URL provided")
	}

	// Validate dimensions to prevent excessive sizes
	if width > 70 {
		width = 70
	}
	if height > 30 {
		height = 30
	}
	if width < 20 {
		width = 20
	}
	if height < 8 {
		height = 8
	}

	// Check for persistent cache file first
	cacheDir := "/tmp/ytui_thumbs"
	os.MkdirAll(cacheDir, 0755)
	cacheFile := fmt.Sprintf("%s/%s_%dx%d.txt", cacheDir, itemID, width, height)
	
	// Try to read from cache
	if cached, err := os.ReadFile(cacheFile); err == nil {
		return string(cached), nil
	}

	// Download image to temporary file with timeout (reuse existing logic but optimize)
	tmpFile := fmt.Sprintf("/tmp/ytui_img_%s.jpg", itemID)
	
	// Check if image already downloaded
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		client := &http.Client{Timeout: 5 * time.Second} // Reduced timeout
		resp, err := client.Get(imageURL)
		if err != nil {
			return "", fmt.Errorf("failed to download image: %w", err)
		}
		defer resp.Body.Close()

		file, err := os.Create(tmpFile)
		if err != nil {
			return "", fmt.Errorf("failed to create temp file: %w", err)
		}
		defer file.Close()

		// Copy with size limit
		limitedReader := io.LimitReader(resp.Body, 5*1024*1024) // Reduced to 5MB
		_, err = io.Copy(file, limitedReader)
		if err != nil {
			return "", fmt.Errorf("failed to write temp file: %w", err)
		}
	}

	// Render image using go-termimg with forced halfblock renderer
	img, err := termimg.Open(tmpFile)
	if err != nil {
		os.Remove(tmpFile) // Clean up on error
		return "", fmt.Errorf("failed to open image: %w", err)
	}

	// Configure dimensions and explicitly force halfblock renderer
	rendered, err := img.Width(width).Height(height).Protocol(termimg.Halfblocks).Render()
	if err != nil {
		return "", fmt.Errorf("failed to render image: %w", err)
	}

	// Validate the output doesn't exceed expected dimensions
	lines := strings.Split(rendered, "\n")
	if len(lines) > height+2 { // Allow some tolerance
		// Truncate if too many lines
		rendered = strings.Join(lines[:height], "\n")
	}

	// Cache the result to disk for next time
	os.WriteFile(cacheFile, []byte(rendered), 0644)

	return rendered, nil
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
	"s: sort by date",
	"p/Space: play",
	"d: download",
	"/: search",
	"q: quit",
}, " • ")

func (m model) renderHelp() string {
	if len(helpText) > m.width-2 {
		return dimStyle.Render(lipgloss.NewStyle().Width(m.width-2).Render(helpText))
	}
	
	return dimStyle.Render(helpText)
}