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

// GetCommentsForDocument mengambil semua komentar untuk sebuah dokumen
func GetCommentsForDocument(c *fiber.Ctx) error {
	docID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid document ID"})
	}

	collection := config.GetCollection("comments")
	cursor, err := collection.Find(context.Background(), bson.M{"documentId": docID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch comments"})
	}

	var comments []models.Comment
	if err = cursor.All(context.Background(), &comments); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode comments"})
	}

	return c.JSON(comments)
}

// CreateComment membuat komentar baru
func CreateComment(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	authorID, _ := primitive.ObjectIDFromHex(claims.UserID)
	docID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid document ID"})
	}

	var body struct {
		Content    string `json:"content"`
		MarkedText string `json:"markedText"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	now := time.Now()
	newComment := models.Comment{
		BaseModel: models.BaseModel{
			ID:         primitive.NewObjectID(),
			CreatedOn:  now,
			CreatedBy:  &authorID,
			ModifiedOn: now,
			ModifiedBy: &authorID,
		},
		DocumentID: docID,
		AuthorID:   authorID,
		Content:    body.Content,
		MarkedText: body.MarkedText,
		Replies:    []models.Reply{},
	}

	collection := config.GetCollection("comments")
	_, err = collection.InsertOne(context.Background(), newComment)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create comment"})
	}

	return c.Status(fiber.StatusCreated).JSON(newComment)
}

// CreateReply menambahkan balasan ke komentar
func CreateReply(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	authorID, _ := primitive.ObjectIDFromHex(claims.UserID)
	commentID, err := primitive.ObjectIDFromHex(c.Params("commentId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid comment ID"})
	}

	var body struct {
		Content string `json:"content"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	newReply := models.Reply{
		ID:        primitive.NewObjectID(),
		AuthorID:  authorID,
		Content:   body.Content,
		CreatedAt: time.Now(),
	}

	collection := config.GetCollection("comments")
	update := bson.M{
		"$push": bson.M{"replies": newReply},
		"$set":  bson.M{"modifiedOn": time.Now(), "modifiedBy": &authorID},
	}

	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": commentID}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to add reply"})
	}

	return c.Status(fiber.StatusCreated).JSON(newReply)
}
