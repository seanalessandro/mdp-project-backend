// models/approval_history.go
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ApprovalHistoryEntry represents an immutable record of an approval action.
type ApprovalHistoryEntry struct {
	ID         primitive.ObjectID `json:"id" bson:"_id"`
	DocumentID primitive.ObjectID `json:"documentId" bson:"documentId"`
	Action     string             `json:"action" bson:"action"` // e.g., "approved", "rejected", "submitted"
	Level      int                `json:"level" bson:"level"`
	RoleName   string             `json:"roleName" bson:"roleName"`
	UserID     primitive.ObjectID `json:"userId" bson:"userId"`
	Username   string             `json:"username" bson:"username"`
	Timestamp  time.Time          `json:"timestamp" bson:"timestamp"`
	PrevStatus string             `json:"prevStatus" bson:"prevStatus"`
	NewStatus  string             `json:"newStatus" bson:"newStatus"`
	Comments   string             `json:"comments,omitempty" bson:"comments,omitempty"` // Optional comments
}
