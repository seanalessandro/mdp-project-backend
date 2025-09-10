package middleware

import (
	"mdp-project-backend/utils"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func AuthRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Authorization header is required",
				"code":    "MISSING_TOKEN",
				"message": "Please provide a valid access token",
			})
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Invalid authorization format. Use 'Bearer <token>'",
				"code":    "INVALID_FORMAT",
				"message": "Authorization header must be in format: Bearer <token>",
			})
		}
		claims, err := utils.ValidateJWT(tokenString)
		if err != nil {
			// Check if it's a token expiration error
			if strings.Contains(err.Error(), "token is expired") {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error":   "Access token has expired",
					"code":    "TOKEN_EXPIRED",
					"message": "Please refresh your access token",
				})
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Invalid or expired token",
				"code":    "INVALID_TOKEN",
				"message": "Please login again",
			})
		}
		c.Locals("user", claims)
		return c.Next()
	}
}

func RoleRequired(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := c.Locals("user").(*utils.Claims)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
		}
		for _, role := range roles {
			if user.Role == role {
				return c.Next()
			}
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Insufficient permissions"})
	}
}
