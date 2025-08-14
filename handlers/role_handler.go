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
		IsDefault:   false,
	}

	_, err = collection.InsertOne(context.Background(), newRole)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create role"})
	}
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
// Ganti fungsi DeleteRole yang lama dengan ini
func DeleteRole(c *fiber.Ctx) error {
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

	// --- PERUBAHAN: Gunakan DeleteOne untuk menghapus permanen ---
	result, err := collection.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete role"})
	}

	if result.DeletedCount == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "Role not found for deletion"})
	}

	return c.JSON(fiber.Map{"message": "Role deleted successfully"})
}

// --- FUNGSI BARU YANG DIPERBAIKI ---
// GetAllPermissions sekarang mengambil data dari collection 'permissions'
func GetAllPermissions(c *fiber.Ctx) error {
	collection := config.GetCollection("permissions")

	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch permissions"})
	}

	var results []bson.M
	if err = cursor.All(context.Background(), &results); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode permissions"})
	}

	// Ekstrak hanya field 'name' untuk dikirim ke frontend
	var permissionNames []string
	for _, doc := range results {
		if name, ok := doc["name"].(string); ok {
			permissionNames = append(permissionNames, name)
		}
	}

	return c.JSON(permissionNames)
}

func ToggleRoleStatus(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	adminID, _ := primitive.ObjectIDFromHex(claims.UserID)

	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid role ID format"})
	}

	var body struct {
		IsActive bool `json:"isActive"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	collection := config.GetCollection("roles")
	update := bson.M{
		"$set": bson.M{
			"isActive":   body.IsActive,
			"modifiedOn": time.Now(),
			"modifiedBy": &adminID,
		},
	}

	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": id}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update role status"})
	}

	return c.JSON(fiber.Map{"message": "Role status updated successfully"})
}
