package main

import (
	"log"
	"mdp-project-backend/config"
	"mdp-project-backend/routes" // Import paket routes
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Load environment variables dari .env
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found")
	}

	// 2. Inisialisasi konfigurasi
	config.ConnectDB()
	config.SetupOAuth()

	// 3. Buat aplikasi Fiber
	app := fiber.New()

	// 4. Tambahkan Middleware
	app.Use(cors.New())   // Middleware untuk CORS
	app.Use(logger.New()) // Middleware untuk logging request

	// 5. Setup Routes dari paket routes
	routes.SetupRoutes(app)

	// 6. Jalankan Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3033"
	}
	log.Printf("Server starting on port %s...", port)
	log.Fatal(app.Listen(":" + port))
}
