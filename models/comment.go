package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Hapus struct Reply jika masih ada, karena sudah tidak digunakan.

// Definisikan ulang struct Comment agar memiliki ParentID
type Comment struct {
	BaseModel  `json:",inline" bson:",inline"`
	DocumentID primitive.ObjectID `json:"documentId" bson:"documentId"`

	ParentID   *primitive.ObjectID `json:"parentId,omitempty" bson:"parentId,omitempty"`
	AuthorID   primitive.ObjectID  `json:"authorId" bson:"authorId"`
	Content    string              `json:"content" bson:"content"`
	MarkedText string              `json:"markedText,omitempty" bson:"markedText,omitempty"` // Jadikan opsional
}
