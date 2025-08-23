package main

import (
	"context"
	"log"
	"mdp-project-backend/config"
	"mdp-project-backend/models"
	"mdp-project-backend/utils"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found")
	}

	// Connect to database
	config.ConnectDB()

	log.Println("Starting database seeding...")

	// Seed roles first
	roleIDs := seedRoles()

	// Seed admin user
	seedAdminUser(roleIDs["Admin"])

	log.Println("Database seeding completed successfully!")
}

func seedRoles() map[string]primitive.ObjectID {
	collection := config.GetCollection("roles")
	roleIDs := make(map[string]primitive.ObjectID)

	roles := []models.Role{
		{
			BaseModel:   models.BaseModel{ID: primitive.NewObjectID()},
			Name:        "Admin",
			Description: "Administrator dengan akses penuh ke sistem",
			IsActive:    true,
			IsDefault:   true,
			Permissions: []string{
				"user:create", "user:read", "user:update", "user:delete",
				"role:create", "role:read", "role:update", "role:delete",
				"document:create", "document:read", "document:update", "document:delete",
				"admin:access",
			},
		},
		{
			BaseModel:   models.BaseModel{ID: primitive.NewObjectID()},
			Name:        "Manager",
			Description: "Manager dengan akses terbatas untuk mengelola user",
			IsActive:    true,
			IsDefault:   false,
			Permissions: []string{
				"user:read", "user:update",
				"document:create", "document:read", "document:update",
			},
		},
		{
			BaseModel:   models.BaseModel{ID: primitive.NewObjectID()},
			Name:        "User",
			Description: "User biasa dengan akses terbatas",
			IsActive:    true,
			IsDefault:   false,
			Permissions: []string{
				"document:read", "document:create",
			},
		},
		{
			BaseModel:   models.BaseModel{ID: primitive.NewObjectID()},
			Name:        "Developer",
			Description: "Developer dengan akses ke sistem development",
			IsActive:    true,
			IsDefault:   false,
			Permissions: []string{
				"user:read",
				"document:create", "document:read", "document:update",
				"system:debug",
			},
		},
	}

	for _, role := range roles {
		// Check if role already exists
		var existingRole models.Role
		err := collection.FindOne(context.Background(), bson.M{"name": role.Name}).Decode(&existingRole)

		if err == mongo.ErrNoDocuments {
			// Role doesn't exist, create it
			_, err := collection.InsertOne(context.Background(), role)
			if err != nil {
				log.Printf("Error creating role %s: %v", role.Name, err)
				continue
			}
			log.Printf("✅ Created role: %s", role.Name)
			roleIDs[role.Name] = role.ID
		} else if err != nil {
			log.Printf("Error checking role %s: %v", role.Name, err)
		} else {
			log.Printf("⚠️  Role %s already exists, skipping", role.Name)
			roleIDs[role.Name] = existingRole.ID
		}
	}

	return roleIDs
}

func seedAdminUser(adminRoleID primitive.ObjectID) {
	collection := config.GetCollection("users")

	// Check if admin user already exists
	var existingUser models.User
	err := collection.FindOne(context.Background(), bson.M{"username": "admin"}).Decode(&existingUser)

	if err == mongo.ErrNoDocuments {
		// Admin user doesn't exist, create it
		hashedPassword, err := utils.HashPassword("admin123")
		if err != nil {
			log.Printf("Error hashing password: %v", err)
			return
		}

		adminUser := models.User{
			BaseModel: models.BaseModel{ID: primitive.NewObjectID()},
			Username:  "admin",
			Email:     "admin@company.com",
			Password:  hashedPassword,
			FullName:  "System Administrator",
			RoleID:    adminRoleID,
			IsActive:  true,
			Provider:  "local",
			UnitKerja: "IT Department",
		}

		_, err = collection.InsertOne(context.Background(), adminUser)
		if err != nil {
			log.Printf("Error creating admin user: %v", err)
			return
		}

		log.Printf("✅ Created admin user: admin / admin123")
	} else if err != nil {
		log.Printf("Error checking admin user: %v", err)
	} else {
		log.Printf("⚠️  Admin user already exists, skipping")
	}
}
