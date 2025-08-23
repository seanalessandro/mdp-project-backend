package handlers

import (
	"context"
	"log"
	"mdp-project-backend/config"
	"mdp-project-backend/models"
	"mdp-project-backend/services"
	"mdp-project-backend/utils"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// GetUsers retrieves all users with pagination and filtering
func GetUsers(c *fiber.Ctx) error {
	// Check if user has admin permissions
	user := c.Locals("user").(*utils.Claims)
	if user.Role != "admin" && user.Role != "super_admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}

	// Parse query parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	search := c.Query("search", "")
	role := c.Query("role", "")
	status := c.Query("status", "")

	// Build filter
	filter := bson.M{}
	if search != "" {
		filter["$or"] = []bson.M{
			{"username": bson.M{"$regex": search, "$options": "i"}},
			{"email": bson.M{"$regex": search, "$options": "i"}},
		}
	}
	if role != "" {
		// Since we're using roleId now, we might need to search by role name differently
		// For now, let's comment this out or modify it to work with the new structure
		// filter["role"] = role
	}
	if status != "" {
		if status == "active" {
			filter["isActive"] = true
		} else if status == "inactive" {
			filter["isActive"] = false
		}
	}

	collection := config.GetCollection("users")

	// Count total documents
	total, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	// Calculate pagination
	skip := (page - 1) * limit

	// Use aggregation to populate role details
	pipeline := []bson.M{
		// Match stage (filtering)
		{"$match": filter},
		// Sort stage
		{"$sort": bson.D{{"createdOn", -1}}},
		// Skip stage
		{"$skip": skip},
		// Limit stage
		{"$limit": limit},
		// Lookup to populate role details
		{
			"$lookup": bson.M{
				"from":         "roles",
				"localField":   "roleId",
				"foreignField": "_id",
				"as":           "role",
			},
		},
		// Unwind the role array (convert from array to object)
		{
			"$unwind": bson.M{
				"path":                       "$role",
				"preserveNullAndEmptyArrays": true,
			},
		},
		// Project to remove password and format the response
		{
			"$project": bson.M{
				"password": 0, // Exclude password field
			},
		},
	}

	// Execute aggregation
	cursor, err := collection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Database error during aggregation",
		})
	}
	defer cursor.Close(context.Background())

	var users []bson.M // Use bson.M to handle the populated role
	if err = cursor.All(context.Background(), &users); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	return c.JSON(fiber.Map{
		"data": users,
		"pagination": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetUser retrieves a single user by ID
func GetUser(c *fiber.Ctx) error {
	// Check permissions
	user := c.Locals("user").(*utils.Claims)
	if user.Role != "admin" && user.Role != "super_admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}

	userID := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	collection := config.GetCollection("users")
	var dbUser models.User
	err = collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&dbUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(404).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	// Remove password from response
	dbUser.Password = ""

	return c.JSON(dbUser)
}

// CreateUser creates a new user
func CreateUser(c *fiber.Ctx) error {
	// Check permissions
	user := c.Locals("user").(*utils.Claims)
	if user.Role != "admin" && user.Role != "super_admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}

	var newUser models.User
	if err := c.BodyParser(&newUser); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request format",
		})
	}

	// Validate required fields (password not required since we'll generate it)
	if newUser.Username == "" || newUser.Email == "" || newUser.FullName == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Username, email, and full name are required",
		})
	}

	// Validate username format
	if err := utils.ValidateUsername(newUser.Username); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	collection := config.GetCollection("users")

	// Check if username or email already exists
	existingUser := collection.FindOne(context.Background(), bson.M{
		"$or": []bson.M{
			{"username": newUser.Username},
			{"email": newUser.Email},
		},
	})
	if existingUser.Err() == nil {
		return c.Status(409).JSON(fiber.Map{
			"error": "Username or email already exists",
		})
	}

	// Generate random password (instead of using provided password)
	// This is similar to auto-generating passwords in enterprise systems
	randomPassword := services.GenerateRandomPassword(12) // Generate 12-character password
	log.Printf("Generated password for user %s: %s", newUser.Username, randomPassword)

	// Hash the generated password
	hashedPassword, err := utils.HashPassword(randomPassword)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error processing password",
		})
	}

	// Set user data
	now := time.Now()
	newUser.ID = primitive.NewObjectID()
	newUser.BaseModel = models.BaseModel{
		ID:         newUser.ID,
		CreatedOn:  now,
		CreatedBy:  nil,
		ModifiedOn: now,
		ModifiedBy: nil,
	}
	newUser.Password = hashedPassword
	newUser.IsActive = true
	newUser.Provider = "local"

	// Insert user
	_, err = collection.InsertOne(context.Background(), newUser)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	// Send welcome email with generated password
	// This is similar to @Async email sending in Spring Boot
	emailService := config.GetEmailService()
	if emailService != nil && config.IsEmailServiceEnabled() {
		// Send email asynchronously (non-blocking)
		go func() {
			err := emailService.SendWelcomeEmail(
				newUser.Email,
				newUser.FullName,
				newUser.Username,
				randomPassword, // Send the plain password via email
			)
			if err != nil {
				log.Printf("Failed to send welcome email to %s: %v", newUser.Email, err)
			}
		}()
		log.Printf("Welcome email queued for %s", newUser.Email)
	} else {
		log.Printf("Email service not available, welcome email not sent for %s", newUser.Email)
		// You might want to return this info to the admin
	}

	// Log activity
	if userObjectID, err := primitive.ObjectIDFromHex(user.UserID); err == nil {
		utils.LogActivity(userObjectID, user.Username, "create_user", c.IP(), c.Get("User-Agent"))
	}

	// Remove password from response
	newUser.Password = ""

	// Return success response with info about email
	response := fiber.Map{
		"message": "User created successfully",
		"user":    newUser,
	}

	if config.IsEmailServiceEnabled() {
		response["email_status"] = "Welcome email sent to " + newUser.Email
	} else {
		response["email_status"] = "Email service unavailable. Please manually provide credentials to user."
		response["generated_password"] = randomPassword // Include password in response if email failed
	}

	return c.Status(201).JSON(response)
}

// UpdateUser updates an existing user
func UpdateUser(c *fiber.Ctx) error {
	// Check permissions
	user := c.Locals("user").(*utils.Claims)
	if user.Role != "admin" && user.Role != "super_admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}

	userID := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var updateData bson.M
	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request format",
		})
	}

	// Remove sensitive fields that shouldn't be updated via this endpoint
	delete(updateData, "_id")
	delete(updateData, "password")
	delete(updateData, "createdOn")
	updateData["modifiedOn"] = time.Now()

	collection := config.GetCollection("users")

	// Check if user exists
	var existingUser models.User
	err = collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&existingUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(404).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	// Update user
	_, err = collection.UpdateOne(
		context.Background(),
		bson.M{"_id": objectID},
		bson.M{"$set": updateData},
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	// Log activity
	if userObjectID, err := primitive.ObjectIDFromHex(user.UserID); err == nil {
		utils.LogActivity(userObjectID, user.Username, "update_user", c.IP(), c.Get("User-Agent"))
	}

	// Get updated user
	var updatedUser models.User
	err = collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&updatedUser)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	// Remove password from response
	updatedUser.Password = ""

	return c.JSON(updatedUser)
}

// DeleteUser soft deletes a user
func DeleteUser(c *fiber.Ctx) error {
	// Check permissions
	user := c.Locals("user").(*utils.Claims)
	if user.Role != "admin" && user.Role != "super_admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}

	userID := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Prevent user from deleting themselves
	if userID == user.UserID {
		return c.Status(400).JSON(fiber.Map{
			"error": "Cannot delete your own account",
		})
	}

	collection := config.GetCollection("users")

	// Check if user exists
	var existingUser models.User
	err = collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&existingUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(404).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	// Soft delete by updating status
	_, err = collection.UpdateOne(
		context.Background(),
		bson.M{"_id": objectID},
		bson.M{
			"$set": bson.M{
				"isActive":   false,
				"modifiedOn": time.Now(),
			},
		},
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	// Log activity
	if userObjectID, err := primitive.ObjectIDFromHex(user.UserID); err == nil {
		utils.LogActivity(userObjectID, user.Username, "delete_user", c.IP(), c.Get("User-Agent"))
	}

	return c.JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}

// ResetUserPassword resets a user's password
func ResetUserPassword(c *fiber.Ctx) error {
	// Check permissions
	user := c.Locals("user").(*utils.Claims)
	if user.Role != "admin" && user.Role != "super_admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}

	userID := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var resetReq struct {
		NewPassword string `json:"new_password"`
	}
	if err := c.BodyParser(&resetReq); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request format",
		})
	}

	if resetReq.NewPassword == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "New password is required",
		})
	}

	collection := config.GetCollection("users")

	// Check if user exists
	var existingUser models.User
	err = collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&existingUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(404).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(resetReq.NewPassword)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error processing password",
		})
	}

	// Update password
	_, err = collection.UpdateOne(
		context.Background(),
		bson.M{"_id": objectID},
		bson.M{
			"$set": bson.M{
				"password":   hashedPassword,
				"modifiedOn": time.Now(),
			},
		},
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	// Log activity
	if userObjectID, err := primitive.ObjectIDFromHex(user.UserID); err == nil {
		utils.LogActivity(userObjectID, user.Username, "reset_user_password", c.IP(), c.Get("User-Agent"))
	}

	return c.JSON(fiber.Map{
		"message": "Password reset successfully",
	})
}

// UpdateUserRole updates only the role of a specific user
func UpdateUserRole(c *fiber.Ctx) error {
	// Check permissions
	user := c.Locals("user").(*utils.Claims)
	if user.Role != "admin" && user.Role != "super_admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}

	userID := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var roleUpdateReq struct {
		RoleId string `json:"roleId" validate:"required"`
	}

	if err := c.BodyParser(&roleUpdateReq); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate role ID
	roleObjectID, err := primitive.ObjectIDFromHex(roleUpdateReq.RoleId)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid role ID format",
		})
	}

	// Check if role exists
	roleCollection := config.GetCollection("roles")
	var role models.Role
	err = roleCollection.FindOne(context.Background(), bson.M{"_id": roleObjectID}).Decode(&role)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(404).JSON(fiber.Map{
				"error": "Role not found",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"error": "Database error checking role",
		})
	}

	// Update user role
	collection := config.GetCollection("users")
	updateData := bson.M{
		"$set": bson.M{
			"roleId":     roleObjectID,
			"modifiedOn": time.Now(),
		},
	}

	if adminObjectID, err := primitive.ObjectIDFromHex(user.UserID); err == nil {
		updateData["$set"].(bson.M)["modifiedBy"] = adminObjectID
	}

	result, err := collection.UpdateOne(context.Background(), bson.M{"_id": objectID}, updateData)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Database error updating user",
		})
	}

	if result.MatchedCount == 0 {
		return c.Status(404).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Log activity
	if userObjectID, err := primitive.ObjectIDFromHex(user.UserID); err == nil {
		utils.LogActivity(userObjectID, user.Username, "update_user_role", c.IP(), c.Get("User-Agent"))
	}

	return c.JSON(fiber.Map{
		"message": "User role updated successfully",
	})
}
