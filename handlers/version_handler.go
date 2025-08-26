package handlers

import (
	"context"
	"encoding/json" // Pastikan package ini di-import
	"mdp-project-backend/config"
	"mdp-project-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/sergi/go-diff/diffmatchpatch"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// FUNGSI HELPER BARU UNTUK EKSTRAK TEKS DARI JSON TIPTAP
func extractTextFromTiptapJSON(jsonString string) string {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonString), &data); err != nil {
		return jsonString // Kembalikan string asli jika gagal parse
	}

	var textContent string
	if content, ok := data["content"].([]interface{}); ok {
		for _, node := range content {
			if nodeMap, ok := node.(map[string]interface{}); ok {
				if nodeContent, ok := nodeMap["content"].([]interface{}); ok {
					for _, textNode := range nodeContent {
						if textNodeMap, ok := textNode.(map[string]interface{}); ok {
							if text, ok := textNodeMap["text"].(string); ok {
								textContent += text
							}
						}
					}
				}
				// Tambahkan baris baru setelah setiap paragraf/heading
				textContent += "\n"
			}
		}
	}
	return textContent
}

// GetVersionHistory mengambil daftar riwayat versi untuk sebuah dokumen.
func GetVersionHistory(c *fiber.Ctx) error {
	docID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid document ID"})
	}

	collection := config.GetCollection("document_versions")
	filter := bson.M{"documentId": docID}
	opts := options.Find().SetSort(bson.D{{Key: "createdOn", Value: -1}})

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

	collection.FindOne(context.Background(), bson.M{"_id": fromVersionID}).Decode(&fromVersion)
	collection.FindOne(context.Background(), bson.M{"_id": toVersionID}).Decode(&toVersion)

	if fromVersion.ID.IsZero() || toVersion.ID.IsZero() {
		return c.Status(404).JSON(fiber.Map{"error": "One or both versions not found"})
	}

	// Ekstrak teks dari konten JSON sebelum di-diff
	fromText := extractTextFromTiptapJSON(fromVersion.Content)
	toText := extractTextFromTiptapJSON(toVersion.Content)

	// Lakukan perbandingan (diff) pada teks yang sudah diekstrak
	dmp := diffmatchpatch.New()
	diffsRaw := dmp.DiffMain(fromText, toText, true)

	// Ubah format output menjadi array sederhana agar mudah dibaca frontend
	var diffsSimple [][2]interface{}
	for _, diff := range diffsRaw {
		diffsSimple = append(diffsSimple, [2]interface{}{diff.Type, diff.Text})
	}

	return c.JSON(diffsSimple)
}
