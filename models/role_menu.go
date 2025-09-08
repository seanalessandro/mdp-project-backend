package models

import "time"

// RoleMenuMapping mendefinisikan pemetaan antara sebuah role dan menu.
// Digunakan untuk mengontrol menu mana yang bisa diakses oleh role tertentu.
type RoleMenuMapping struct {
	ID         string    `json:"id" bson:"_id,omitempty"`
	RoleID     string    `json:"roleId" bson:"roleId"`
	MenuID     string    `json:"menuId" bson:"menuId"`
	IsActive   bool      `json:"isActive" bson:"isActive"`
	CreatedOn  time.Time `json:"createdOn" bson:"createdOn"`
	CreatedBy  *string   `json:"createdBy" bson:"createdBy,omitempty"`
	ModifiedOn time.Time `json:"modifiedOn" bson:"modifiedOn"`
	ModifiedBy *string   `json:"modifiedBy" bson:"modifiedBy,omitempty"`
}

// RoleMenuMappingRequest digunakan untuk validasi body request saat membuat pemetaan.
type RoleMenuMappingRequest struct {
	RoleID string `json:"roleId" validate:"required"`
	MenuID string `json:"menuId" validate:"required"`
}
