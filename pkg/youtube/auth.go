package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/oauth2"
)

// AuthService handles OAuth2 authentication
type AuthService struct {
	client *Client
}

// NewAuthService creates a new auth service
func (c *Client) Auth() *AuthService {
	return &AuthService{client: c}
}

// AuthenticateAsync performs OAuth2 authentication asynchronously
func (a *AuthService) AuthenticateAsync() (chan *http.Client, error) {
	tokenFile := a.getTokenFilePath()
	apiChan := make(chan *http.Client)

	go func() {
		defer close(apiChan)
		
		token, err := a.loadToken(tokenFile)
		if err != nil || a.isTokenExpired(token) {
			if err := a.startOAuthFlow(apiChan, tokenFile); err != nil {
				return
			}
		} else {
			// Check if token needs refreshing
			if !token.Valid() && token.RefreshToken != "" {
				refreshedToken, refreshErr := a.refreshToken(token)
				if refreshErr != nil {
					if err := a.startOAuthFlow(apiChan, tokenFile); err != nil {
						return
					}
					return
				}
				
				// Save the refreshed token
				if saveErr := a.saveToken(tokenFile, refreshedToken); saveErr != nil {
					// Log error but continue
				}
				token = refreshedToken
			}
			
			client := a.client.oauth2Config.Client(context.Background(), token)
			apiChan <- client
		}
	}()
	
	return apiChan, nil
}

// Authenticate performs OAuth2 authentication synchronously
func (a *AuthService) Authenticate() (*http.Client, error) {
	clientChan, err := a.AuthenticateAsync()
	if err != nil {
		return nil, err
	}
	
	client := <-clientChan
	if client == nil {
		return nil, fmt.Errorf("authentication failed")
	}
	
	return client, nil
}

func (a *AuthService) saveToken(filename string, token *oauth2.Token) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(token)
}

func (a *AuthService) loadToken(filename string) (*oauth2.Token, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	token := &oauth2.Token{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(token); err != nil {
		return nil, err
	}

	return token, nil
}

func (a *AuthService) isTokenExpired(token *oauth2.Token) bool {
	if token == nil {
		return true
	}
	
	if token.Valid() {
		return false
	}
	
	if token.RefreshToken != "" {
		return false
	}
	
	return true
}

func (a *AuthService) refreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	if token.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}
	
	tokenSource := a.client.oauth2Config.TokenSource(context.Background(), token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, err
	}
	
	return newToken, nil
}

func (a *AuthService) getTokenFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".config", "ytui", "credentials.json")
}

func (a *AuthService) startOAuthFlow(apiChan chan *http.Client, tokenFile string) error {
	authURL := a.client.oauth2Config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	
	err := exec.Command("xdg-open", authURL).Start()
	if err != nil {
		return err
	}

	return a.startOAuthServer(apiChan, tokenFile)
}

func (a *AuthService) startOAuthServer(apiChan chan *http.Client, tokenFile string) error {
	http.HandleFunc("/oauth2callback", func(w http.ResponseWriter, r *http.Request) {
		a.handleOAuthCallback(w, r, apiChan, tokenFile)
	})

	return http.ListenAndServe(":8080", nil)
}

func (a *AuthService) handleOAuthCallback(w http.ResponseWriter, r *http.Request, apiChan chan *http.Client, tokenFile string) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Authorization code not found", http.StatusBadRequest)
		return
	}

	token, err := a.client.oauth2Config.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Failed to exchange authorization code for token", http.StatusInternalServerError)
		return
	}

	if err := a.saveToken(tokenFile, token); err != nil {
		http.Error(w, "Failed to save token", http.StatusInternalServerError)
		return
	}

	client := a.client.oauth2Config.Client(context.Background(), token)
	apiChan <- client

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
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