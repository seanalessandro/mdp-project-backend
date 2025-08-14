package routes

import (
	"mdp-project-backend/handlers"
	"mdp-project-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupRoutes mendefinisikan semua rute untuk aplikasi.
func SetupRoutes(app *fiber.App) {
	// Public route
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Welcome to MDP Project Backend API",
			"version": "1.2.0 (with separated routes)",
		})
	})

	// Grup untuk otentikasi
	auth := app.Group("/api/auth")

	// Otentikasi Lokal (username/password)
	auth.Post("/login", handlers.Login)

	// Otentikasi via Google OAuth2
	auth.Get("/google/login", handlers.GoogleLogin)
	auth.Get("/google/callback", handlers.GoogleCallback)

	auth.Get("/reset-admin-password", handlers.ResetAdminPassword) // Route sementara untuk reset

	// ==================================================================
	// Grup untuk API yang memerlukan otentikasi (Protected Routes)
	// ==================================================================
	api := app.Group("/api", middleware.AuthRequired())

	// Rute Profil Pengguna
	api.Get("/profile", handlers.GetProfile)
	api.Post("/change-password", handlers.ChangePassword)
	api.Post("/logout", handlers.Logout) // Meskipun hanya di sisi client, endpoint ini bisa untuk logging

	// Contoh rute khusus Admin
	adminApi := api.Group("/admin", middleware.RoleRequired("admin"))
	adminApi.Post("/users", handlers.AdminCreateUser)

	//CRUD Master Role
	adminApi.Post("/roles", handlers.CreateRole)       // FR-5.2.1: Membuat role baru
	adminApi.Get("/roles", handlers.GetAllRoles)       // Membaca semua role
	adminApi.Get("/roles/:id", handlers.GetRoleByID)   // Membaca satu role
	adminApi.Put("/roles/:id", handlers.UpdateRole)    // FR-5.2.3: Mengedit role
	adminApi.Delete("/roles/:id", handlers.DeleteRole) // FR-5.2.3: Menghapus role
	adminApi.Patch("/roles/:id/status", handlers.ToggleRoleStatus)

	adminApi.Get("/permissions", handlers.GetAllPermissions)

}
