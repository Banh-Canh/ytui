package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Save the token to a file
func saveToken(filename string, token *oauth2.Token) error {
	// Create all necessary directories
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Encode the token into the file
	encoder := json.NewEncoder(file)
	return encoder.Encode(token)
}

// Load the token from a file
func loadToken(filename string) (*oauth2.Token, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	token := &oauth2.Token{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(token)
	if err != nil {
		return nil, err
	}
	return token, nil
}

// NewYouTubeAPI initializes the YouTube API by handling the OAuth2 flow.
func NewYouTubeAPI(clientID, clientSecret string) (chan *YouTubeAPI, error) {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  "http://localhost:8080/oauth2callback",
		Scopes:       []string{"https://www.googleapis.com/auth/youtube.readonly"},
		Endpoint:     google.Endpoint,
	}

	tokenFile := getTokenFilePath()
	apiChan := make(chan *YouTubeAPI)
	var startOAuthErr error

	go func() {
		defer close(apiChan)
		token, err := loadToken(tokenFile)
		if err != nil {
			if startOAuthErr = startOAuthFlow(config, apiChan, tokenFile); startOAuthErr != nil {
				return
			}
		} else {
			client := config.Client(context.Background(), token)
			apiChan <- &YouTubeAPI{Client: client}
		}
	}()

	return apiChan, startOAuthErr
}

// getTokenFilePath constructs the token file path in the user's home directory.
func getTokenFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting user home directory: %v", err)
	}
	tokenFile := filepath.Join(homeDir, ".config", "ytui", "credentials.json")
	return tokenFile
}

func startOAuthFlow(config *oauth2.Config, apiChan chan *YouTubeAPI, tokenFile string) error {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	log.Println("Opening browser to initiate OAuth2 login...")

	err := exec.Command("xdg-open", authURL).Start()
	if err != nil {
		log.Fatalf("Failed to open browser: %v", err)
		return err
	}

	if err := startOAuthServer(config, apiChan, tokenFile); err != nil {
		return err
	}
	return nil
}

// startOAuthServer starts the local server to handle OAuth2 callback.
func startOAuthServer(config *oauth2.Config, apiChan chan *YouTubeAPI, tokenFile string) error {
	http.HandleFunc("/oauth2callback", func(w http.ResponseWriter, r *http.Request) {
		handleOAuthCallback(config, w, r, apiChan, tokenFile)
	})

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
		return err
	}
	return nil
}

func handleOAuthCallback(config *oauth2.Config, w http.ResponseWriter, r *http.Request, apiChan chan *YouTubeAPI, tokenFile string) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Authorization code not found", http.StatusBadRequest)
		return
	}

	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	if err := saveToken(tokenFile, token); err != nil {
		log.Printf("Failed to save token: %v", err)
	}

	fmt.Fprintln(w, "Authorization successful! You can close this window.")
	log.Println("OAuth2 authorization successful!")
	apiChan <- &YouTubeAPI{Client: config.Client(context.Background(), token)}
	close(apiChan)
}
