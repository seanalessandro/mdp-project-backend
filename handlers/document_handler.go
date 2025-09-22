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

	// Initialize approval workflow with empty arrays
	// Will be populated when document is submitted for review
	newDoc.CurrentApprovalLevel = 0
	newDoc.Approvals = []models.ApprovalLevel{}

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
		log.Printf("Failed to decode documents into model: %v", err)
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

	update := bson.M{
		"$inc": bson.M{"version": 1.0},
	}
	_, err = docCollection.UpdateOne(context.Background(), bson.M{"_id": docID}, update)
	if err != nil {
		return fmt.Errorf("failed to update document version: %w", err)
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

	collection := config.GetCollection("documents")

	// First, get the current document to retrieve owner information
	var currentDoc models.Document
	err = collection.FindOne(context.Background(), bson.M{"_id": docID}).Decode(&currentDoc)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Document not found"})
	}

	// Update the document status

	update := bson.M{
		"$set": bson.M{
			"status":     body.Status,
			"modifiedOn": time.Now(),
			"modifiedBy": &modifierID,
		},
	}

	// Pemicu: saat status diubah menjadi Final Approved
	if body.Status == "Final Approved" {
		// Panggil helper untuk membuat snapshot versi
		err := helperCreateVersion(docID, "Status changed to "+body.Status)
		if err != nil {
			log.Printf("Failed to create document version: %v", err)
		} else {
			// Jika snapshot berhasil dibuat, tambahkan operasi $inc ke BSON update
			update["$inc"] = bson.M{"version": 1.0}
		}

		// Push document to Coda product backlog table
		go func() {
			codaService := config.GetCodaService()
			if codaService != nil {
				codaResp, err := codaService.UpsertRowIntoProductBacklogTable(currentDoc.Title)
				if err != nil {
					log.Printf("Failed to upsert document '%s' to Coda product backlog: %v", currentDoc.Title, err)
					// Update document with sync error
					collection := config.GetCollection("documents")
					collection.UpdateOne(context.Background(), bson.M{"_id": docID}, bson.M{
						"$set": bson.M{
							"codaSyncStatus": "failed",
							"codaSyncError":  err.Error(),
						},
					})
				} else {
					log.Printf("Successfully added document '%s' to Coda product backlog, Request ID: %s", currentDoc.Title, codaResp.RequestID)
					// Update document with request ID and pending status
					collection := config.GetCollection("documents")
					now := time.Now()

					// Extract the first (and only) row ID from the response
					var codaRowID string
					if len(codaResp.AddedRowIDs) > 0 {
						codaRowID = codaResp.AddedRowIDs[0]
					}

					collection.UpdateOne(context.Background(), bson.M{"_id": docID}, bson.M{
						"$set": bson.M{
							"codaRequestId":  codaResp.RequestID,
							"codaSyncStatus": "pending",
							"codaLastSyncAt": now,
							"codaRowId":      codaRowID,
						},
					})
				}
			} else {
				log.Printf("Coda service not available, document '%s' not added to backlog", currentDoc.Title)
			}
		}()
	}

	_, err = config.GetCollection("documents").UpdateOne(context.Background(), bson.M{"_id": docID}, update)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update document status"})
	}

	// Log activity
	utils.LogActivity(modifierID, claims.Username, "update_document_status", c.IP(), string(c.Request().Header.UserAgent()), &docID)

	// Send email notification to document owner
	go func() {
		// Get document owner details
		userCollection := config.GetCollection("users")
		var documentOwner models.User
		err := userCollection.FindOne(context.Background(), bson.M{"_id": currentDoc.OwnerID}).Decode(&documentOwner)
		if err != nil {
			log.Printf("Failed to find document owner for email notification: %v", err)
			return
		}

		// Get modifier (reviewer) details for additional context
		var modifierUser models.User
		modifierName := "System"
		err = userCollection.FindOne(context.Background(), bson.M{"_id": modifierID}).Decode(&modifierUser)
		if err == nil {
			modifierName = modifierUser.FullName
			if modifierName == "" {
				modifierName = modifierUser.Username
			}
		}

		// Send email notification using the email service
		emailService := config.GetEmailService()
		if emailService != nil && config.IsEmailServiceEnabled() {
			err := emailService.SendDocumentStatusEmail(
				documentOwner.Email,    // recipient email
				documentOwner.FullName, // author name
				currentDoc.Title,       // document title
				body.Status,            // new status
				modifierName,           // reviewer name
			)
			if err != nil {
				log.Printf("Failed to send document status email to %s: %v", documentOwner.Email, err)
			} else {
				log.Printf("Document status email sent successfully to %s (Document: %s, Status: %s)",
					documentOwner.Email, currentDoc.Title, body.Status)
			}
		} else {
			log.Printf("Email service not available, status notification not sent for document: %s", currentDoc.Title)
		}
	}()

	// return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Document status updated successfully"})

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Document status updated successfully",
		"document": fiber.Map{
			"id":     currentDoc.ID,
			"title":  currentDoc.Title,
			"status": body.Status,
		},
		"emailNotification": config.IsEmailServiceEnabled(),
	})
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

// GetDashboardStats mendapatkan statistik dokumen untuk dashboard
func GetDashboardStats(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	ownerID, _ := primitive.ObjectIDFromHex(claims.UserID)

	collection := config.GetCollection("documents")
	filter := bson.M{"ownerId": ownerID}

	// Hitung total dokumen
	totalCount, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to count documents"})
	}

	// Hitung berdasarkan status
	statusCounts := make(map[string]int64)
	statuses := []string{"draft", "Ready for Review", "Menunggu persetujuan BR", "Menunggu persetujuan DH", "Final Approved", "Rejected"}

	for _, status := range statuses {
		statusFilter := bson.M{"ownerId": ownerID, "status": status}
		count, err := collection.CountDocuments(context.Background(), statusFilter)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to count by status"})
		}
		statusCounts[status] = count
	} // Ambil dokumen terbaru (limit 10)
	opts := options.Find().SetSort(bson.D{{Key: "modifiedOn", Value: -1}}).SetLimit(10)
	cursor, err := collection.Find(context.Background(), filter, opts)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch recent documents"})
	}

	var recentDocuments []models.Document
	if err = cursor.All(context.Background(), &recentDocuments); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode documents"})
	}

	stats := fiber.Map{
		"totalDocuments": totalCount,
		"statusCounts": fiber.Map{
			"draft":          statusCounts["draft"],
			"readyForReview": statusCounts["Ready for Review"],
			"waitingBR":      statusCounts["Menunggu persetujuan BR"],
			"waitingDH":      statusCounts["Menunggu persetujuan DH"],
			"finalApproved":  statusCounts["Final Approved"],
			"rejected":       statusCounts["Rejected"],
		},
		"recentDocuments": recentDocuments,
	}

	return c.JSON(stats)
}

// SubmitDocumentForReview initializes the approval workflow for a document
func SubmitDocumentForReview(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	userID, _ := primitive.ObjectIDFromHex(claims.UserID)
	userRoleName := claims.Role

	docID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid document ID"})
	}

	collection := config.GetCollection("documents")

	// First, get the document to check ownership
	var doc models.Document
	err = collection.FindOne(context.Background(), bson.M{"_id": docID}).Decode(&doc)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Document not found"})
	}

	// Check if user owns the document
	if doc.OwnerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You can only submit your own documents"})
	}

	// Check if document is in draft or rejected status (allow resubmission of rejected documents)
	if doc.Status != "Draft" && doc.Status != "Rejected" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Only draft or rejected documents can be submitted for review"})
	}

	// Initialize or reset approval workflow
	if doc.Status == "Rejected" {
		// Reset the workflow for resubmission
		doc.ResetApprovalWorkflow()
		// Record resubmission in history

	} else {
		// Initialize new approval workflow
		doc.InitializeApprovalWorkflow(userID, claims.Username)
	}

	// Update the document
	update := bson.M{
		"$set": bson.M{
			"status":               doc.Status,
			"currentApprovalLevel": doc.CurrentApprovalLevel,
			"approvals":            doc.Approvals,
			"modifiedOn":           time.Now(),
			"modifiedBy":           &userID,
		},
	}

	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": docID}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to submit document for review"})
	}

	// Log the activity
	logAction := "Submitted document for review"
	utils.LogActivity(userID, claims.Username, logAction, c.IP(), string(c.Request().Header.UserAgent()), &docID)

	go utils.LogApprovalHistory(
		docID, // Gunakan newDocID
		"submitted",
		0,            // Level
		userRoleName, // RoleName
		userID,
		claims.Username,
		"Draft",
		doc.Status,
		"Dokumen diajukan untuk ditinjau",
	)

	// TODO: Send notification to SH approvers

	return c.JSON(fiber.Map{
		"message": "Document submitted for review successfully",
		"status":  doc.Status,
	})
}

// ApproveDocument handles approval of a document at current level
func ApproveDocument(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	userID, _ := primitive.ObjectIDFromHex(claims.UserID)
	docID, err := primitive.ObjectIDFromHex(c.Params("id"))

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid document ID"})
	}

	var body struct {
		Comments string `json:"comments"`
	}
	if err := c.BodyParser(&body); err != nil {
		body.Comments = ""
	}

	collection := config.GetCollection("documents")

	// Get the document
	var doc models.Document
	err = collection.FindOne(context.Background(), bson.M{"_id": docID}).Decode(&doc)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Document not found"})
	}

	prevStatus := doc.Status
	userRole := getUserRole(claims.UserID)

	// Check if user can approve at current level
	if !doc.CanUserApprove(userRole) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fmt.Sprintf("You cannot approve at the current level. Current level requires: %s", doc.GetCurrentApprovalLevel().RoleName),
		})
	}

	// Approve the current level
	err = doc.ApproveCurrentLevel(userID, claims.Username, body.Comments)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Update the document in database
	update := bson.M{
		"$set": bson.M{
			"status":               doc.Status,
			"currentApprovalLevel": doc.CurrentApprovalLevel,
			"approvals":            doc.Approvals,
			"modifiedOn":           time.Now(),
			"modifiedBy":           &userID,
		},
	}

	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": docID}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to approve document"})
	}

	// Check if document is now "Final Approved" and trigger Coda sync
	if doc.Status == "Final Approved" {
		log.Printf("Document '%s' has reached Final Approval status, triggering Coda sync", doc.Title)

		// Create version snapshot for final approval
		err := helperCreateVersion(docID, "Status changed to "+doc.Status)
		if err != nil {
			log.Printf("Failed to create document version: %v", err)
		} else {
			// If snapshot was created successfully, increment version
			collection.UpdateOne(context.Background(), bson.M{"_id": docID}, bson.M{
				"$inc": bson.M{"version": 1.0},
			})
		}

		// Push document to Coda product backlog table
		go func() {
			codaService := config.GetCodaService()
			if codaService != nil {
				codaResp, err := codaService.UpsertRowIntoProductBacklogTable(doc.Title)
				if err != nil {
					log.Printf("Failed to upsert document '%s' to Coda product backlog: %v", doc.Title, err)
					// Update document with sync error
					collection := config.GetCollection("documents")
					collection.UpdateOne(context.Background(), bson.M{"_id": docID}, bson.M{
						"$set": bson.M{
							"codaSyncStatus": "failed",
							"codaSyncError":  err.Error(),
						},
					})
				} else {
					log.Printf("Successfully added document '%s' to Coda product backlog, Request ID: %s", doc.Title, codaResp.RequestID)
					// Update document with request ID and pending status
					collection := config.GetCollection("documents")
					now := time.Now()

					// Extract the first (and only) row ID from the response
					var codaRowID string
					if len(codaResp.AddedRowIDs) > 0 {
						codaRowID = codaResp.AddedRowIDs[0]
					}

					collection.UpdateOne(context.Background(), bson.M{"_id": docID}, bson.M{
						"$set": bson.M{
							"codaRequestId":  codaResp.RequestID,
							"codaSyncStatus": "pending",
							"codaLastSyncAt": now,
							"codaRowId":      codaRowID,
						},
					})
				}
			} else {
				log.Printf("Coda service not available, document '%s' not added to backlog", doc.Title)
			}
		}()
	}

	// Log the activity
	logAction := fmt.Sprintf("Approved document at level %s", userRole)
	utils.LogActivity(userID, claims.Username, logAction, c.IP(), string(c.Request().Header.UserAgent()), &docID)
	go utils.LogApprovalHistory(
		docID,
		"approved",
		doc.CurrentApprovalLevel-1,
		userRole,
		userID,
		claims.Username,
		prevStatus,
		doc.Status,
		body.Comments,
	)
	// TODO: Send notification to next level approvers or document owner

	return c.JSON(fiber.Map{
		"message":         "Document approved successfully",
		"status":          doc.Status,
		"isFullyApproved": doc.IsFullyApproved(),
	})
}

// RejectDocument handles rejection of a document at current level
func RejectDocument(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	userID, _ := primitive.ObjectIDFromHex(claims.UserID)

	docID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid document ID"})
	}

	var body struct {
		Comments string `json:"comments"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Comments are required for rejection"})
	}

	if body.Comments == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Comments are required for rejection"})
	}

	collection := config.GetCollection("documents")

	// Get the document
	var doc models.Document
	err = collection.FindOne(context.Background(), bson.M{"_id": docID}).Decode(&doc)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Document not found"})
	}
	prevStatus := doc.Status
	// Get user role
	userRole := getUserRole(claims.UserID)

	// Check if user can reject at current level
	if !doc.CanUserApprove(userRole) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fmt.Sprintf("You cannot reject at the current level. Current level requires: %s", doc.GetCurrentApprovalLevel().RoleName),
		})
	}

	// Reject the current level
	err = doc.RejectCurrentLevel(userID, claims.Username, body.Comments)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Update the document in database
	update := bson.M{
		"$set": bson.M{
			"status":               doc.Status,
			"currentApprovalLevel": doc.CurrentApprovalLevel,
			"approvals":            doc.Approvals,
			"modifiedOn":           time.Now(),
			"modifiedBy":           &userID,
		},
	}

	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": docID}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to reject document"})
	}

	// Log the activity
	logAction := fmt.Sprintf("Rejected document at level %s: %s", userRole, body.Comments)
	utils.LogActivity(userID, claims.Username, logAction, c.IP(), string(c.Request().Header.UserAgent()), &docID)

	go utils.LogApprovalHistory(
		docID,
		"rejected",
		doc.CurrentApprovalLevel,
		userRole,
		userID,
		claims.Username,
		prevStatus,
		doc.Status,
		body.Comments,
	)
	// TODO: Send notification to document owner

	return c.JSON(fiber.Map{
		"message":  "Document rejected successfully",
		"status":   doc.Status,
		"comments": body.Comments,
	})
}

// GetDocumentApprovalStatus returns the approval status and history of a document
func GetDocumentApprovalStatus(c *fiber.Ctx) error {
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

	return c.JSON(fiber.Map{
		"documentId":           doc.ID,
		"title":                doc.Title,
		"status":               doc.Status,
		"currentApprovalLevel": doc.CurrentApprovalLevel,
		"approvals":            doc.Approvals,
		"isFullyApproved":      doc.IsFullyApproved(),
	})
}

// Helper function to get user role - needs to be implemented based on your user system
func getUserRole(userID string) string {
	// TODO: Implement this function to fetch user role from database
	// This is a placeholder implementation
	collection := config.GetCollection("users")
	userObjID, _ := primitive.ObjectIDFromHex(userID)

	var user models.User
	err := collection.FindOne(context.Background(), bson.M{"_id": userObjID}).Decode(&user)
	if err != nil {
		return "unknown"
	}

	// Get role name from roles collection
	roleCollection := config.GetCollection("roles")
	var role models.Role
	err = roleCollection.FindOne(context.Background(), bson.M{"_id": user.RoleID}).Decode(&role)
	if err != nil {
		return "unknown"
	}

	return role.Name
}

// GetPendingDocumentsForRole returns documents pending approval for the current user's role
func GetPendingDocumentsForRole(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	userRole := getUserRole(claims.UserID)

	if userRole == "unknown" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unable to determine user role"})
	}

	collection := config.GetCollection("documents")

	// Build filter based on role and current approval level
	var filter bson.M

	switch userRole {
	case "SH":
		// For SH level approval (level 1)
		filter = bson.M{
			"currentApprovalLevel": 1,
			"approvals.0.status":   "pending", // First approval level is pending
		}
	case "DH":
		// DH can approve at both level 1 (SH) and level 3 (DH) - FR-5.4.2.3
		filter = bson.M{
			"$or": []bson.M{
				{
					"currentApprovalLevel": 1,
					"approvals.0.status":   "pending", // Level 1: SH approval
				},
				{
					"currentApprovalLevel": 3,
					"approvals.2.status":   "pending",  // Level 3: DH approval
					"approvals.0.status":   "approved", // First level must be approved
					"approvals.1.status":   "approved", // Second level must be approved
				},
			},
		}
	case "BR":
		// For BR level approval (level 2)
		filter = bson.M{
			"currentApprovalLevel": 2,
			"approvals.1.status":   "pending",  // Second approval level is pending
			"approvals.0.status":   "approved", // First level must be approved
		}
	case "GDH":
		// For GDH final approval (level 3 only)
		filter = bson.M{
			"currentApprovalLevel": 3,
			"approvals.2.status":   "pending",  // Third approval level is pending
			"approvals.0.status":   "approved", // First level must be approved
			"approvals.1.status":   "approved", // Second level must be approved
		}
	default:
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Role not authorized for approvals"})
	}

	// Add condition to only show documents that are in approval workflow
	filter["status"] = bson.M{"$in": []string{"Ready for Review", "Menunggu persetujuan BR", "Menunggu persetujuan DH"}}

	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch pending documents"})
	}

	var documents []models.Document
	if err = cursor.All(context.Background(), &documents); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode documents"})
	}

	return c.JSON(fiber.Map{
		"role":             userRole,
		"pendingDocuments": documents,
		"count":            len(documents),
	})
}
func GetDocumentApprovalHistory(c *fiber.Ctx) error {
	docID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid document ID"})
	}

	// Akses koleksi riwayat persetujuan yang baru
	historyCollection := config.GetCollection("approval_histories")

	// Cari semua entri riwayat untuk dokumen ini
	filter := bson.M{"documentId": docID}

	// Urutkan berdasarkan timestamp agar terlama di atas (urutan kronologis)
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}})

	cursor, err := historyCollection.Find(context.Background(), filter, opts)
	if err != nil {
		log.Printf("Failed to fetch approval history for doc %s: %v", docID.Hex(), err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch approval history"})
	}

	var history []models.ApprovalHistoryEntry
	if err = cursor.All(context.Background(), &history); err != nil {
		log.Printf("Failed to decode approval history: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode approval history"})
	}

	return c.JSON(history)
}

func ReviseDocument(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	userID, _ := primitive.ObjectIDFromHex(claims.UserID)
	userRoleName := claims.Role

	docID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid document ID"})
	}

	// Panggil helper untuk membuat versi baru
	err = helperCreateVersion(docID, "Dokumen direvisi, versi baru dibuat.")
	if err != nil {
		log.Printf("Failed to create new version for revision: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create new document version for revision"})
	}

	// Ambil dokumen terbaru setelah helperCreateVersion menaikkan versinya
	collection := config.GetCollection("documents")
	var doc models.Document
	err = collection.FindOne(context.Background(), bson.M{"_id": docID}).Decode(&doc)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Document not found after revision"})
	}

	// Update status dokumen menjadi Draft
	update := bson.M{
		"$set": bson.M{
			"status":               "Draft",
			"approvals":            []models.ApprovalLevel{}, // Reset approval workflow
			"currentApprovalLevel": 0,
			"modifiedOn":           time.Now(),
			"modifiedBy":           &userID,
		},
	}
	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": docID}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update document status to Draft"})
	}

	// Log riwayat persetujuan untuk aksi revisi
	go utils.LogApprovalHistory(
		docID,
		"revised", // Aksi baru: revised
		0,         // Level
		userRoleName,
		userID,
		claims.Username,
		doc.Status, // Status lama sebelum menjadi Draft
		"Draft",
		"Dokumen direvisi menjadi versi "+fmt.Sprintf("%.1f", doc.Version),
	)

	return c.JSON(fiber.Map{
		"message":    "Document revised successfully",
		"newVersion": doc.Version,
		"newStatus":  "Draft",
	})
}

func GetDocuments(c *fiber.Ctx) error {

	collection := config.GetCollection("documents")
	filters := bson.D{}

	// 🔎 Search (judul/isi dokumen)
	if search := c.Query("search"); search != "" {
		filters = append(filters, bson.E{
			Key: "$or", Value: bson.A{
				bson.D{{"title", primitive.Regex{Pattern: search, Options: "i"}}},
				bson.D{{"content", primitive.Regex{Pattern: search, Options: "i"}}},
			},
		})
	}

	// 📂 Filter jenis dokumen
	if docType := c.Query("type"); docType != "" {
		filters = append(filters, bson.E{Key: "type", Value: docType})
	}

	// 👤 Filter pembuat dokumen (username)
	if creator := c.Query("creator"); creator != "" {
		filters = append(filters, bson.E{Key: "owner.username", Value: primitive.Regex{Pattern: creator, Options: "i"}})
	}

	// 📌 Filter status
	if status := c.Query("status"); status != "" {
		filters = append(filters, bson.E{Key: "status", Value: status})
	}

	// 📅 Filter rentang tanggal createdOn
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")
	dateFilter := bson.D{}
	if startDateStr != "" {
		if startDate, err := time.Parse("2006-01-02", startDateStr); err == nil {
			dateFilter = append(dateFilter, bson.E{Key: "$gte", Value: startDate})
		}
	}
	if endDateStr != "" {
		if endDate, err := time.Parse("2006-01-02", endDateStr); err == nil {
			dateFilter = append(dateFilter, bson.E{Key: "$lte", Value: endDate})
		}
	}
	if len(dateFilter) > 0 {
		filters = append(filters, bson.E{Key: "createdOn", Value: dateFilter})
	}

	// 🔗 Pipeline agregasi
	pipeline := mongo.Pipeline{}

	if len(filters) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: filters}})
	}

	// Join ke users biar dapet nama pembuat
	pipeline = append(pipeline,
		bson.D{{Key: "$lookup", Value: bson.D{
			{"from", "users"},
			{"localField", "ownerId"},
			{"foreignField", "_id"},
			{"as", "owner"},
		}}},
		bson.D{{Key: "$unwind", Value: bson.D{
			{"path", "$owner"},
			{"preserveNullAndEmptyArrays", true},
		}}},
		bson.D{{Key: "$addFields", Value: bson.D{
			{"id", "$_id"}, // Transform _id to id for frontend compatibility
			{"ownerUsername", "$owner.username"},
			{"ownerEmail", "$owner.email"},
			{"dokumenDibuat", "$createdOn"},
		}}},
		bson.D{{Key: "$project", Value: bson.D{
			{"_id", 0}, // Exclude the original _id field
			{"owner", 0}, // Exclude the full owner object
		}}},
	)

	cursor, err := collection.Aggregate(c.Context(), pipeline)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch documents",
		})
	}
	defer cursor.Close(c.Context())

	var documents []bson.M
	if err := cursor.All(c.Context(), &documents); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to decode documents",
		})
	}

	return c.JSON(documents)
}

// CheckDocumentCodaStatus checks the Coda mutation status for a document
func CheckDocumentCodaStatus(c *fiber.Ctx) error {
	// Get document ID from params
	docID := c.Params("id")
	if docID == "" {
		return c.Status(400).JSON(fiber.Map{
			"status":  "error",
			"message": "Document ID is required",
		})
	}

	// Convert to ObjectID
	objectID, err := primitive.ObjectIDFromHex(docID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid document ID",
		})
	}

	// Get document from database
	collection := config.GetCollection("documents")
	var doc models.Document
	err = collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(404).JSON(fiber.Map{
				"status":  "error",
				"message": "Document not found",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to retrieve document",
		})
	}

	// Check if document has a Coda request ID
	if doc.CodaRequestID == "" {
		return c.Status(400).JSON(fiber.Map{
			"status":  "error",
			"message": "Document has no Coda sync request",
		})
	}

	// Check mutation status with Coda
	codaService := config.GetCodaService()
	if codaService == nil {
		return c.Status(500).JSON(fiber.Map{
			"status":  "error",
			"message": "Coda service not available",
		})
	}

	mutationStatus, err := codaService.CheckMutationStatus(doc.CodaRequestID)
	if err != nil {
		// Update document with sync error
		now := time.Now()
		collection.UpdateOne(context.Background(), bson.M{"_id": objectID}, bson.M{
			"$set": bson.M{
				"codaSyncStatus": "failed",
				"codaSyncError":  err.Error(),
				"codaLastSyncAt": now,
			},
		})

		return c.Status(500).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Failed to check Coda mutation status: %v", err),
		})
	}

	// Update document with latest sync status
	now := time.Now()
	syncStatus := "pending"
	if mutationStatus.Completed {
		syncStatus = "completed"
	}

	var syncError string
	if mutationStatus.Warning != "" {
		syncError = mutationStatus.Warning
	}

	updateDoc := bson.M{
		"codaSyncStatus": syncStatus,
		"codaLastSyncAt": now,
	}

	if syncError != "" {
		updateDoc["codaSyncError"] = syncError
	} else {
		updateDoc["codaSyncError"] = ""
	}

	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": objectID}, bson.M{
		"$set": updateDoc,
	})
	if err != nil {
		log.Printf("Failed to update document sync status: %v", err)
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"requestId":  doc.CodaRequestID,
			"syncStatus": syncStatus,
			"completed":  mutationStatus.Completed,
			"warning":    mutationStatus.Warning,
			"lastSyncAt": now,
		},
	})
}

// RetryCodaSync retries the Coda sync for a failed document
func RetryCodaSync(c *fiber.Ctx) error {
	// Get document ID from params
	docID := c.Params("id")
	if docID == "" {
		return c.Status(400).JSON(fiber.Map{
			"status":  "error",
			"message": "Document ID is required",
		})
	}

	// Convert to ObjectID
	objectID, err := primitive.ObjectIDFromHex(docID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid document ID",
		})
	}

	// Get document from database
	collection := config.GetCollection("documents")
	var doc models.Document
	err = collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(404).JSON(fiber.Map{
				"status":  "error",
				"message": "Document not found",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to retrieve document",
		})
	}

	// Check if document is eligible for retry (must be Final Approved and have failed sync)
	if doc.Status != "Final Approved" {
		return c.Status(400).JSON(fiber.Map{
			"status":  "error",
			"message": "Only Final Approved documents can be synced to Coda",
		})
	}

	// Get Coda service
	codaService := config.GetCodaService()
	if codaService == nil {
		return c.Status(500).JSON(fiber.Map{
			"status":  "error",
			"message": "Coda service not available",
		})
	}

	// Set status to pending before attempting sync
	now := time.Now()
	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": objectID}, bson.M{
		"$set": bson.M{
			"codaSyncStatus": "pending",
			"codaSyncError":  "",
			"codaLastSyncAt": now,
		},
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to update document status",
		})
	}

	// Attempt sync in background
	go func() {
		codaResp, err := codaService.UpsertRowIntoProductBacklogTable(doc.Title)
		if err != nil {
			log.Printf("Retry failed: Failed to upsert document '%s' to Coda product backlog: %v", doc.Title, err)
			// Update document with sync error
			collection.UpdateOne(context.Background(), bson.M{"_id": objectID}, bson.M{
				"$set": bson.M{
					"codaSyncStatus": "failed",
					"codaSyncError":  err.Error(),
					"codaLastSyncAt": time.Now(),
				},
			})
		} else {
			log.Printf("Retry successful: Successfully added document '%s' to Coda product backlog, Request ID: %s", doc.Title, codaResp.RequestID)
			// Extract the first (and only) row ID from the response
			var codaRowID string
			if len(codaResp.AddedRowIDs) > 0 {
				codaRowID = codaResp.AddedRowIDs[0]
			}

			collection.UpdateOne(context.Background(), bson.M{"_id": objectID}, bson.M{
				"$set": bson.M{
					"codaRequestId":  codaResp.RequestID,
					"codaSyncStatus": "pending",
					"codaLastSyncAt": time.Now(),
					"codaRowId":      codaRowID,
					"codaSyncError":  "",
				},
			})
		}
	}()

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"message":   "Coda sync retry initiated",
			"status":    "pending",
			"timestamp": now,
		},
	})
}

// FetchDocumentDevelopmentStatus fetches the development status from Coda for a document
func FetchDocumentDevelopmentStatus(c *fiber.Ctx) error {
	// Get document ID from params
	docID := c.Params("id")
	if docID == "" {
		return c.Status(400).JSON(fiber.Map{
			"status":  "error",
			"message": "Document ID is required",
		})
	}

	// Convert to ObjectID
	objectID, err := primitive.ObjectIDFromHex(docID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid document ID",
		})
	}

	// Get document from database
	collection := config.GetCollection("documents")
	var doc models.Document
	err = collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(404).JSON(fiber.Map{
				"status":  "error",
				"message": "Document not found",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to retrieve document",
		})
	}

	// Check if document has a Coda row ID
	if doc.CodaRowID == "" {
		return c.Status(400).JSON(fiber.Map{
			"status":  "error",
			"message": "Document has no Coda row ID",
		})
	}

	// Check if sync status is completed
	if doc.CodaSyncStatus != "completed" {
		return c.Status(400).JSON(fiber.Map{
			"status":  "error",
			"message": "Document sync is not completed yet",
		})
	}

	// Get development status from Coda
	codaService := config.GetCodaService()
	if codaService == nil {
		return c.Status(500).JSON(fiber.Map{
			"status":  "error",
			"message": "Coda service not available",
		})
	}

	developmentStatus, err := codaService.GetRowDevelopmentStatus(doc.CodaRowID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Failed to fetch development status: %v", err),
		})
	}

	// Update document with development status
	now := time.Now()
	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": objectID}, bson.M{
		"$set": bson.M{
			"codaDevelopmentStatus": developmentStatus,
			"modifiedOn":            now,
		},
	})
	if err != nil {
		log.Printf("Failed to update document development status: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to update document with development status",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"developmentStatus": developmentStatus,
			"updatedAt":         now,
		},
		"message": "Development status fetched successfully",
	})
}
