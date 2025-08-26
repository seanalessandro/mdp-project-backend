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

	// User Management Routes - FR-5.2.2
	adminApi.Get("/users", handlers.GetUsers)                              // Get all users with pagination/filtering
	adminApi.Get("/users/:id", handlers.GetUser)                           // Get single user
	adminApi.Post("/users", handlers.CreateUser)                           // FR-5.2.2.1: Create new user account
	adminApi.Put("/users/:id", handlers.UpdateUser)                        // FR-5.2.2.3: Update user role/unit
	adminApi.Patch("/users/:id/role", handlers.UpdateUserRole)             // FR-5.2.2.3: Update user role specifically
	adminApi.Patch("/users/:id/status", handlers.UpdateUser)               // FR-5.2.2.2: Toggle user active/inactive status
	adminApi.Post("/users/:id/reset-password", handlers.ResetUserPassword) // FR-5.2.2.5: Reset user password
	// Note: FR-5.2.2.4 - No delete route as per requirement (only deactivate)

	//CRUD Master Role
	adminApi.Post("/roles", handlers.CreateRole)       // FR-5.2.1: Membuat role baru
	adminApi.Get("/roles", handlers.GetAllRoles)       // Membaca semua role
	adminApi.Get("/roles/:id", handlers.GetRoleByID)   // Membaca satu role
	adminApi.Put("/roles/:id", handlers.UpdateRole)    // FR-5.2.3: Mengedit role
	adminApi.Delete("/roles/:id", handlers.DeleteRole) // FR-5.2.3: Menghapus role
	adminApi.Patch("/roles/:id/status", handlers.ToggleRoleStatus)

	adminApi.Get("/permissions", handlers.GetAllPermissions)

	// CRUD Dokumen
	docs := api.Group("/documents", middleware.AuthRequired())
	docs.Post("/", handlers.CreateDocument)
	docs.Get("/", handlers.GetMyDocuments)
	docs.Get("/:id", handlers.GetDocumentByID)
	docs.Put("/:id", handlers.UpdateDocument)
	docs.Patch("/:id/status", handlers.UpdateDocumentStatus)
	docs.Delete("/:id", handlers.DeleteDocument)
	docs.Get("/:id/comments", handlers.GetCommentsForDocument)
	docs.Post("/:id/comments", handlers.CreateComment)

	// PDF Export Routes - FR-5.3.4: Export document to PDF
	docs.Get("/:id/export/pdf", handlers.ExportDocumentToPDF)    // Download PDF
	docs.Get("/:id/preview/pdf", handlers.GetDocumentPDFPreview) // Preview PDF inline

	api.Post("/upload/image", middleware.AuthRequired(), handlers.UploadImage)
	api.Get("/templates", middleware.AuthRequired(), handlers.GetTemplates)
	api.Get("/document-templates", handlers.GetDocumentTemplates)
	api.Get("/documents/:id/versions", handlers.GetVersionHistory)
	api.Get("/documents/:id/versions/compare", handlers.CompareVersions)
	api.Get("/documents/:id/history", handlers.GetDocumentHistory)

}
