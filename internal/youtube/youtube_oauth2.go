package youtube

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/Banh-Canh/ytui/internal/utils"
)

// Save the token to a file
func saveToken(filename string, token *oauth2.Token) error {
	utils.Logger.Debug("Saving OAuth2 token.", zap.String("filename", filename))

	// Create all necessary directories
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		utils.Logger.Error("Failed to create directories.", zap.String("directory", dir), zap.Error(err))
		return err
	}
	utils.Logger.Info("Directories created.", zap.String("directory", dir))

	file, err := os.Create(filename)
	if err != nil {
		utils.Logger.Error("Failed to create file.", zap.String("filename", filename), zap.Error(err))
		return err
	}
	defer file.Close()
	utils.Logger.Debug("File created.", zap.String("filename", filename))

	// Encode the token into the file
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(token); err != nil {
		utils.Logger.Error("Failed to encode token.", zap.String("filename", filename), zap.Error(err))
		return err
	}
	utils.Logger.Info("Token saved successfully.", zap.String("filename", filename))

	return nil
}

func loadToken(filename string) (*oauth2.Token, error) {
	utils.Logger.Debug("Loading OAuth2 token.", zap.String("filename", filename))

	file, err := os.Open(filename)
	if err != nil {
		utils.Logger.Error("Failed to open file.", zap.String("filename", filename), zap.Error(err))
		return nil, err
	}
	defer file.Close()
	utils.Logger.Debug("File opened.", zap.String("filename", filename))

	token := &oauth2.Token{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(token); err != nil {
		utils.Logger.Error("Failed to decode token.", zap.String("filename", filename), zap.Error(err))
		return nil, err
	}
	utils.Logger.Info("Token loaded successfully.", zap.String("filename", filename))

	return token, nil
}

// isTokenExpired checks if the token is expired.
func isTokenExpired(token *oauth2.Token) bool {
	utils.Logger.Debug("Checking if token is expired.")
	if token == nil || !token.Valid() || time.Now().After(token.Expiry) {
		utils.Logger.Info("Token is expired or invalid.")
		return true
	}
	utils.Logger.Debug("Token is valid.")
	return false
}

// NewYouTubeAPI initializes the YouTube API by handling the OAuth2 flow.
func NewYouTubeAPI(clientID, clientSecret string) (chan *YouTubeAPI, error) {
	utils.Logger.Info("Initializing YouTube API.", zap.String("client_id", clientID))

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  "http://localhost:8080/oauth2callback",
		Scopes:       []string{"https://www.googleapis.com/auth/youtube.readonly"},
		Endpoint:     google.Endpoint,
	}

	tokenFile := getTokenFilePath()
	utils.Logger.Debug("Token file path determined.", zap.String("token_file", tokenFile))

	apiChan := make(chan *YouTubeAPI)
	var startOAuthErr error

	go func() {
		defer close(apiChan)
		utils.Logger.Debug("Loading OAuth2 token.")
		token, err := loadToken(tokenFile)
		if err != nil || isTokenExpired(token) {
			// If token loading fails or token is expired, start OAuth flow
			if err != nil {
				utils.Logger.Error("Failed to load token.", zap.String("token_file", tokenFile), zap.Error(err))
			} else {
				utils.Logger.Info("Token is expired. Starting OAuth2 flow.")
			}
			utils.Logger.Debug("Starting OAuth2 flow.")
			if startOAuthErr = startOAuthFlow(config, apiChan, tokenFile); startOAuthErr != nil {
				utils.Logger.Error("OAuth2 flow failed.", zap.Error(startOAuthErr))
				return
			}
		} else {
			utils.Logger.Debug("Token loaded successfully.")
			client := config.Client(context.Background(), token)
			utils.Logger.Info("YouTube API client created.")
			apiChan <- &YouTubeAPI{Client: client}
		}
	}()
	return apiChan, startOAuthErr
}

// getTokenFilePath constructs the token file path in the user's home directory.
func getTokenFilePath() string {
	utils.Logger.Debug("Determining token file path.")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		utils.Logger.Fatal("Error getting user home directory.", zap.Error(err))
	}
	tokenFile := filepath.Join(homeDir, ".config", "ytui", "credentials.json")
	utils.Logger.Info("Token file path determined.", zap.String("token_file", tokenFile))
	return tokenFile
}

func startOAuthFlow(config *oauth2.Config, apiChan chan *YouTubeAPI, tokenFile string) error {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	utils.Logger.Debug("Opening browser to initiate OAuth2 login.", zap.String("auth_url", authURL))

	err := exec.Command("xdg-open", authURL).Start()
	if err != nil {
		utils.Logger.Error("Failed to open browser.", zap.Error(err))
		return err
	}
	utils.Logger.Debug("Browser opened successfully for OAuth2 login.")

	if err := startOAuthServer(config, apiChan, tokenFile); err != nil {
		utils.Logger.Error("Failed to start OAuth2 server.", zap.Error(err))
		return err
	}
	utils.Logger.Info("OAuth2 server started successfully.")

	return nil
}

// startOAuthServer starts the local server to handle OAuth2 callback.
func startOAuthServer(config *oauth2.Config, apiChan chan *YouTubeAPI, tokenFile string) error {
	http.HandleFunc("/oauth2callback", func(w http.ResponseWriter, r *http.Request) {
		handleOAuthCallback(config, w, r, apiChan, tokenFile)
	})

	utils.Logger.Info("Starting server to handle OAuth2 callback.", zap.String("address", ":8080"))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		utils.Logger.Fatal("Failed to start server.", zap.Error(err))
		return err
	}
	return nil
}

func handleOAuthCallback(config *oauth2.Config, w http.ResponseWriter, r *http.Request, apiChan chan *YouTubeAPI, tokenFile string) {
	// Retrieve the authorization code from the request
	code := r.URL.Query().Get("code")
	if code == "" {
		utils.Logger.Warn("Authorization code not found in request.")
		http.Error(w, "Authorization code not found", http.StatusBadRequest)
		return
	}

	utils.Logger.Debug("Exchanging authorization code for token.", zap.String("code", code))

	// Exchange the authorization code for an OAuth token
	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		utils.Logger.Error("Failed to exchange authorization code for token.", zap.Error(err))
		http.Error(w, "Failed to exchange authorization code for token", http.StatusInternalServerError)
		return
	}

	utils.Logger.Info("Token exchanged successfully. Saving token to file.")
	// Save the token to a file for future use
	if err := saveToken(tokenFile, token); err != nil {
		utils.Logger.Error("Failed to save token.", zap.Error(err))
		http.Error(w, "Failed to save token", http.StatusInternalServerError)
		return
	}

	// Send the client (with the token) through the channel for use in the application
	utils.Logger.Info("OAuth2 authorization successful. Sending client to API channel.")
	apiChan <- &YouTubeAPI{Client: config.Client(context.Background(), token)}
	close(apiChan)

	// Provide the user with feedback and instruct them to return to the application
	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(`
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Authentication Complete</title>
		</head>
		<body>
			<h1>Authentication Successful</h1>
			<p>You have successfully authenticated with your Google account.</p>
			<p>You can now return to the application to continue.</p>
		</body>
		</html>
	`))
}
