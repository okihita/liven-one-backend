package main

import (
	"liven-one-go/handlers"
	"liven-one-go/models"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite" // Or your preferred database driver
	"gorm.io/gorm"
)

func main() {

	/* DATABASE SETUP STARTS */
	// Initialize database (replace with your actual database setup)
	db, openDbErr := gorm.Open(sqlite.Open("liven.db"), &gorm.Config{})
	if openDbErr != nil {
		log.Fatalf("Failed to connect to database: %v", openDbErr)
		os.Exit(1)
	}
	handlers.DB = db

	migrateErr := db.AutoMigrate(&models.User{}, &models.Venue{})
	if migrateErr != nil {
		log.Fatalf("Failed to migrate database: %v", openDbErr)
	}
	/* DATABASE SETUP ENDS */

	/* ROUTING STARTS */
	router := gin.Default()

	// Authentication routes
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", handlers.AuthHandler)
		authGroup.POST("/login", handlers.AuthHandler)
	}

	// For diners and public listing
	router.GET("/venues", handlers.ListVenuesHandler)

	merchantRoutes := router.Group("/merchant")
	{
		merchantRoutes.GET("", handlers.AuthMiddleware(), handlers.MerchantAccountHandler)
		merchantVenueRoutes := merchantRoutes.Group("/venues")
		merchantVenueRoutes.Use(handlers.AuthMiddleware())
		{
			merchantVenueRoutes.POST("", handlers.CreateVenueHandler)
			merchantVenueRoutes.GET("", handlers.GetMerchantVenuesHandler)
			merchantVenueRoutes.GET("/:id", handlers.GetVenueHandler)
			merchantVenueRoutes.PUT("/:id", handlers.UpdateVenueHandler)
			merchantVenueRoutes.DELETE("/:id", handlers.DeleteVenueHandler)
		}
	}

	router.GET("/diners", handlers.AuthMiddleware(), handlers.DinerAccountHandler)

	/* ROUTING ENDS */

	port := ":8080"
	log.Printf("Server listening on port %s", port)
	if err := router.Run(port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
		os.Exit(1)
	}
}
