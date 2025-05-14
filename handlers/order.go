package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"liven-one-go/models"
	"liven-one-go/utils"
	"log"
	"net/http"
	"time"
)

// OrderItemRequest is part of PlaceOrderRequest
type OrderItemRequest struct {
	MenuItemID uint  `json:"menu_item_id" binding:"required"`
	Quantity   int64 `json:"quantity" binding:"required,gt=0"`
}

// PlaceOrderRequest defines the request body (JSON) for a diner placing an order
type PlaceOrderRequest struct {
	VenueID uint               `json:"venue_id" binding:"required"`
	Items   []OrderItemRequest `json:"items" binding:"required,min=1"`
}

// UpdateOrderStatusRequest defines the request body for a merchant updating an order request
type UpdateOrderStatusRequest struct {
	Status models.OrderStatus `json:"status" binding:"required"`
}

type OrderResponse struct {
	models.Order
}

// PlaceOrderHandler handles a diner placing a new order
func PlaceOrderHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database connection failed"})
		return
	}

	var req PlaceOrderRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userClaimsInterface, _ := c.Get(UserClaimsHandlerKey)
	if userClaimsInterface == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User isn't authorized"})
		return
	}

	userClaims := userClaimsInterface.(*utils.Claims)
	if userClaims.UserType != models.UserTypeDiner {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only diners can place orders"})
		return
	}

	// --- Transaction for order creation
	tx := DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": tx.Error.Error()})
		return
	}

	// Defer a rollback in case of panic or error
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback() // Handle error
		} else if tx.Error != nil {
			tx.Rollback() // Check for GORM errors on tx
		}
	}()

	// 1. Validate Venue
	var venue models.Venue
	if err := tx.First(&venue, req.VenueID).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Venue not found"})
			return
		}
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": tx.Error.Error()})
		return
	}

	// 2. Process Order Items and Calculate Total Amount
	var orderItems []models.OrderItem
	var calculatedTotalAmountInCents int64 = 0

	menuItemIDs := []uint{}
	for _, menuItem := range req.Items {
		menuItemIDs = append(menuItemIDs, menuItem.MenuItemID)
	}

	var menuItemsFromDB []models.MenuItem
	// Fetch all menu items at once to reduce DB calls and check they belong to the venue
	if err := tx.Where("id IN ? AND venue_id = ?", menuItemIDs, venue.ID).Find(&menuItemsFromDB).Error; err != nil {
		tx.Rollback()
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": tx.Error.Error()})
		return
	}

	// Create a map for quick lookup of fetched menu items
	menuItemMap := make(map[uint]models.MenuItem)
	for _, menuItem := range menuItemsFromDB {
		menuItemMap[menuItem.ID] = menuItem
	}

	for _, orderItem := range req.Items {
		menuItem, exists := menuItemMap[orderItem.MenuItemID]
		if !exists {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid menu item ID, or item not found in this venue"})
			return
		}

		if orderItem.Quantity <= 0 {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quantity"})
			return
		}

		orderItem := models.OrderItem{
			MenuItemID:          menuItem.ID,
			Quantity:            orderItem.Quantity,
			PriceInCentsAtOrder: menuItem.PriceInCents,
		}
		orderItems = append(orderItems, orderItem)
		calculatedTotalAmountInCents += menuItem.PriceInCents * orderItem.Quantity
	}

	// 3. Create the Order
	order := models.Order{
		DinerID:            userClaims.UserID,
		VenueID:            venue.ID,
		TotalAmountInCents: calculatedTotalAmountInCents,
		Status:             models.OrderStatusPending,
		OrderTimestamp:     time.Now(),
		OrderItems:         orderItems,
	}

	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": tx.Error.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": tx.Error.Error()})
		return
	}

	var createdOrderWithDetails models.Order
	if err := DB.Preload("OrderItems.MenuItem").Preload("Diner").Preload("Venue").First(&createdOrderWithDetails, order.ID).Error; err != nil {
		log.Println(err)
		c.JSON(http.StatusOK, order)
		return
	}

	c.JSON(http.StatusOK, createdOrderWithDetails)

}

func GetMerchantOrdersHandler(c *gin.Context) {
	venueIDStr := c.Param("venue_id")
	venue, owned := CheckVenueOwnership(c, venueIDStr)
	if !owned {
		return
	}

	statusFilter := c.Query("status")

	var orders []models.Order
	query := DB.Where("venue_id = ?", venue.ID)
	if statusFilter != "" {
		query = query.Where("status = ?", models.OrderStatus(statusFilter))
	}

	// Preload related data for merchant view
	if err := query.
		Preload("OrderItems.MenuItem").Preload("Diner").
		Order("created_at DESC").Find(&orders).Error; err != nil {
		log.Printf("Failed to get orders from venue %d: %v\n", venue.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if orders == nil {
		orders = []models.Order{}
	}

	c.JSON(http.StatusOK, orders)
}

func UpdateOrderStatusHandler(c *gin.Context) {
	orderIDStr := c.Param("order_id")

	var request UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the status from the request
	switch request.Status {

	case
		models.OrderStatusPending,
		models.OrderStatusRejected,
		models.OrderStatusAccepted,
		models.OrderStatusCancelled,
		models.OrderStatusPreparing,
		models.OrderStatusReadyForDelivery,
		models.OrderStatusCompleted:
		// Do nothing. Go to the next blocks of code.
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status value"})
		return
	}

	userClaimsInterface, _ := c.Get(UserClaimsHandlerKey)
	userClaims := userClaimsInterface.(*utils.Claims)
	if userClaimsInterface == nil || userClaims.UserType != models.UserTypeMerchant {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Access forbidden: Only merchants can update order status."})
		return
	}

	var order models.Order
	if err := DB.
		Joins("JOIN venues ON venues.id = orders.venue_id AND venues.merchant_id = ?", userClaims.UserID).
		Find(&order, orderIDStr).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}

		log.Printf("Failed to get order: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Basic state transition validation (can be more complex)
	// For MVP, we might allow most transitions.
	// Example: if order.Status == models.OrderStatusCompleted && req.Status != models.OrderStatusCompleted {
	//    c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot change status of a completed order"})
	//    return
	// }

	if err := DB.Model(&order).Update("status", request.Status).Error; err != nil {
		log.Printf("Failed to update order: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var updatedOrderWithDetails models.Order
	if err := DB.Preload("OrderItems.MenuItem").
		Preload("Diner").Preload("Venue").
		First(&updatedOrderWithDetails, order.ID).Error; err != nil {
		log.Printf("Failed to get order: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedOrderWithDetails)

}

func GetDinerOrdersHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not initialized"})
		return
	}

	userClaimsInterface, _ := c.Get(UserClaimsHandlerKey)
	if userClaimsInterface == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication details not found"})
		return
	}

	userClaims := userClaimsInterface.(*utils.Claims)
	if userClaims.UserType != models.UserTypeDiner {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only diners can view orders here."})
		return
	}

	statusFilter := c.Query("status")

	var orders []models.Order
	query := DB.Where("diner_id = ?", userClaims.UserID)
	if statusFilter != "" {
		query = query.Where("status = ?", models.OrderStatus(statusFilter))
	}

	if err := query.Preload("OrderItems.MenuItem").Preload("Venue").
		Order("created_at DESC").Find(&orders).Error; err != nil {
		log.Printf("Failed to get orders from venue %d: %v\n", userClaims.UserID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if orders == nil {
		orders = []models.Order{}
	}

	c.JSON(http.StatusOK, orders)
}

func GetDinerSingleOrderHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not initialized"})
		return
	}

	orderIDStr := c.Param("order_id")
	userClaimsInterface, _ := c.Get(UserClaimsHandlerKey)
	if userClaimsInterface == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication details not found"})
		return
	}

	userClaims := userClaimsInterface.(*utils.Claims)
	if userClaims.UserType != models.UserTypeDiner {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only diners can view order here."})
	}

	var order models.Order
	if err := DB.Preload("OrderItems.MenuItem").Preload("Venue").
		Where("id = ? AND diner_id", orderIDStr, userClaims.UserID).First(&order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found or you don't have permission to view this order."})
			return
		}

		log.Printf("Failed to get order: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)

}
