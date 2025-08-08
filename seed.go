package main

import (
	"context"
	"log"
	"mdp-project-backend/config"
	"mdp-project-backend/models"
	"mdp-project-backend/utils"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func main() {
	// Initialize database connection
	config.ConnectDB()

	// Create test users
	users := []models.User{
		{
			Username:  "admin",
			Role:      "admin",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Username:  "manager",
			Role:      "manager",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Username:  "user",
			Role:      "user",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	collection := config.GetCollection("users")

	// Set password for all users (password: "password123!")
	password := "password123!"
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	for _, user := range users {
		user.Password = hashedPassword
		
		// Check if user already exists
		var existingUser models.User
		err := collection.FindOne(context.Background(), bson.M{"username": user.Username}).Decode(&existingUser)
		if err != nil {
			// User doesn't exist, create new one
			_, err := collection.InsertOne(context.Background(), user)
			if err != nil {
				log.Printf("Failed to create user %s: %v", user.Username, err)
			} else {
				log.Printf("Created user: %s with role: %s", user.Username, user.Role)
			}
		} else {
			log.Printf("User %s already exists, skipping...", user.Username)
		}
	}

	log.Println("Seed completed!")
	log.Println("Test credentials:")
	log.Println("  Admin: username=admin, password=password123!")
	log.Println("  Manager: username=manager, password=password123!")
	log.Println("  User: username=user, password=password123!")
}
