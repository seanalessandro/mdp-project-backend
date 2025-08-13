package middleware

import (
	"context"
	"log"
	"mdp-project-backend/config"
	"mdp-project-backend/models"
	"mdp-project-backend/utils"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
)

func PermissionRequired(requiredPerm string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals("user").(*utils.Claims)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Invalid user claims"})
		}

		// Ambil role dari database berdasarkan nama role di token
		roleCollection := config.GetCollection("roles")
		var role models.Role
		err := roleCollection.FindOne(context.Background(), bson.M{"name": claims.Role}).Decode(&role)
		if err != nil {
			log.Printf("Permission check failed: could not find role '%s'", claims.Role)
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied. Role not found."})
		}

		// Cek jika role memiliki permission '*' (semua akses)
		if containsString(role.Permissions, "*") {
			return c.Next()
		}

		// Cek jika role memiliki permission yang dibutuhkan
		if !containsString(role.Permissions, requiredPerm) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":             "You do not have the required permission for this action.",
				"permission_needed": requiredPerm,
			})
		}

		return c.Next()
	}
}

// helper function untuk mengecek keberadaan string dalam slice
func containsString(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}
