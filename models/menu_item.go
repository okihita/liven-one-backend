package models

import (
	"gorm.io/gorm"
)

type MenuItem struct {
	gorm.Model
	Name         string `json:"name" gorm:"not null"`
	Description  string `json:"description" `
	PriceInCents int64  `json:"price_in_cents"`
	Category     string `json:"category" gorm:"index"`
	VenueId      uint   `json:"venue_id" gorm:"not null"`
	Venue        Venue  `json:"-"`
}
