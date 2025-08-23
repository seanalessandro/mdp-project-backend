package config

import (
	"log"
	"mdp-project-backend/services"
)

// Global email service instance
// Similar to @Autowired in Spring Boot or Service Container in Laravel
var EmailService *services.EmailService

// InitializeEmailService initializes the email service
// Similar to @PostConstruct in Spring Boot or Service Provider in Laravel
func InitializeEmailService() {
	EmailService = services.NewEmailService()

	// Validate email configuration
	if err := EmailService.ValidateEmailConfig(); err != nil {
		log.Printf("Email service configuration warning: %v", err)
		log.Println("Email functionality will be disabled. Please check your environment variables.")
		EmailService = nil // Disable email service if not properly configured
	} else {
		log.Println("Email service initialized successfully!")
	}
}

// GetEmailService returns the global email service instance
// Similar to @Autowired getter or Service facade in Laravel
func GetEmailService() *services.EmailService {
	return EmailService
}

// IsEmailServiceEnabled checks if email service is available
func IsEmailServiceEnabled() bool {
	return EmailService != nil
}
