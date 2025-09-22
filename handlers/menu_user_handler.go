package handlers

import (
	"context"
	"log"
	"mdp-project-backend/config"
	"mdp-project-backend/models"
	"mdp-project-backend/utils"
	"sort"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetUserMenus returns the menus accessible to the current user based on their role
func GetUserMenus(c *fiber.Ctx) error {
	claims := c.Locals("user").(*utils.Claims)
	userID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	// Get user details to find their role
	userCollection := config.GetCollection("users")
	var user models.User
	err = userCollection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	// Debug logging for user data
	log.Printf("User found - ID: %s, Role ID: %s", user.ID.Hex(), user.RoleID.Hex())

	// Get role-menu mappings for this user's role
	mappingCollection := config.GetCollection("role_menu_mapping") // Changed from plural to singular
	var roleMenuMapping struct {
		RoleID   primitive.ObjectID   `bson:"roleId"`
		MenuIds  []primitive.ObjectID `bson:"menuIds"`
		IsActive bool                 `bson:"isActive"`
	}

	// Add debug logging
	log.Printf("Looking for role mappings for user role ID: %s", user.RoleID.Hex())

	err = mappingCollection.FindOne(context.Background(), bson.M{
		"roleId":   user.RoleID,
		"isActive": true,
	}).Decode(&roleMenuMapping)

	if err != nil {
		log.Printf("No menu mapping found for role %s: %v", user.RoleID.Hex(), err)
		
		return c.JSON(fiber.Map{
			"menus": []interface{}{},
		})
	}

	log.Printf("Found role mapping with %d menu IDs", len(roleMenuMapping.MenuIds))

	// Get the actual menu details
	menuCollection := config.GetCollection("menus")
	cursor, err := menuCollection.Find(context.Background(), bson.M{
		"_id":      bson.M{"$in": roleMenuMapping.MenuIds},
		"isActive": true,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch menus"})
	}

	var menus []models.Menu
	if err = cursor.All(context.Background(), &menus); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to decode menus"})
	}

	// Build hierarchical menu structure
	menuTree := buildMenuTree(menus)

	return c.JSON(fiber.Map{
		"menus": menuTree,
	})
}

// buildMenuTree creates a hierarchical menu structure with parent-child relationships
func buildMenuTree(menus []models.Menu) []models.MenuWithChildren {
	// Create a map for quick lookup
	menuMap := make(map[primitive.ObjectID]*models.MenuWithChildren)
	var rootMenus []models.MenuWithChildren

	// Initialize all menus in the map
	for _, menu := range menus {
		menuWithChildren := models.MenuWithChildren{
			Menu:     menu,
			Children: []models.MenuWithChildren{},
		}
		menuMap[menu.ID] = &menuWithChildren
	}

	// Build the tree structure
	for _, menu := range menus {
		if menu.ParentID != nil && *menu.ParentID != primitive.NilObjectID {
			// This is a child menu
			if parent, exists := menuMap[*menu.ParentID]; exists {
				parent.Children = append(parent.Children, *menuMap[menu.ID])
			}
		} else {
			// This is a root menu
			rootMenus = append(rootMenus, *menuMap[menu.ID])
		}
	}

	// Sort menus by name for consistent ordering
	sort.Slice(rootMenus, func(i, j int) bool {
		return rootMenus[i].Name < rootMenus[j].Name
	})

	for i := range rootMenus {
		sort.Slice(rootMenus[i].Children, func(a, b int) bool {
			return rootMenus[i].Children[a].Name < rootMenus[i].Children[b].Name
		})
	}

	return rootMenus
}
