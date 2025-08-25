package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Document merepresentasikan sebuah dokumen teks di database.
type Document struct {
	BaseModel `json:",inline" bson:",inline"` // Menyematkan ID, CreatedOn, CreatedBy, dll.
	Title     string                          `json:"title" bson:"title"`     // Judul dokumen
	Content   string                          `json:"content" bson:"content"` // Konten dokumen dalam format JSON string dari Lexical
	OwnerID   primitive.ObjectID              `json:"ownerId" bson:"ownerId"` // ID pengguna yang memiliki dokumen
	Status    string                          `json:"status" bson:"status"`
	DocNo     string                          `json:"docNo" bson:"docNo"`
	Version   float64                         `json:"version" bson:"version"` // <-- Ganti dari string
	Priority  string                          `json:"priority" bson:"priority"`
}
