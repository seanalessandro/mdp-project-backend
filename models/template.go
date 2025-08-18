package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DocumentTemplate merepresentasikan sebuah template dokumen
type DocumentTemplate struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description" bson:"description"`
	Content     string             `json:"content" bson:"content"` // Konten dalam format JSON Tiptap
}
