package main

import (
	"log"
	"mdp-project-backend/config"
	"mdp-project-backend/handlers"
	"mdp-project-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

func main() {
	// Initialize database connection
	config.ConnectDB()

	// Create Fiber app
	app := fiber.New()

	// Add CORS middleware
	app.Use(middleware.CORS())

	// Public routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(200).JSON(fiber.Map{
			"message": "MDP Project Backend API",
			"version": "1.0.0",
		})
	})

	// Auth routes
	auth := app.Group("/api/auth")
	auth.Post("/login", handlers.Login)

	// Protected routes
	api := app.Group("/api", middleware.AuthRequired())
	api.Get("/profile", handlers.GetProfile)
	api.Post("/change-password", handlers.ChangePassword)
	api.Post("/logout", handlers.Logout)

	log.Println("Server starting on port 3033...")
	log.Fatal(app.Listen(":3033"))
}
