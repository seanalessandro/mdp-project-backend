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

// CreateRoleMenuMapping creates a new role-to-menu mapping.
func CreateRoleMenuMapping(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	adminID, _ := primitive.ObjectIDFromHex(claims.UserID)

	var req models.RoleMenuMappingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	roleID, err := primitive.ObjectIDFromHex(req.RoleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid role ID format"})
	}

	collection := config.GetCollection("role_menu_mapping")

	// Check if mapping for this role already exists
	count, err := collection.CountDocuments(context.Background(), bson.M{"roleId": roleID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database error"})
	}
	if count > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Role menu mapping already exists for this role."})
	}

	var menuObjectIDs []primitive.ObjectID
	for _, idStr := range req.MenuIDs {
		menuID, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid menu ID format in list"})
		}
		menuObjectIDs = append(menuObjectIDs, menuID)
	}

	now := time.Now()
	newMapping := models.RoleMenuMapping{
		RoleID:   roleID,
		MenuIDs:  menuObjectIDs,
		IsActive: true,
		BaseModel: models.BaseModel{
			ID:         primitive.NewObjectID(),
			CreatedOn:  now,
			CreatedBy:  &adminID,
			ModifiedOn: now,
			ModifiedBy: &adminID,
		},
	}

	_, err = collection.InsertOne(context.Background(), newMapping)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create mapping"})
	}
	return c.Status(fiber.StatusCreated).JSON(newMapping)
}

// GetRoleMenuMappings fetches all role-menu mappings.
func GetRoleMenuMappings(c *fiber.Ctx) error {
	collection := config.GetCollection("role_menu_mapping")
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get mappings"})
	}

	var mappings []models.RoleMenuMapping
	if err = cursor.All(context.Background(), &mappings); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode mappings"})
	}
	return c.JSON(mappings)
}

// GetRoleMenuMappingByID fetches a single mapping by its ID.
func GetRoleMenuMappingByID(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid mapping ID format"})
	}

	collection := config.GetCollection("role_menu_mapping")
	var mapping models.RoleMenuMapping
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&mapping)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Mapping not found"})
	}
	return c.JSON(mapping)
}

// UpdateRoleMenuMapping updates an existing role-to-menu mapping.
func UpdateRoleMenuMapping(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	adminID, _ := primitive.ObjectIDFromHex(claims.UserID)

	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid mapping ID format"})
	}

	var req models.RoleMenuMappingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var menuObjectIDs []primitive.ObjectID
	if len(req.MenuIDs) > 0 {
		for _, idStr := range req.MenuIDs {
			menuID, err := primitive.ObjectIDFromHex(idStr)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid menu ID format in list"})
			}
			menuObjectIDs = append(menuObjectIDs, menuID)
		}
	}

	collection := config.GetCollection("role_menu_mapping")

	update := bson.M{
		"menuIds":    menuObjectIDs,
		"modifiedOn": time.Now(),
		"modifiedBy": &adminID,
	}
	if req.IsActive != nil {
		update["isActive"] = *req.IsActive
	}

	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{"$set": update})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update mapping"})
	}
	return c.JSON(fiber.Map{"message": "Mapping updated successfully"})
}

// DeleteRoleMenuMapping deletes a role-to-menu mapping.
func DeleteRoleMenuMapping(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid mapping ID format"})
	}

	collection := config.GetCollection("role_menu_mapping")
	result, err := collection.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete mapping"})
	}

	if result.DeletedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Mapping not found"})
	}
	return c.JSON(fiber.Map{"message": "Mapping deleted successfully"})
}
