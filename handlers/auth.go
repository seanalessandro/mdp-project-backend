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

// Ganti lagi fungsi Login Anda dengan versi final ini
func Login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request format"})
	}

	log.Printf("DEBUG: Mencoba login untuk username: '%s'", req.Username)

	// --- PERBAIKAN: Gunakan context dengan timeout ---
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// ---------------------------------------------

	collection := config.GetCollection("users")
	var user models.User

	// Gunakan 'ctx' yang baru
	err := collection.FindOne(ctx, bson.M{"username": req.Username}).Decode(&user)

	if err != nil {
		log.Printf("DEBUG: Error saat FindOne: %v", err) // Log ini sekarang HARUS muncul jika ada error

		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid username or password (user not found)"})
		}
		// Mengembalikan pesan error yang lebih spesifik jika bukan karena 'user not found'
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database query failed", "details": err.Error()})
	}

	log.Printf("DEBUG: User '%s' ditemukan di database.", user.Username)

	if user.Provider != "local" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "This account uses a social login. Please log in with Google."})
	}

	if !utils.CheckPasswordHash(req.Password, user.Password) {
		log.Printf("DEBUG: Password untuk user '%s' TIDAK COCOK.", user.Username)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid username or password (password incorrect)"})
	}

	log.Printf("DEBUG: Password cocok. Login berhasil untuk user '%s'.", user.Username)
	return generateLoginResponse(c, user)
}

// ChangePassword untuk pengguna yang sedang login
func ChangePassword(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	userID, _ := primitive.ObjectIDFromHex(claims.UserID)

	var req models.ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request format"})
	}

	collection := config.GetCollection("users")
	var user models.User
	err := collection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	// Verifikasi password lama
	if !utils.CheckPasswordHash(req.OldPassword, user.Password) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Current password is incorrect"})
	}

	// Hash password baru
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to process new password"})
	}

	// Update password di database dengan audit fields
	update := bson.M{
		"$set": bson.M{
			"password":   hashedPassword,
			"modifiedOn": time.Now(),
			"modifiedBy": &userID,
		},
	}
	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": userID}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update password"})
	}

	return c.JSON(fiber.Map{"message": "Password changed successfully"})
}

// Logout (opsional, bisa digunakan untuk logging)
func Logout(c *fiber.Ctx) error {
	// Logout pada JWT biasanya ditangani oleh frontend dengan menghapus token.
	// Endpoint ini bisa ada untuk tujuan logging jika diperlukan.
	return c.JSON(fiber.Map{"message": "Logged out successfully"})
}
