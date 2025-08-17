package handlers

import (
	"context"
	"mdp-project-backend/config"
	"mdp-project-backend/models"
	"mdp-project-backend/utils"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateDocument membuat dokumen baru yang kosong

func CreateDocument(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	ownerID, _ := primitive.ObjectIDFromHex(claims.UserID)

	var body struct {
		Title    string `json:"title"`
		DocNo    string `json:"docNo"`
		Priority string `json:"priority"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if body.Title == "" || body.DocNo == "" || body.Priority == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Title, DocNo, and Priority are required"})
	}

	now := time.Now()
	initialContent := `{"type":"doc","content":[{"type":"paragraph"}]}`
	newDocID := primitive.NewObjectID()

	newDoc := models.Document{
		BaseModel: models.BaseModel{
			ID:         newDocID,
			CreatedOn:  now,
			CreatedBy:  &ownerID,
			ModifiedOn: now,
			ModifiedBy: &ownerID,
		},
		Title:    body.Title,
		Content:  initialContent,
		OwnerID:  ownerID,
		Status:   "Draft",
		DocNo:    body.DocNo,
		Version:  "1.0",
		Priority: body.Priority,
	}

	collection := config.GetCollection("documents")
	_, err := collection.InsertOne(context.Background(), newDoc)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create document"})
	}

	// --- PERBAIKAN DI SINI ---
	// Jangan langsung kembalikan 'newDoc'. Ambil data yang baru dibuat dari database.
	var createdDoc models.Document
	err = collection.FindOne(context.Background(), bson.M{"_id": newDocID}).Decode(&createdDoc)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve created document"})
	}

	// Kembalikan 'createdDoc' yang datanya sudah pasti dari database
	return c.Status(fiber.StatusCreated).JSON(createdDoc)
}

// GetDocumentByID mengambil satu dokumen
func GetDocumentByID(c *fiber.Ctx) error {
	docID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid document ID"})
	}
	// TODO: Tambahkan pengecekan hak akses (apakah user ini boleh melihat dokumen ini)

	collection := config.GetCollection("documents")
	var doc models.Document
	err = collection.FindOne(context.Background(), bson.M{"_id": docID}).Decode(&doc)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Document not found"})
	}
	return c.JSON(doc)
}

// GetMyDocuments mengambil semua dokumen milik user yang login
func GetMyDocuments(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	ownerID, _ := primitive.ObjectIDFromHex(claims.UserID)

	collection := config.GetCollection("documents")
	filter := bson.M{"ownerId": ownerID}

	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch documents"})
	}

	var documents []models.Document
	if err = cursor.All(context.Background(), &documents); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode documents"})
	}

	return c.JSON(documents)
}

// UpdateDocument menyimpan perubahan pada judul dan konten dokumen
func UpdateDocument(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	modifierID, _ := primitive.ObjectIDFromHex(claims.UserID)

	docID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid document ID"})
	}

	// --- PERBAIKAN 1: Tambahkan DocNo ke body request ---
	var body struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		DocNo   string `json:"docNo"` // Tambahkan ini
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	collection := config.GetCollection("documents")
	// --- PERBAIKAN 2: Tambahkan DocNo ke BSON update ---
	update := bson.M{
		"$set": bson.M{
			"title":      body.Title,
			"content":    body.Content,
			"docNo":      body.DocNo, // Tambahkan ini
			"modifiedOn": time.Now(),
			"modifiedBy": &modifierID,
		},
	}

	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": docID}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update document"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// UpdateDocument menyimpan perubahan pada dokumen
func UpdateDocumentStatus(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	modifierID, _ := primitive.ObjectIDFromHex(claims.UserID)
	docID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid document ID"})
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil || body.Status == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body, 'status' is required"})
	}

	update := bson.M{
		"$set": bson.M{
			"status":     body.Status,
			"modifiedOn": time.Now(),
			"modifiedBy": &modifierID,
		},
	}

	collection := config.GetCollection("documents")
	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": docID}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update document status"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Document status updated successfully"})
}

// DeleteDocument menghapus dokumen
func DeleteDocument(c *fiber.Ctx) error {
	docID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid document ID"})
	}

	collection := config.GetCollection("documents")
	result, err := collection.DeleteOne(context.Background(), bson.M{"_id": docID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete document"})
	}

	if result.DeletedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Document not found"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
