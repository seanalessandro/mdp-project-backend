package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// DocumentVersion menyimpan snapshot dari sebuah dokumen pada waktu tertentu.
type DocumentVersion struct {
	BaseModel         `json:",inline" bson:",inline"`
	DocumentID        primitive.ObjectID `json:"documentId" bson:"documentId"`               // ID dokumen asli
	Version           float64            `json:"version" bson:"version"`                     // <-- Ganti dari string
	Content           string             `json:"content" bson:"content"`                     // Salinan lengkap konten Tiptap JSON
	ChangeDescription string             `json:"changeDescription" bson:"changeDescription"` // Catatan pemicu, e.g., "Submitted for Review"
}
