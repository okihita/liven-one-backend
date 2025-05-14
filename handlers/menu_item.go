package handlers

import (
	"gorm.io/gorm"
	"liven-one-go/models"
	"liven-one-go/utils"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CreateMenuItemRequest struct {
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description" binding:"required"`
	PriceInCents uint   `json:"price_in_cents" binding:"required,gt=0"`
	Category     string `json:"category" binding:"required"`
}

type UpdateMenuItemRequest struct {
	Name         *string `json:"name" binding:"required"`
	Description  *string `json:"description" binding:"required"`
	PriceInCents *uint   `json:"price_in_cents" binding:"required,gt=0"`
	Category     *string `json:"category" binding:"required"`
}

func CheckVenueOwnership(c *gin.Context, venueIdString string) (*models.Venue, bool) {

	if DB == nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Database not initialized"})
		return nil, false
	}

	userClaimsInterface, _ := c.Get("user_claims")
	if userClaimsInterface == nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "User not recognized"})
		return nil, false
	}
	userClaims := userClaimsInterface.(*utils.Claims)

	var venue models.Venue
	if err := DB.First(&venue, venueIdString).Error; err != nil {

		if err == gorm.ErrRecordNotFound {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Venue not found"})
			return nil, false
		}

		log.Println(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Venue not found"})
		return nil, false
	}

	if venue.MerchantID != userClaims.UserID {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "You don't own this venue"})
		return nil, false
	}

	return &venue, true
}

func CreateMenuItemHandler(c *gin.Context) {
	venueIdString := c.Param("venue_id")
	venue, owned := CheckVenueOwnership(c, venueIdString)
	if !owned {
		return // Error response already sent by CheckVenueOwnership
	}

	var request CreateMenuItemRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// menuItem variable holds the memory address of this newly-created object
	// So that GROM can directly modify the object and not merely modify a copy of the object.
	menuItem := &models.MenuItem{
		Name:         request.Name,
		Description:  request.Description,
		PriceInCents: request.PriceInCents,
		Category:     request.Category,
		VenueId:      venue.ID,
	}

	if err := DB.Create(&menuItem).Error; err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, menuItem)
}

func GetMenuItemsForVenueHandler(c *gin.Context) {
	venueIdString := c.Param("venue_id")
	venue, owned := CheckVenueOwnership(c, venueIdString)
	if !owned {
		return
	}

	var menuItems []models.MenuItem
	if err := DB.Where("venue_id = ?", venue.ID).Find(&menuItems).Error; err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get menu items: " + err.Error()})
		return
	}

	if menuItems == nil {
		menuItems = []models.MenuItem{}
	}

	c.JSON(http.StatusOK, menuItems)
}

// Path: merchant/venue/:venue_id/item/:item_id
func UpdateMenuItemHandler(c *gin.Context) {

	venueIdString := c.Param("venue_id")
	itemIdString := c.Param("item_id")

	venue, owned := CheckVenueOwnership(c, venueIdString)

	if !owned {
		return
	}

	var request UpdateMenuItemRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var menuItem models.MenuItem
	if err := DB.Where("id = ? AND venue_id = ?", itemIdString, venue.ID).First(&menuItem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Menu item not found"})
			return
		}

		log.Println(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Menu item not found"})
		return
	}

	// Build map for updates to handle partial updates correctly with pointers
	updates := make(map[string]interface{})

	if request.Name != nil {
		updates["name"] = *request.Name
	}

	if request.Description != nil {
		updates["description"] = *request.Description
	}

	if request.PriceInCents != nil {
		updates["price_in_cents"] = *request.PriceInCents
	}

	if request.Category != nil {
		updates["category"] = *request.Category
	}

	if len(updates) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "No update fields provided"})
		return
	}

	if err := DB.Model(&menuItem).Updates(updates).Error; err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, menuItem)
}

func DeleteMenuItemHandler(c *gin.Context) {

	venueIdString := c.Param("venue_id")
	itemIdString := c.Param("item_id")

	venue, owned := CheckVenueOwnership(c, venueIdString)
	if !owned {
		return
	}

	var menuItem models.MenuItem
	if err := DB.Where("id = ? AND venue_id = ?", itemIdString, venue.ID).First(&menuItem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Menu item not found"})
			return
		}

		log.Println(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Menu item not found"})
		return
	}

	if err := DB.Delete(&menuItem).Error; err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete menu item: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deleted menu item"})
}

func GetVenueMenuForDinersHandler(c *gin.Context) {
	if DB == nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Database not initialized"})
		return
	}

	venueIdString := c.Param("venue_id")

	var venue models.Venue
	if err := DB.Where("id = ?", venueIdString).First(&venue).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Venue not found"})
			return
		}
		log.Println("Failed retrieving Venue from DB", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed retrieving Venue: " + err.Error()})

		return
	}

	var menuItems []models.MenuItem

	if err := DB.Where("venue_id = ?", venueIdString).Find(&menuItems).Error; err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get menu items"})
		return
	}

	if menuItems == nil {
		menuItems = []models.MenuItem{}
	}

	c.JSON(http.StatusOK, menuItems)
}
