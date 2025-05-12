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
	// Initialize database (replace with your actual database setup)
	db, err := gorm.Open(sqlite.Open("user.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
		os.Exit(1)
	}
	handlers.DB = db
	db.AutoMigrate(&models.User{})

	router := gin.Default()

	// Authentication routes
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", handlers.AuthHandler)
		authGroup.POST("/login", handlers.AuthHandler)
	}

	// Protected route example
	router.GET("/protected", handlers.AuthMiddleware(), handlers.ProtectedHandler)

	port := ":8080"
	log.Printf("Server listening on port %s", port)
	if err := router.Run(port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
		os.Exit(1)
	}
}
