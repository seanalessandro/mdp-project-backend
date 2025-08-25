package handlers

import (
	"context"
	"mdp-project-backend/config"
	"mdp-project-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/sergi/go-diff/diffmatchpatch"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetVersionHistory mengambil daftar riwayat versi untuk sebuah dokumen.
func GetVersionHistory(c *fiber.Ctx) error {
	docID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid document ID"})
	}

	collection := config.GetCollection("document_versions")
	filter := bson.M{"documentId": docID}
	opts := options.Find().SetSort(bson.D{{Key: "createdOn", Value: -1}}) // Urutkan dari yang terbaru

	cursor, err := collection.Find(context.Background(), filter, opts)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch version history"})
	}

	var versions []models.DocumentVersion
	if err = cursor.All(context.Background(), &versions); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to decode versions"})
	}

	return c.JSON(versions)
}

// CompareVersions membandingkan dua versi dokumen dan mengembalikan perbedaannya.
func CompareVersions(c *fiber.Ctx) error {
	fromVersionID, err := primitive.ObjectIDFromHex(c.Query("from"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid 'from' version ID"})
	}
	toVersionID, err := primitive.ObjectIDFromHex(c.Query("to"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid 'to' version ID"})
	}

	collection := config.GetCollection("document_versions")
	var fromVersion, toVersion models.DocumentVersion

	// Ambil data dua versi dari database
	collection.FindOne(context.Background(), bson.M{"_id": fromVersionID}).Decode(&fromVersion)
	collection.FindOne(context.Background(), bson.M{"_id": toVersionID}).Decode(&toVersion)

	if fromVersion.ID.IsZero() || toVersion.ID.IsZero() {
		return c.Status(404).JSON(fiber.Map{"error": "One or both versions not found"})
	}

	// Lakukan perbandingan (diff)
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(fromVersion.Content, toVersion.Content, true)

	// Kirim hasil diff ke frontend
	return c.JSON(diffs)
}
