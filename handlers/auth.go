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
	claims := c.Locals("user").(*utils.Claims)
	userID, _ := primitive.ObjectIDFromHex(claims.UserID)

	// Clear refresh token from database
	collection := config.GetCollection("users")
	_, err := collection.UpdateOne(
		context.Background(),
		bson.M{"_id": userID},
		bson.M{"$unset": bson.M{"refreshToken": "", "tokenExpiry": ""}},
	)
	if err != nil {
		log.Printf("Failed to clear refresh token for user %s: %v", claims.Username, err)
	}

	return c.JSON(fiber.Map{"message": "Logged out successfully"})
}

// RefreshToken generates new access token using refresh token
func RefreshToken(c *fiber.Ctx) error {
	var req models.RefreshTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request format"})
	}

	// Validate refresh token
	refreshClaims, err := utils.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid refresh token"})
	}

	userID, err := primitive.ObjectIDFromHex(refreshClaims.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	// Check if refresh token exists in database and is still valid
	collection := config.GetCollection("users")
	var user models.User
	err = collection.FindOne(context.Background(), bson.M{
		"_id":          userID,
		"refreshToken": req.RefreshToken,
		"tokenExpiry":  bson.M{"$gte": time.Now()},
	}).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Refresh token expired or invalid"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database error"})
	}

	// Get user role
	roleCollection := config.GetCollection("roles")
	var role models.Role
	err = roleCollection.FindOne(context.Background(), bson.M{"_id": user.RoleID}).Decode(&role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "User role configuration is invalid"})
	}

	// Generate new tokens
	newToken, err := utils.GenerateJWT(user.Username, role.Name, user.ID.Hex())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate new access token"})
	}

	newRefreshToken, err := utils.GenerateRefreshToken(user.Username, user.ID.Hex())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate new refresh token"})
	}

	// Update refresh token in database
	now := time.Now()
	tokenExpiry := now.Add(7 * 24 * time.Hour)
	_, err = collection.UpdateOne(
		context.Background(),
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{
			"refreshToken": newRefreshToken,
			"tokenExpiry":  tokenExpiry,
			"modifiedOn":   now,
		}},
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update refresh token"})
	}

	return c.JSON(models.TokenResponse{
		Token:        newToken,
		RefreshToken: newRefreshToken,
		User:         user,
		Role:         role,
	})
}
