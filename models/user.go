package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	Username string `json:"username" bson:"username,unique"`
	Email    string `json:"email" bson:"email,unique"`
	Password string `json:"-" bson:"password,omitempty"`
	// --- PERUBAHAN PENTING ---
	RoleID primitive.ObjectID `json:"roleId" bson:"roleId"`
	// --------------------------
	IsActive   bool       `json:"isActive" bson:"isActive"`
	LastLogin  *time.Time `json:"lastLogin,omitempty" bson:"lastLogin,omitempty"`
	Provider   string     `json:"provider" bson:"provider"`
	ProviderID string     `json:"-" bson:"providerId,omitempty"`
	BaseModel  `json:",inline" bson:",inline"`
}

// Struct untuk response login, agar bisa menyertakan detail role
type LoginResponse struct {
	Token string      `json:"token"`
	User  User        `json:"user"`
	Role  interface{} `json:"role"` // Kita akan sertakan detail role di sini
}

// ... sisa file (LoginRequest, ChangePasswordRequest, dll. tetap sama) ...
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type AdminCreateUserRequest struct {
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	RoleID   string `json:"roleId" validate:"required"` // Diubah ke RoleID
}

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required,min=8"`
}
