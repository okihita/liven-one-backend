package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"liven-one-go/models"
	"liven-one-go/utils"
	"log"
	"net/http"
)

// CreateVenueRequest defines the request body (JSON) for creating a new venue
type CreateVenueRequest struct {
	Name        string `json:"name" binding:"required"`
	Address     string `json:"address" binding:"required"`
	Description string `json:"description" binding:"required"`
	CuisineType string `json:"cuisine_type" binding:"required"`
}

type UpdateVenueRequest struct {
	Name        string `json:"name"`
	Address     string `json:"address"`
	LatLong     string `json:"lat_long"`
	Description string `json:"description"`
	CuisineType string `json:"cuisine_type"`
}

func CreateVenueHandler(c *gin.Context) {
	// DB is already a global value inside this module. Extract?
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not initialized"})
		return
	}

	var request CreateVenueRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get authenticated user details from context
	userClaimsInterface, _ := c.Get(UserClaimsHandlerKey)
	if userClaimsInterface == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication details not found"})
		return
	}

	userClaims := userClaimsInterface.(*utils.Claims)
	if userClaims.UserType != models.UserTypeMerchant {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only a Merchant can add a venue"})
		return
	}

	venue := models.Venue{
		Name:        request.Name,
		Address:     request.Address,
		Description: request.Description,
		CuisineType: request.CuisineType,
		MerchantID:  userClaims.UserID,
	}

	if err := DB.Create(&venue).Error; err != nil {
		log.Printf("Failed to create venue %v: %v", venue, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create venue: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"venue": venue})
}

func GetMerchantVenuesHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not initialized"})
		return
	}

	userClaimsInterface, _ := c.Get(UserClaimsHandlerKey)
	if userClaimsInterface == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication details not found"})
		return
	}

	userClaims := userClaimsInterface.(*utils.Claims)
	if userClaims.UserType != models.UserTypeMerchant {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access forbidden. Only a Merchant can get venues."})
		return
	}

	var venues []models.Venue
	if err := DB.Where("merchant_id = ?", userClaims.UserID).Find(&venues).Error; err != nil {
		log.Printf("Failed to get venues for user %v: %v", userClaims.UserID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get venues: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"venues": venues})
}

// GetVenueHandler assumes a merchant will only manage one venue
func GetVenueHandler(c *gin.Context) {

	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not initialized"})
		return
	}

	venueId := c.Param("venue_id")

	userClaimsInterface, _ := c.Get(UserClaimsHandlerKey)
	if userClaimsInterface == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication details not found"})
		return
	}

	var venue models.Venue
	if err := DB.Where("id = ?", venueId).First(&venue).Error; err != nil {

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Venue not found"})
			return
		}

		log.Printf("Failed to get venue %v: %v", venueId, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get venue: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"venue": venue})

}

func UpdateVenueHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not initialized"})
		return
	}

	venueId := c.Param("venue_id")

	var request UpdateVenueRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON error: " + err.Error()})
		return
	}

	userClaimsInterface, _ := c.Get(UserClaimsHandlerKey)
	if userClaimsInterface == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication details not found"})
		return
	}

	userClaims := userClaimsInterface.(*utils.Claims)
	if userClaims.UserType != models.UserTypeMerchant {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only merchant can update venues."})
		return
	}

	var venue models.Venue
	if err := DB.First(&venue, venueId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Venue not found"})
			return
		}
		log.Printf("Failed to get venue %v: %v", venueId, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get venue: " + err.Error()})
		return
	}
	if venue.MerchantID != userClaims.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't own this venue"})
		return
	}

	updateData := models.Venue{
		Name:        request.Name,
		Address:     request.Address,
		Description: request.Description,
		CuisineType: request.CuisineType,
	}

	if err := DB.Model(&venue).Updates(updateData).Error; err != nil {
		log.Printf("Failed to update venue %v: %v", venueId, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update venue: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"venue": venue})
}

func DeleteVenueHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not initialized"})
		return
	}

	venueId := c.Param("venue_id")
	userClaimsInterface, _ := c.Get(UserClaimsHandlerKey)
	if userClaimsInterface == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication details not found"})
		return
	}

	userClaims := userClaimsInterface.(*utils.Claims)
	if userClaims.UserType != models.UserTypeMerchant {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only merchant can delete venues."})
		return
	}

	var venue models.Venue
	if err := DB.Where("id = ?", venueId).First(&venue).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Venue not found"})
			return
		}
		log.Printf("Failed to get venue %v: %v", venueId, err)
	}

	if venue.MerchantID != userClaims.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't own this venue"})
		return
	}

	if err := DB.Delete(&venue).Error; err != nil {
		log.Printf("Failed to delete venue %v: %v", venueId, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete venue: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Venue deleted successfully"})
}

func ListVenuesHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not initialized"})
		return
	}

	var venues []models.Venue
	query := DB.Model(&models.Venue{})

	// Simple search by name, case-insensitive partial match
	if nameQuery := c.Query("name"); nameQuery != "" {
		query = query.Where("LOWER(name) LIKE LOWER(?)", "%"+nameQuery+"%")
	}

	if cuisineQuery := c.Query("cuisine"); cuisineQuery != "" {
		query = query.Where("LOWER(cuisine) LIKE LOWER(?)", "%"+cuisineQuery+"%")
	}

	if err := query.Find(&venues).Error; err != nil {
		log.Printf("Failed to list venues: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list venues: " + err.Error()})
		return
	}

	if venues == nil {
		venues = []models.Venue{}
	}

	c.JSON(http.StatusOK, gin.H{"venues": venues})
}
