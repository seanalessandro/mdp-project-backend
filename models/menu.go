package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Menu mendefinisikan sebuah item menu dalam sistem.
type Menu struct {
	Name      string              `json:"name" bson:"name"`                             // Nama menu
	Path      string              `json:"path" bson:"path"`                             // Path atau URL menu
	Icon      string              `json:"icon" bson:"icon"`                             // Nama ikon menu
	IsActive  bool                `json:"isActive" bson:"isActive"`                     // Status aktif menu
	ParentID  *primitive.ObjectID `json:"parentId,omitempty" bson:"parentId,omitempty"` // ID menu induk (untuk sub-menu)
	BaseModel `json:",inline" bson:",inline"`
}

// MenuRequest adalah struct untuk validasi body request saat membuat/update menu.
type MenuRequest struct {
	Name     string `json:"name" validate:"required"`
	Path     string `json:"path" validate:"required"`
	Icon     string `json:"icon"`
	IsActive *bool  `json:"isActive"` // Pointer agar bisa membedakan antara false dan tidak dikirim
	ParentID string `json:"parentId"`
}

// MenuWithChildren represents a menu item with its children for hierarchical display
type MenuWithChildren struct {
	Menu     `json:",inline" bson:",inline"`
	Children []MenuWithChildren `json:"children,omitempty"`
}
