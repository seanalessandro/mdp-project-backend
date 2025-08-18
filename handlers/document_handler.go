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

// CreateDocument membuat dokumen baru, bisa dari template atau kosong
func CreateDocument(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	ownerID, _ := primitive.ObjectIDFromHex(claims.UserID)

	var body struct {
		Title    string `json:"title"`
		DocNo    string `json:"docNo"`
		Priority string `json:"priority"`
		Content  string `json:"content"` // Menerima konten opsional dari template
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if body.Title == "" || body.DocNo == "" || body.Priority == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Title, DocNo, and Priority are required"})
	}

	// --- PERBAIKAN LOGIKA DI SINI ---
	// 1. Deklarasikan initialContent dengan nilai default
	initialContent := `{"type":"doc","content":[{"type":"paragraph"}]}`

	// 2. Jika ada konten dari template, timpa nilainya
	if body.Content != "" {
		initialContent = body.Content
	}
	// ---------------------------------

	now := time.Now()
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
		Content:  initialContent, // Gunakan variabel yang sudah benar
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

	var createdDoc models.Document
	err = collection.FindOne(context.Background(), bson.M{"_id": newDocID}).Decode(&createdDoc)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve created document"})
	}

	return c.Status(fiber.StatusCreated).JSON(createdDoc)
}

// GetDocumentByID mengambil satu dokumen
func GetDocumentByID(c *fiber.Ctx) error {
	docID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid document ID"})
	}

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

// UpdateDocument menyimpan perubahan pada judul, konten, dan nomor dokumen
func UpdateDocument(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	modifierID, _ := primitive.ObjectIDFromHex(claims.UserID)

	docID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid document ID"})
	}

	// --- PERBAIKAN 1: Tambahkan Priority ke body request ---
	var body struct {
		Title    string `json:"title"`
		Content  string `json:"content"`
		DocNo    string `json:"docNo"`
		Priority string `json:"priority"` // Tambahkan ini
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	collection := config.GetCollection("documents")
	// --- PERBAIKAN 2: Tambahkan Priority ke BSON update ---
	update := bson.M{
		"$set": bson.M{
			"title":      body.Title,
			"content":    body.Content,
			"docNo":      body.DocNo,
			"priority":   body.Priority, // Tambahkan ini
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

// UpdateDocumentStatus hanya mengubah status dokumen
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
