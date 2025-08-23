package utils

import (
	"context"
	"log"
	"mdp-project-backend/config"
	"mdp-project-backend/models"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// logActivity logs user activity to the database
func LogActivity(userID primitive.ObjectID, username, action, ipAddress, userAgent string) {
	activity := models.ActivityLog{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		Username:  username,
		Action:    action,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Timestamp: time.Now(),
	}

	collection := config.GetCollection("activity_logs")
	_, err := collection.InsertOne(context.Background(), activity)
	if err != nil {
		log.Printf("Failed to log activity: %v", err)
	}
}
