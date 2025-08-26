package handlers

import (
	"context"
	"encoding/json" // Pastikan package ini di-import
	"mdp-project-backend/config"
	"mdp-project-backend/models"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo" // Tambahkan import ini
)

type VersionWithUsername struct {
	models.DocumentVersion `bson:",inline"`
	CreatedByUsername      string `json:"createdByUsername" bson:"createdByUsername"`
}

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

	// Pipa aggregation
	pipeline := mongo.Pipeline{
		// Filter versi berdasarkan documentID
		{{"$match", bson.M{"documentId": docID}}},

		// Lookup untuk mengambil data user dari koleksi 'users'
		{{"$lookup", bson.M{
			"from":         "users",
			"localField":   "createdBy",
			"foreignField": "_id",
			"as":           "creator",
		}}},

		// Flatten array 'creator' menjadi objek tunggal
		{{"$unwind", bson.M{"path": "$creator", "preserveNullAndEmptyArrays": true}}},

		// Proyeksi untuk memformat output
		{{"$project", bson.M{
			"_id":               "$_id",
			"documentId":        "$documentId",
			"version":           "$version",
			"content":           "$content",
			"createdOn":         "$createdOn",
			"changeDescription": "$changeDescription",
			"createdBy":         "$createdBy",        // Simpan ID aslinya
			"createdByUsername": "$creator.username", // Ambil field username dari hasil lookup
		}}},

		// Urutkan berdasarkan tanggal
		{{"$sort", bson.M{"createdOn": -1}}},
	}

	cursor, err := collection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch version history with username"})
	}
	var versions []VersionWithUsername

	if err = cursor.All(context.Background(), &versions); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode versions"})
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

	// Ambil kedua versi
	collection.FindOne(context.Background(), bson.M{"_id": fromVersionID}).Decode(&fromVersion)
	collection.FindOne(context.Background(), bson.M{"_id": toVersionID}).Decode(&toVersion)

	if fromVersion.ID.IsZero() || toVersion.ID.IsZero() {
		return c.Status(404).JSON(fiber.Map{"error": "One or both versions not found"})
	}

	// Ubah JSON string menjadi objek Go untuk membandingkan secara struktural
	// Namun, Go tidak punya library diff yang mudah untuk struktur JSON.
	// Solusi terbaik adalah membandingkan teks yang diformat khusus.
	// Di sini, kita akan tetap mengirimkan kedua JSON string ke frontend
	// dan membiarkan frontend yang menanganinya.

	// Perbaikan: HAPUS LOGIKA DIFF DI BACKEND
	// Cukup kembalikan konten dari kedua versi
	return c.JSON(fiber.Map{
		"fromContent": fromVersion.Content,
		"toContent":   toVersion.Content,
	})
}
