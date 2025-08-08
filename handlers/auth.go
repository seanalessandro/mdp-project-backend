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

func Login(c *fiber.Ctx) error {
	var loginReq models.LoginRequest
	if err := c.BodyParser(&loginReq); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request format",
		})
	}

	// Validate input format
	if err := utils.ValidateUsername(loginReq.Username); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Username format is invalid",
		})
	}

	if loginReq.Password == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Password is required",
		})
	}

	// Find user in database
	collection := config.GetCollection("users")
	var user models.User
	err := collection.FindOne(context.Background(), bson.M{"username": loginReq.Username}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(401).JSON(fiber.Map{
				"error": "Username or password is incorrect",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	// Check if user is active
	if !user.IsActive {
		return c.Status(401).JSON(fiber.Map{
			"error": "Account is deactivated",
		})
	}

	// Verify password
	if !utils.CheckPasswordHash(loginReq.Password, user.Password) {
		// Log failed login attempt
		logActivity(user.ID, user.Username, "failed_login", c.IP(), c.Get("User-Agent"))
		
		return c.Status(401).JSON(fiber.Map{
			"error": "Username or password is incorrect",
		})
	}

	// Update last login
	now := time.Now()
	user.LastLogin = &now
	collection.UpdateOne(
		context.Background(),
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{"last_login": now}},
	)

	// Generate JWT token
	token, err := utils.GenerateJWT(user.Username, user.Role, user.ID.Hex())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	// Log successful login
	logActivity(user.ID, user.Username, "successful_login", c.IP(), c.Get("User-Agent"))

	return c.JSON(models.LoginResponse{
		Token: token,
		User:  user,
	})
}

func ChangePassword(c *fiber.Ctx) error {
	user := c.Locals("user").(*utils.Claims)
	
	var changeReq models.ChangePasswordRequest
	if err := c.BodyParser(&changeReq); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request format",
		})
	}

	// Validate new password
	if err := utils.ValidatePassword(changeReq.NewPassword); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Get user from database
	collection := config.GetCollection("users")
	userID, _ := primitive.ObjectIDFromHex(user.UserID)
	var dbUser models.User
	err := collection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&dbUser)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	// Verify old password
	if !utils.CheckPasswordHash(changeReq.OldPassword, dbUser.Password) {
		return c.Status(400).JSON(fiber.Map{
			"error": "Current password is incorrect",
		})
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(changeReq.NewPassword)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to hash password",
		})
	}

	// Update password in database
	_, err = collection.UpdateOne(
		context.Background(),
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{
			"password":   hashedPassword,
			"updated_at": time.Now(),
		}},
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to update password",
		})
	}

	// Log password change
	logActivity(userID, user.Username, "password_changed", c.IP(), c.Get("User-Agent"))

	return c.JSON(fiber.Map{
		"message": "Password changed successfully",
	})
}

func GetProfile(c *fiber.Ctx) error {
	user := c.Locals("user").(*utils.Claims)
	
	collection := config.GetCollection("users")
	userID, _ := primitive.ObjectIDFromHex(user.UserID)
	var dbUser models.User
	err := collection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&dbUser)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	return c.JSON(dbUser)
}

func Logout(c *fiber.Ctx) error {
	user := c.Locals("user").(*utils.Claims)
	userID, _ := primitive.ObjectIDFromHex(user.UserID)
	
	// Log logout activity
	logActivity(userID, user.Username, "logout", c.IP(), c.Get("User-Agent"))

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// Helper function to log user activities
func logActivity(userID primitive.ObjectID, username, action, ipAddress, userAgent string) {
	collection := config.GetCollection("activity_logs")
	log := models.ActivityLog{
		UserID:    userID,
		Username:  username,
		Action:    action,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Timestamp: time.Now(),
	}
	collection.InsertOne(context.Background(), log)
}
