package config

import (
	"log"
	"mdp-project-backend/services"
	"os"
)

var codaService *services.CodaAPIService

// InitCodaService initializes the Coda API service
func InitCodaService() {
	baseURL := os.Getenv("CODA_BASE_URL")
	apiToken := os.Getenv("CODA_API_TOKEN")

	if baseURL == "" || apiToken == "" {
		log.Println("Coda service disabled: CODA_BASE_URL or CODA_API_TOKEN not configured")
		return
	}

	codaService = services.NewCodaAPIService(baseURL, apiToken)
	log.Println("Coda API service initialized successfully!")
}

// GetCodaService returns the initialized Coda service
func GetCodaService() *services.CodaAPIService {
	return codaService
}

// IsCodaServiceEnabled returns true if Coda service is configured and available
func IsCodaServiceEnabled() bool {
	return codaService != nil
}
