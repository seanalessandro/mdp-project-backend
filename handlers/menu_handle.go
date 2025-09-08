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

// CreateMenu membuat item menu baru
func CreateMenu(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	adminID, _ := primitive.ObjectIDFromHex(claims.UserID)

	var req models.MenuRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	collection := config.GetCollection("menus")
	count, err := collection.CountDocuments(context.Background(), bson.M{"path": req.Path})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database error on checking path"})
	}
	if count > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Path menu sudah digunakan."})
	}

	now := time.Now()
	newMenu := models.Menu{
		BaseModel: models.BaseModel{
			ID:         primitive.NewObjectID(),
			CreatedOn:  now,
			CreatedBy:  &adminID,
			ModifiedOn: now,
			ModifiedBy: &adminID,
		},
		Name:     req.Name,
		Path:     req.Path,
		Icon:     req.Icon,
		IsActive: true, // Default: aktif
	}

	if req.ParentID != "" {
		parentID, err := primitive.ObjectIDFromHex(req.ParentID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid parent ID format"})
		}
		newMenu.ParentID = &parentID
	}

	_, err = collection.InsertOne(context.Background(), newMenu)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create menu"})
	}
	return c.Status(fiber.StatusCreated).JSON(newMenu)
}

// GetAllMenus mendapatkan semua menu
func GetAllMenus(c *fiber.Ctx) error {
	collection := config.GetCollection("menus")
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get menus"})
	}

	var menus []models.Menu
	if err = cursor.All(context.Background(), &menus); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode menus"})
	}
	return c.JSON(menus)
}

// GetMenuByID mendapatkan satu menu berdasarkan ID
func GetMenuByID(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid menu ID format"})
	}

	collection := config.GetCollection("menus")
	var menu models.Menu
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&menu)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Menu not found"})
	}
	return c.JSON(menu)
}

// UpdateMenu memperbarui data menu
func UpdateMenu(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	adminID, _ := primitive.ObjectIDFromHex(claims.UserID)

	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid menu ID format"})
	}

	var req models.MenuRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	collection := config.GetCollection("menus")
	update := bson.M{
		"$set": bson.M{
			"name":       req.Name,
			"path":       req.Path,
			"icon":       req.Icon,
			"modifiedOn": time.Now(),
			"modifiedBy": &adminID,
		},
	}
	if req.IsActive != nil {
		update["$set"].(bson.M)["isActive"] = *req.IsActive
	}

	if req.ParentID != "" {
		parentID, err := primitive.ObjectIDFromHex(req.ParentID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid parent ID format"})
		}
		update["$set"].(bson.M)["parentId"] = &parentID
	} else {
		update["$set"].(bson.M)["parentId"] = nil
	}

	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": id}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update menu"})
	}
	return c.JSON(fiber.Map{"message": "Menu updated successfully"})
}

// DeleteMenu menghapus menu secara permanen
func DeleteMenu(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid menu ID format"})
	}

	collection := config.GetCollection("menus")
	result, err := collection.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete menu"})
	}

	if result.DeletedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Menu not found for deletion"})
	}

	return c.JSON(fiber.Map{"message": "Menu deleted successfully"})
}

// ToggleMenuStatus memperbarui status 'IsActive'
func ToggleMenuStatus(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	adminID, _ := primitive.ObjectIDFromHex(claims.UserID)

	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid menu ID format"})
	}

	var body struct {
		IsActive bool `json:"isActive"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	collection := config.GetCollection("menus")
	update := bson.M{
		"$set": bson.M{
			"isActive":   body.IsActive,
			"modifiedOn": time.Now(),
			"modifiedBy": &adminID,
		},
	}

	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": id}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update menu status"})
	}
	return c.JSON(fiber.Map{"message": "Menu status updated successfully"})
}
