package models

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Username     string             `json:"username" bson:"username"`
	Password     string             `json:"-" bson:"password"` // Hidden from JSON
	Role         string             `json:"role" bson:"role"`
	IsActive     bool               `json:"is_active" bson:"is_active"`
	LastLogin    *time.Time         `json:"last_login,omitempty" bson:"last_login,omitempty"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at" bson:"updated_at"`
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type ActivityLog struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	Username  string             `json:"username" bson:"username"`
	Action    string             `json:"action" bson:"action"`
	IPAddress string             `json:"ip_address" bson:"ip_address"`
	UserAgent string             `json:"user_agent" bson:"user_agent"`
	Timestamp time.Time          `json:"timestamp" bson:"timestamp"`
}
