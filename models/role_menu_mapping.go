package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RoleMenuMapping defines a document that maps a single Role to multiple Menus.
type RoleMenuMapping struct {
	RoleID    primitive.ObjectID   `json:"roleId" bson:"roleId"`
	MenuIDs   []primitive.ObjectID `json:"menuIds" bson:"menuIds"`
	IsActive  bool                 `json:"isActive" bson:"isActive"`
	BaseModel `json:",inline" bson:",inline"`
}

// RoleMenuMappingRequest is the struct for validating the request body.
type RoleMenuMappingRequest struct {
	RoleID   string   `json:"roleId" validate:"required"`
	MenuIDs  []string `json:"menuIds"`
	IsActive *bool    `json:"isActive"`
}
