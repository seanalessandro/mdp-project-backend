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

// CreateRoleMenuMapping membuat pemetaan baru antara role dan menu.
func CreateRoleMenuMapping(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	adminID := claims.UserID

	var req models.RoleMenuMappingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Memastikan pemetaan belum ada untuk menghindari duplikasi.
	count, err := config.GetCollection("role_menus").CountDocuments(
		context.Background(),
		bson.M{"roleId": req.RoleID, "menuId": req.MenuID},
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database error"})
	}
	if count > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Pemetaan sudah ada"})
	}

	now := time.Now()
	newMapping := models.RoleMenuMapping{
		ID:         primitive.NewObjectID().Hex(),
		RoleID:     req.RoleID,
		MenuID:     req.MenuID,
		IsActive:   true,
		CreatedOn:  now,
		CreatedBy:  &adminID,
		ModifiedOn: now,
		ModifiedBy: &adminID,
	}

	_, err = config.GetCollection("role_menus").InsertOne(context.Background(), newMapping)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create role-menu mapping"})
	}

	return c.Status(fiber.StatusCreated).JSON(newMapping)
}

// GetAllRoleMenuMappings mendapatkan semua pemetaan role-menu.
func GetAllRoleMenuMappings(c *fiber.Ctx) error {
	collection := config.GetCollection("role_menus")
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get role-menu mappings"})
	}

	var mappings []models.RoleMenuMapping
	if err = cursor.All(context.Background(), &mappings); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode mappings"})
	}

	return c.JSON(mappings)
}

// DeleteRoleMenuMapping menghapus pemetaan berdasarkan ID.
func DeleteRoleMenuMapping(c *fiber.Ctx) error {
	id := c.Params("id")
	_, err := config.GetCollection("role_menus").DeleteOne(
		context.Background(),
		bson.M{"_id": id},
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete mapping"})
	}

	return c.JSON(fiber.Map{"message": "Mapping deleted successfully"})
}

// UpdateRoleMenus adalah handler untuk memperbarui pemetaan menu secara massal untuk sebuah role.
// Endpoint ini menghapus semua pemetaan yang ada dan membuat yang baru.
func UpdateRoleMenus(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	adminID := claims.UserID

	var req struct {
		MenuIDs []string `json:"menuIds"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	roleID := c.Params("roleId")

	collection := config.GetCollection("role_menus")

	// Hapus semua pemetaan yang sudah ada untuk role ini
	_, err := collection.DeleteMany(context.Background(), bson.M{"roleId": roleID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete old mappings"})
	}

	// Tambahkan pemetaan baru hanya jika ada menu yang dipilih
	if len(req.MenuIDs) > 0 {
		var documents []interface{}
		now := time.Now()
		for _, menuID := range req.MenuIDs {
			documents = append(documents, models.RoleMenuMapping{
				ID:         primitive.NewObjectID().Hex(),
				RoleID:     roleID,
				MenuID:     menuID,
				IsActive:   true,
				CreatedOn:  now,
				CreatedBy:  &adminID,
				ModifiedOn: now,
				ModifiedBy: &adminID,
			})
		}
		_, err = collection.InsertMany(context.Background(), documents)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to insert new mappings"})
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Role menus updated successfully"})
}
