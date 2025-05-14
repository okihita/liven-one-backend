package models

import (
	"gorm.io/gorm"
	"time"
)

type OrderStatus string

const (
	OrderStatusPending          OrderStatus = "Pending"
	OrderStatusRejected         OrderStatus = "Rejected" // Rejected by Merchant
	OrderStatusAccepted         OrderStatus = "Accepted"
	OrderStatusCancelled        OrderStatus = "Cancelled"
	OrderStatusPreparing        OrderStatus = "Preparing"
	OrderStatusReadyForDelivery OrderStatus = "ReadyForDelivery"
	OrderStatusCompleted        OrderStatus = "Completed"
)

type Order struct {
	gorm.Model
	DinerID            uint        `json:"diner_id" gorm:"not null"`
	Diner              User        `json:"diner,omitempty" gorm:"foreignKey:DinerID"`
	VenueID            uint        `json:"venue_id" gorm:"not null"`
	Venue              Venue       `json:"venue,omitempty" gorm:"foreignKey:VenueID"`
	OrderItems         []OrderItem `json:"order_items" gorm:"foreignKey:OrderID"`
	TotalAmountInCents int64       `json:"total_amount_in_cents" gorm:"not null"`
	Status             OrderStatus `json:"status" gorm:"not null;index"`
	OrderTimestamp     time.Time   `json:"order_timestamp" gorm:"not null"`
}

type OrderItem struct {
	gorm.Model
	OrderID             uint     `json:"order_id" gorm:"not null;index"`
	MenuItemID          uint     `json:"menu_item_id" gorm:"not null;"`
	MenuItem            MenuItem `json:"menu_item" gorm:"foreignKey:MenuItemID"`
	Quantity            int64    `json:"quantity" gorm:"not null"`
	PriceInCentsAtOrder int64    `json:"price_in_cents_at_order" gorm:"not null"`
}
