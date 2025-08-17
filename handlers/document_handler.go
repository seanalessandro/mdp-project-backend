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
		Title string `json:"title"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	now := time.Now()

	initialContent := `{"type":"doc","content":[{"type":"paragraph"}]}`

	newDoc := models.Document{
		BaseModel: models.BaseModel{
			ID:         primitive.NewObjectID(),
			CreatedOn:  now,
			CreatedBy:  &ownerID,
			ModifiedOn: now,
			ModifiedBy: &ownerID,
		},
		Title:   body.Title,
		Content: initialContent,
		OwnerID: ownerID,
	}

	collection := config.GetCollection("documents")
	_, err := collection.InsertOne(context.Background(), newDoc)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create document"})
	}

	return c.Status(fiber.StatusCreated).JSON(newDoc)
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
