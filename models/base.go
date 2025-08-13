package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BaseModel struct {
	ID         primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	CreatedOn  time.Time           `json:"createdOn" bson:"createdOn"`
	CreatedBy  *primitive.ObjectID `json:"createdBy,omitempty" bson:"createdBy,omitempty"`
	ModifiedOn time.Time           `json:"modifiedOn" bson:"modifiedOn"`
	ModifiedBy *primitive.ObjectID `json:"modifiedBy,omitempty" bson:"modifiedBy,omitempty"`
}
