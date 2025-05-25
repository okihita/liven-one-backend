package main

import (
	"github.com/joho/godotenv"
	"liven-one-go/handlers"
	"liven-one-go/models"
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite" // Or your preferred database driver
	"gorm.io/gorm"
)

func main() {

	/* DATABASE SETUP STARTS */

	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file for database URI. Using environment variables.")
	}

	dbURI := os.Getenv("DATABASE_URI")
	if dbURI == "" {
		dbURI = "test.db"
		log.Println("Warning: DATABASE_URI not found in environment variables. Using default: " + dbURI)
	}

	// Initialize database (replace with your actual database setup)
	db, openDbErr := gorm.Open(sqlite.Open(dbURI), &gorm.Config{})
	if openDbErr != nil {
		log.Fatalf("Failed to connect to database: %v", openDbErr)
		os.Exit(1)
	}
	handlers.DB = db

	migrateErr := db.AutoMigrate(&models.User{}, &models.Venue{}, &models.MenuItem{}, &models.Order{}, &models.OrderItem{})
	if migrateErr != nil {
		log.Fatalf("Failed to migrate database: %v", openDbErr)
	}
	/* DATABASE SETUP ENDS */

	/* ROUTING STARTS */
	router := gin.Default()

	env := os.Getenv("APP_ENV")

	var corsConfig cors.Config
	if env == "debug" || os.Getenv("APP_ENV") == "development" {
		// Development: Allow all origins
		corsConfig = cors.Config{
			AllowOrigins:     []string{"*"}, // Allows all origins
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true, // Be cautious with this in conjunction with AllowOrigins: "*"
			MaxAge:           12 * time.Hour,
		}
	} else {
		// Production: Be specific and secure
		corsConfig = cors.Config{
			AllowOrigins:     []string{"https://your-production-frontend.com"}, // Replace with your actual frontend domain
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}
	}

	router.Use(cors.New(corsConfig))

	// --- Authentication Routes ---
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", handlers.AuthHandler)
		authGroup.POST("/login", handlers.AuthHandler)
	}

	// --- Public/Diner Venue and Menu Routes --- (Auth token not needed)
	publicGroup := router.Group("/public")
	{
		publicGroup.GET("/venues", handlers.ListVenuesHandler)
		publicGroup.GET("/venues/:venue_id", handlers.GetVenueHandler)
		publicGroup.GET("/venues/:venue_id/menu", handlers.GetSingleVenueMenuHandler)
	}

	// --- Diner Protected Routes ---
	dinerRoutes := router.Group("/diner", handlers.AuthMiddleware())
	{
		dinerRoutes.GET("", handlers.DinerAccountHandler)
		orderRoutes := dinerRoutes.Group("/orders")
		{
			orderRoutes.POST("", handlers.PlaceOrderHandler)
			orderRoutes.GET("", handlers.GetDinerOrdersHandler)
			orderRoutes.GET("/:order_id", handlers.GetDinerSingleOrderHandler)
		}
	}

	// --- Merchant Protected Routes ---
	merchantRoutes := router.Group("/merchant", handlers.AuthMiddleware())
	{

		// Account Management
		merchantRoutes.GET("", handlers.MerchantAccountHandler)

		// Merchant Venue Management
		venueRoutes := merchantRoutes.Group("/venues")
		{
			venueRoutes.POST("", handlers.CreateVenueHandler)
			venueRoutes.GET("", handlers.GetSingleMerchantVenuesHandler) // Gets venues for the authenticated Merchant

			venueRoutes.GET("/:venue_id", handlers.GetVenueHandler)
			venueRoutes.PUT("/:venue_id", handlers.UpdateVenueHandler)
			venueRoutes.DELETE("/:venue_id", handlers.DeleteVenueHandler)

			// Merchant Menu Item Management (nested under specific venue)
			menuItemRoutes := venueRoutes.Group("/:venue_id/menuitems")
			{
				menuItemRoutes.POST("", handlers.CreateMenuItemHandler)
				menuItemRoutes.GET("", handlers.GetMenuItemsForVenueHandler)
				menuItemRoutes.PUT("/:item_id", handlers.UpdateMenuItemHandler)
				menuItemRoutes.DELETE("/:item_id", handlers.DeleteMenuItemHandler)
			}

			// Merchant Order Management (for a specific venue they own)
			venueOrderRoutes := venueRoutes.Group("/:venue_id/orders")
			{
				venueOrderRoutes.GET("", handlers.GetMerchantOrdersHandler) // GET /merchant/venues/123/orders
			}
		}

		// Merchant Order Management (venue-agnostic)
		merchantOrderManagementRoutes := merchantRoutes.Group("/orders")
		{
			merchantOrderManagementRoutes.PUT("/:order_id/status", handlers.UpdateOrderStatusHandler)
		}
	}

	/* ROUTING ENDS */

	port := ":8080"
	log.Printf("Server listening on port %s", port)
	if err := router.Run(port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
		os.Exit(1)
	}
}
