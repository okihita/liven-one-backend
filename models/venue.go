package models

import (
	"gorm.io/gorm"
)

type Venue struct {
	gorm.Model         // ID, CreatedAt, UpdatedAt, DeletedAt
	Name        string `json:"name" gorm:"not null, unique"`
	Address     string `json:"address"`
	LatLong     string `json:"lat_long"`
	Description string `json:"description"`
	CuisineType string `json:"cuisine_type"`
	MerchantID  uint   `json:"merchant_id" gorm:"not null"` // Foreign key to the owner (Merchant) account
	Merchant    uint   `json:"-" gorm:"foreignKey:MerchantID"`
}
