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

// CreateRole membuat role baru
func CreateRole(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	adminID, _ := primitive.ObjectIDFromHex(claims.UserID)

	var req models.RoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	collection := config.GetCollection("roles")

	// Cek duplikat nama
	count, err := collection.CountDocuments(context.Background(), bson.M{"name": req.Name})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database error on checking name"})
	}
	if count > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama role sudah digunakan."})
	}

	now := time.Now()
	newRole := models.Role{
		BaseModel: models.BaseModel{
			ID:         primitive.NewObjectID(),
			CreatedOn:  now,
			CreatedBy:  &adminID,
			ModifiedOn: now,
			ModifiedBy: &adminID,
		},
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
		IsActive:    true,
		IsDefault:   false, // Role yang dibuat manual tidak default
	}

	_, err = collection.InsertOne(context.Background(), newRole)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create role"})
	}

	// TODO: Log activity
	return c.Status(fiber.StatusCreated).JSON(newRole)
}

// GetAllRoles mendapatkan semua role
func GetAllRoles(c *fiber.Ctx) error {
	collection := config.GetCollection("roles")
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get roles"})
	}

	var roles []models.Role
	if err = cursor.All(context.Background(), &roles); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode roles"})
	}

	return c.JSON(roles)
}

// GetRoleByID mendapatkan satu role berdasarkan ID
func GetRoleByID(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid role ID format"})
	}

	collection := config.GetCollection("roles")
	var role models.Role
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&role)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Role not found"})
	}

	return c.JSON(role)
}

// UpdateRole memperbarui data role
func UpdateRole(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	adminID, _ := primitive.ObjectIDFromHex(claims.UserID)

	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid role ID format"})
	}

	var req models.RoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	collection := config.GetCollection("roles")
	update := bson.M{
		"$set": bson.M{
			"name":        req.Name,
			"description": req.Description,
			"permissions": req.Permissions,
			"modifiedOn":  time.Now(),
			"modifiedBy":  &adminID,
		},
	}

	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": id}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update role"})
	}

	return c.JSON(fiber.Map{"message": "Role updated successfully"})
}

// DeleteRole menghapus (soft delete) sebuah role
func DeleteRole(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	adminID, _ := primitive.ObjectIDFromHex(claims.UserID)

	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid role ID format"})
	}

	collection := config.GetCollection("roles")

	// Cek apakah role tersebut default
	var roleToDelete models.Role
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&roleToDelete)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Role not found"})
	}

	if roleToDelete.IsDefault {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Role default tidak dapat dihapus."})
	}

	// Lakukan soft delete dengan menonaktifkan role
	update := bson.M{
		"$set": bson.M{
			"isActive":   false,
			"modifiedOn": time.Now(),
			"modifiedBy": &adminID,
		},
	}

	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": id}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete role"})
	}

	return c.JSON(fiber.Map{"message": "Role deactivated successfully"})
}
