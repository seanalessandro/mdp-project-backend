package handlers

import (
	"context"
	"fmt"
	"log"
	"mdp-project-backend/config"
	"mdp-project-backend/models"
	"mdp-project-backend/utils"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CreateDocument membuat dokumen baru, bisa dari template atau kosong
func CreateDocument(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	ownerID, _ := primitive.ObjectIDFromHex(claims.UserID)

	var body struct {
		TemplateID string `json:"templateId"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	templateID, err := primitive.ObjectIDFromHex(body.TemplateID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid template ID"})
	}

	var template models.DocumentTemplate
	err = config.GetCollection("document_templates").FindOne(context.Background(), bson.M{"_id": templateID}).Decode(&template)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Template not found"})
	}

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
		DocNo:    fmt.Sprintf("BRD-%s", newDocID.Hex()[:6]),
		Title:    "Untitled " + template.Name,
		Status:   "Draft",
		Priority: "Medium",
		Version:  1.0,
		Content:  template.Content,
		OwnerID:  ownerID,
	}

	_, err = config.GetCollection("documents").InsertOne(context.Background(), newDoc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return c.Status(400).JSON(fiber.Map{"error": "Gagal membuat dokumen, ID duplikat."})
		}
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create document"})
	}

	logAction := fmt.Sprintf("Membuat dokumen versi %.1f", newDoc.Version)
	utils.LogActivity(ownerID, claims.Username, logAction, c.IP(), string(c.Request().Header.UserAgent()), &newDoc.ID)

	return c.Status(fiber.StatusCreated).JSON(newDoc)
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

	// Log activity
	utils.LogActivity(modifierID, claims.Username, "update_document", c.IP(), string(c.Request().Header.UserAgent()), &docID)

	return c.SendStatus(fiber.StatusNoContent)
}

func helperCreateVersion(docID primitive.ObjectID, changeDesc string) error {
	docCollection := config.GetCollection("documents")
	versionCollection := config.GetCollection("document_versions")

	// 1. Ambil dokumen terkini
	var currentDoc models.Document
	err := docCollection.FindOne(context.Background(), bson.M{"_id": docID}).Decode(&currentDoc)
	if err != nil {
		return err
	}

	// 2. Buat objek versi baru
	newVersion := models.DocumentVersion{
		BaseModel:         models.BaseModel{ID: primitive.NewObjectID(), CreatedOn: time.Now(), CreatedBy: currentDoc.BaseModel.ModifiedBy},
		DocumentID:        currentDoc.ID,
		Version:           currentDoc.Version,
		Content:           currentDoc.Content,
		ChangeDescription: changeDesc,
	}

	// 3. Simpan versi baru
	_, err = versionCollection.InsertOne(context.Background(), newVersion)
	return err
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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Siapkan BSON dasar untuk update
	update := bson.M{
		"$set": bson.M{
			"status":     body.Status,
			"modifiedOn": time.Now(),
			"modifiedBy": &modifierID,
		},
	}

	// Pemicu: saat status diubah menjadi In Review atau Approved
	if body.Status == "In Review" || body.Status == "Approved" {
		// Panggil helper untuk membuat snapshot versi
		err := helperCreateVersion(docID, "Status changed to "+body.Status)
		if err != nil {
			log.Printf("Failed to create document version: %v", err)
		} else {
			// Jika snapshot berhasil dibuat, tambahkan operasi $inc ke BSON update
			update["$inc"] = bson.M{"version": 1.0}
		}
	}

	_, err = config.GetCollection("documents").UpdateOne(context.Background(), bson.M{"_id": docID}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update document status"})
	}

	// Log activity
	utils.LogActivity(modifierID, claims.Username, "update_document_status", c.IP(), string(c.Request().Header.UserAgent()), &docID)
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

func GetDocumentTemplates(c *fiber.Ctx) error {
	collection := config.GetCollection("document_templates")

	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch templates"})
	}

	var templates []models.DocumentTemplate
	if err = cursor.All(context.Background(), &templates); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to decode templates"})
	}

	return c.JSON(templates)
}

func GetDocumentHistory(c *fiber.Ctx) error {
	docID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid document ID"})
	}

	collection := config.GetCollection("activity_logs")
	filter := bson.M{"documentId": docID}
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}}) // Urutkan dari terlama

	cursor, err := collection.Find(context.Background(), filter, opts)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch history"})
	}

	var logs []models.ActivityLog
	if err = cursor.All(context.Background(), &logs); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to decode history"})
	}

	return c.JSON(logs)
}
