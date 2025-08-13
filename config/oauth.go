package config

import (
	"log"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var GoogleOAuthConfig *oauth2.Config

func SetupOAuth() {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	baseURL := os.Getenv("APP_BASE_URL")

	if clientID == "" || clientSecret == "" || baseURL == "" {
		log.Println("Warning: Google OAuth credentials not set. Google login will not work.")
		return
	}

	GoogleOAuthConfig = &oauth2.Config{
		RedirectURL:  baseURL + "/api/auth/google/callback",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}
