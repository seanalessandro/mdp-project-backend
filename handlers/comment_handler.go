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
	"go.mongodb.org/mongo-driver/mongo"
)

// CreateComment sekarang menangani komentar utama dan semua level balasan
func CreateComment(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	authorID, _ := primitive.ObjectIDFromHex(claims.UserID)
	docID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		// Jika tidak ada docID di params, ini mungkin adalah balasan yang dikirim ke endpoint lain
		// Kita akan tangani ini nanti di route terpisah jika diperlukan
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid document ID"})
	}

	var body struct {
		Content  string `json:"content"`
		ParentID string `json:"parentId"` // Terima parentId (opsional)
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if body.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Comment content cannot be empty"})
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
	}

	// Jika ada ParentID yang valid, set sebagai balasan
	if body.ParentID != "" {
		parentID, err := primitive.ObjectIDFromHex(body.ParentID)
		if err == nil {
			newComment.ParentID = &parentID
		}
	}

	collection := config.GetCollection("comments")
	_, err = collection.InsertOne(context.Background(), newComment)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create comment"})
	}

	// Ambil kembali data yang baru saja di-insert beserta info author untuk dikirim ke frontend
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{"_id": newComment.ID}}},
		{{"$lookup", bson.D{
			{"from", "users"},
			{"localField", "authorId"},
			{"foreignField", "_id"},
			{"as", "author"},
		}}},
		{{"$unwind", "$author"}},
	}

	cursor, err := collection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve created comment"})
	}
	var createdCommentWithAuthor []bson.M
	if err = cursor.All(context.Background(), &createdCommentWithAuthor); err != nil || len(createdCommentWithAuthor) == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode created comment"})
	}

	return c.Status(fiber.StatusCreated).JSON(createdCommentWithAuthor[0])
}

// GetCommentsForDocument mengambil SEMUA komentar (termasuk balasan) sebagai daftar datar
func GetCommentsForDocument(c *fiber.Ctx) error {
	docID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid document ID"})
	}

	collection := config.GetCollection("comments")
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{"documentId": docID}}},
		{{"$lookup", bson.D{
			{"from", "users"},
			{"localField", "authorId"},
			{"foreignField", "_id"},
			{"as", "author"},
		}}},
		{{"$unwind", bson.D{{"path", "$author"}, {"preserveNullAndEmptyArrays", true}}}},
		{{"$sort", bson.D{{"createdOn", 1}}}}, // Diurutkan dari terlama ke terbaru
		{{"$project", bson.D{
			{"author.password", 0}, // Pastikan tidak mengirim password
			{"author.roleId", 0},
		}}},
	}

	cursor, err := collection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch comments"})
	}

	var comments []bson.M
	if err = cursor.All(context.Background(), &comments); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode comments"})
	}

	if comments == nil {
		return c.JSON([]models.Comment{})
	}

	return c.JSON(comments)
}

// Handler CreateReply tidak diperlukan lagi, karena sudah ditangani oleh CreateComment
