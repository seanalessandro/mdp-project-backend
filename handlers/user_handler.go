package handlers

import (
	"context"
	"log"
	"mdp-project-backend/config"
	"mdp-project-backend/models"
	"mdp-project-backend/utils"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// GetProfile untuk mendapatkan data profil pengguna yang sedang login
func GetProfile(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	userID, _ := primitive.ObjectIDFromHex(claims.UserID)

	collection := config.GetCollection("users")
	var user models.User
	err := collection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	return c.JSON(user)
}

// AdminCreateUser adalah handler untuk admin membuat user baru
func AdminCreateUser(c *fiber.Ctx) error {
	adminClaims := c.Locals("user").(*utils.Claims)
	adminID, _ := primitive.ObjectIDFromHex(adminClaims.UserID)

	var req models.AdminCreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	roleID, err := primitive.ObjectIDFromHex(req.RoleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid Role ID format"})
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to hash password"})
	}

	now := time.Now()
	newUser := models.User{
		BaseModel: models.BaseModel{
			ID:         primitive.NewObjectID(),
			CreatedOn:  now,
			CreatedBy:  &adminID,
			ModifiedOn: now,
			ModifiedBy: &adminID,
		},
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
		RoleID:   roleID,
		IsActive: true,
		Provider: "local",
	}

	collection := config.GetCollection("users")
	_, err = collection.InsertOne(context.Background(), newUser)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Email or username already exists"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create user"})
	}

	return c.Status(fiber.StatusCreated).JSON(newUser)
}

// generateLoginResponse adalah fungsi helper yang sudah diperbaiki
func generateLoginResponse(c *fiber.Ctx, user models.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// --- AMBIL DETAIL ROLE BERDASARKAN user.RoleID ---
	roleCollection := config.GetCollection("roles")
	var role models.Role
	err := roleCollection.FindOne(ctx, bson.M{"_id": user.RoleID}).Decode(&role)
	if err != nil {
		log.Printf("WARNING: Could not find role with ID %s for user %s. Error: %v", user.RoleID.Hex(), user.Username, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "User role configuration is invalid"})
	}
	// --------------------------------------------

	// Update last login
	now := time.Now()
	update := bson.M{"$set": bson.M{"lastLogin": now, "modifiedOn": now}}
	config.GetCollection("users").UpdateOne(ctx, bson.M{"_id": user.ID}, update)
	user.LastLogin = &now

	// --- PERBAIKAN: Generate JWT dengan NAMA PERAN (role.Name), bukan ID ---
	token, err := utils.GenerateJWT(user.Username, role.Name, user.ID.Hex())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate session token"})
	}

	// --- PERBAIKAN: Kirim response lengkap dengan detail user dan role ---
	return c.JSON(models.LoginResponse{
		Token: token,
		User:  user,
		Role:  role,
	})
}

// ResetAdminPassword adalah handler sementara (sebaiknya dihapus setelah digunakan)
func ResetAdminPassword(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	plainPassword := "password123!"
	newHashedPassword, err := utils.HashPassword(plainPassword)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat hash baru", "details": err.Error()})
	}

	collection := config.GetCollection("users")
	filter := bson.M{"username": "admin"}
	update := bson.M{
		"$set": bson.M{
			"password":   newHashedPassword,
			"modifiedOn": time.Now(),
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal update database", "details": err.Error()})
	}
	if result.MatchedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User 'admin' tidak ditemukan untuk di-reset."})
	}

	log.Printf("SUCCESS: Password untuk 'admin' telah di-reset dengan hash baru: %s", newHashedPassword)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Password untuk user 'admin' berhasil di-reset. Silakan coba login kembali."})
}
