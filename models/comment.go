package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Comment merepresentasikan utas komentar utama pada sebuah dokumen.
type Comment struct {
	BaseModel  `json:",inline" bson:",inline"`
	DocumentID primitive.ObjectID `json:"documentId" bson:"documentId"`
	AuthorID   primitive.ObjectID `json:"authorId" bson:"authorId"`
	Content    string             `json:"content" bson:"content"`
	MarkedText string             `json:"markedText" bson:"markedText"` // Teks yang di-highlight
	Replies    []Reply            `json:"replies" bson:"replies"`
}

// Reply merepresentasikan balasan di dalam sebuah utas komentar.
type Reply struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	AuthorID  primitive.ObjectID `json:"authorId" bson:"authorId"`
	Content   string             `json:"content" bson:"content"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
}
