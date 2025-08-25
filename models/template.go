package models

type DocumentTemplate struct {
	BaseModel    `json:",inline" bson:",inline"`
	Name         string `json:"name" bson:"name"`                 // e.g., "Blank Document", "Template A"
	Description  string `json:"description" bson:"description"`   // e.g., "Silakan pilih template..."
	ThumbnailURL string `json:"thumbnailUrl" bson:"thumbnailUrl"` // URL gambar thumbnail
	Content      string `json:"content" bson:"content"`           // Konten awal Tiptap dalam format JSON string
}
