package main

import (
	"log"
	"mdp-project-backend/handlers"
	"mdp-project-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

func main() {
	// Create Fiber app
	app := fiber.New()

	// Add CORS middleware
	app.Use(middleware.CORS())

	// Public routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(200).JSON(fiber.Map{
			"message": "MDP Project Backend API",
			"version": "1.0.0",
			"status":  "MongoDB connection required for full functionality",
		})
	})

	// Mock login endpoint for demo (remove when MongoDB is available)
	app.Post("/api/auth/login", func(c *fiber.Ctx) error {
		var loginReq map[string]string
		if err := c.BodyParser(&loginReq); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "Invalid request format",
			})
		}

		username := loginReq["username"]
		password := loginReq["password"]

		// Mock authentication for demo
		if (username == "admin" || username == "manager" || username == "user") && password == "password123!" {
			return c.JSON(fiber.Map{
				"token": "demo-jwt-token-" + username,
				"user": fiber.Map{
					"id":       "demo-id-" + username,
					"username": username,
					"role":     username, // role same as username for demo
					"is_active": true,
				},
			})
		}

		return c.Status(401).JSON(fiber.Map{
			"error": "Username or password is incorrect",
		})
	})

	// Auth routes (will work once MongoDB is connected)
	auth := app.Group("/api/auth")
	auth.Post("/login", handlers.Login)

	// Protected routes (will work once MongoDB is connected)
	api := app.Group("/api", middleware.AuthRequired())
	api.Get("/profile", handlers.GetProfile)
	api.Post("/change-password", handlers.ChangePassword)
	api.Post("/logout", handlers.Logout)

	log.Println("Server starting on port 3033...")
	log.Println("Note: Connect MongoDB and run 'go run seed.go' to create test users")
	log.Println("Demo credentials: admin/password123!, manager/password123!, user/password123!")
	log.Fatal(app.Listen(":3033"))
}
