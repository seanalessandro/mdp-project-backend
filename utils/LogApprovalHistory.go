// utils/history_logger.go

package utils

import (
	"context"
	"log"
	"time"

	"mdp-project-backend/config"
	"mdp-project-backend/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// LogApprovalHistory creates a new, immutable history entry in a separate collection.
func LogApprovalHistory(
	docID primitive.ObjectID,
	action string,
	level int,
	roleName string,
	userID primitive.ObjectID,
	username string,
	prevStatus string,
	newStatus string,
	comments string,
) {
	historyCollection := config.GetCollection("approval_histories")

	entry := models.ApprovalHistoryEntry{
		ID:         primitive.NewObjectID(),
		DocumentID: docID,
		Action:     action,
		Level:      level,
		RoleName:   roleName,
		UserID:     userID,
		Username:   username,
		Timestamp:  time.Now(),
		PrevStatus: prevStatus,
		NewStatus:  newStatus,
		Comments:   comments,
	}

	_, err := historyCollection.InsertOne(context.Background(), entry)
	if err != nil {
		log.Printf("Failed to log approval history for document %s: %v", docID.Hex(), err)
	}
}
