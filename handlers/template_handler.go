package handlers

import (
	"context"
	"mdp-project-backend/config"
	"mdp-project-backend/models"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
)

// GetTemplates mengambil semua template dokumen yang tersedia
func GetTemplates(c *fiber.Ctx) error {
	collection := config.GetCollection("templates")
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch templates"})
	}

	var templates []models.DocumentTemplate
	if err = cursor.All(context.Background(), &templates); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode templates"})
	}

	return c.JSON(templates)
}
