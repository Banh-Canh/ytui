package youtube

import (
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestSaveAndLoadToken(t *testing.T) {
	tmpDir := t.TempDir()
	tokenFile := filepath.Join(tmpDir, "test-token.json")

	token := &oauth2.Token{
		AccessToken:  "access-token",
		TokenType:    "Bearer",
		RefreshToken: "refresh-token",
		Expiry:       time.Now().Add(time.Hour),
	}

	if err := saveToken(tokenFile, token); err != nil {
		t.Fatalf("Failed to save token: %v", err)
	}

	loadedToken, err := loadToken(tokenFile)
	if err != nil {
		t.Fatalf("Failed to load token: %v", err)
	}

	if token.AccessToken != loadedToken.AccessToken {
		t.Errorf("Expected AccessToken %s, got %s", token.AccessToken, loadedToken.AccessToken)
	}
	if token.RefreshToken != loadedToken.RefreshToken {
		t.Errorf("Expected RefreshToken %s, got %s", token.RefreshToken, loadedToken.RefreshToken)
	}
	if !token.Expiry.Equal(loadedToken.Expiry) {
		t.Errorf("Expected Expiry %v, got %v", token.Expiry, loadedToken.Expiry)
	}
}
