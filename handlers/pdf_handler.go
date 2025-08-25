package handlers

import (
	"fmt"
	"mdp-project-backend/config"
	"mdp-project-backend/models"
	"mdp-project-backend/services"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ExportDocumentToPDF exports a document to PDF format
func ExportDocumentToPDF(c *fiber.Ctx) error {
	docID := c.Params("id")
	if docID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Document ID is required",
		})
	}

	objectID, err := primitive.ObjectIDFromHex(docID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid document ID format",
		})
	}

	// Fetch document from database
	collection := config.GetCollection("documents")
	var document models.Document
	err = collection.FindOne(c.Context(), bson.M{"_id": objectID}).Decode(&document)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Document not found",
		})
	}

	// Get document author information
	userCollection := config.GetCollection("users")
	var author models.User
	err = userCollection.FindOne(c.Context(), bson.M{"_id": document.OwnerID}).Decode(&author)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch document author information",
		})
	}

	// Create PDF service
	pdfService := services.NewPDFService()

	// Prepare document data for PDF generation
	pdfData := services.DocumentPDFRequest{
		Title:       document.Title,
		Content:     document.Content,
		Status:      document.Status,
		Version:     document.Version,
		DocNo:       document.DocNo,
		Priority:    document.Priority,
		Author:      author.FullName,
		CreatedDate: document.CreatedOn.Format("January 2, 2006"),
	}

	// Generate PDF
	pdfBytes, err := pdfService.GenerateDocumentPDF(pdfData)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to generate PDF: %v", err),
		})
	}

	// Prepare filename
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.pdf", 
		sanitizeFilename(document.Title), 
		timestamp)

	// Set response headers for PDF download
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))

	return c.Send(pdfBytes)
}

// GetDocumentPDFPreview generates a PDF preview
func GetDocumentPDFPreview(c *fiber.Ctx) error {
	docID := c.Params("id")
	if docID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Document ID is required",
		})
	}

	objectID, err := primitive.ObjectIDFromHex(docID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid document ID format",
		})
	}

	// Fetch document from database
	collection := config.GetCollection("documents")
	var document models.Document
	err = collection.FindOne(c.Context(), bson.M{"_id": objectID}).Decode(&document)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Document not found",
		})
	}

	// Get document author information
	userCollection := config.GetCollection("users")
	var author models.User
	err = userCollection.FindOne(c.Context(), bson.M{"_id": document.OwnerID}).Decode(&author)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch document author information",
		})
	}

	// Create PDF service
	pdfService := services.NewPDFService()

	// Prepare document data for PDF generation
	pdfData := services.DocumentPDFRequest{
		Title:       document.Title,
		Content:     document.Content,
		Status:      document.Status,
		Version:     document.Version,
		DocNo:       document.DocNo,
		Priority:    document.Priority,
		Author:      author.FullName,
		CreatedDate: document.CreatedOn.Format("January 2, 2006"),
	}

	// Generate PDF
	pdfBytes, err := pdfService.GenerateDocumentPDF(pdfData)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to generate PDF preview: %v", err),
		})
	}

	// Set response headers for PDF inline display
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "inline")
	c.Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))

	return c.Send(pdfBytes)
}

// sanitizeFilename removes invalid characters from filename
func sanitizeFilename(filename string) string {
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	result := filename
	
	for _, char := range invalidChars {
		result = strings.ReplaceAll(result, char, "_")
	}
	
	if len(result) > 50 {
		result = result[:47] + "..."
	}
	
	return result
}
