package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ActivityLog represents a user activity log entry
type ActivityLog struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"userId" bson:"userId"`
	Username  string             `json:"username" bson:"username"`
	Action    string             `json:"action" bson:"action"`
	IPAddress string             `json:"ipAddress" bson:"ipAddress"`
	UserAgent string             `json:"userAgent" bson:"userAgent"`
	Timestamp time.Time          `json:"timestamp" bson:"timestamp"`
}
