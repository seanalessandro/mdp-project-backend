package models

// Role mendefinisikan sebuah peran dalam sistem.
type Role struct {
	Name        string   `json:"name" bson:"name"`               // Nama peran, harus unik. e.g., "Admin", "Editor"
	Description string   `json:"description" bson:"description"` // Penjelasan singkat tentang peran
	IsActive    bool     `json:"isActive" bson:"isActive"`       // Status aktif atau tidaknya sebuah peran
	IsDefault   bool     `json:"isDefault" bson:"isDefault"`     // Menandai role default yang tidak bisa dihapus
	Permissions []string `json:"permissions" bson:"permissions"` // Daftar hak akses, e.g., ["user:create", "product:read"]
	BaseModel   `json:",inline" bson:",inline"`
}

// RoleRequest adalah struct untuk validasi body request saat membuat/update role.
type RoleRequest struct {
	Name        string   `json:"name" validate:"required"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}
