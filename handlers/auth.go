package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"liven-one-go/models"
	"liven-one-go/utils"
	"net/http"
)

var DB *gorm.DB

// RegisterRequest struct to bind registration data
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=20"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=20"`
}

func AuthHandler(context *gin.Context) {
	// Inject DB here
	if DB == nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not initialized"})
		return
	}

	switch context.Request.URL.Path {
	case "/auth/register":
		register(context)
	case "/auth/login":
		login(context)
	default:
		context.JSON(http.StatusNotFound, gin.H{"error": "Route not found hehe"})
	}
}

func register(context *gin.Context) {
	var req RegisterRequest
	if err := context.ShouldBindJSON(&req); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user with the email already exists
	var existingUser models.User
	queryResult := DB.Where("email = ?", req.Email).First(&existingUser)

	if queryResult.Error == nil {
		// No error means user was found. Email is already registered.
		context.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	// If there was an error, check if it was "record not found"
	if queryResult.Error != gorm.ErrRecordNotFound {
		context.JSON(http.StatusInternalServerError, gin.H{"error": queryResult.Error.Error()})
		return
	}

	// Create a new user
	user := models.User{Email: req.Email}

	// Try hashing password
	if err := user.HashPassword(req.Password); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Insert into database
	if err := DB.Create(&user).Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "User registered successfully", "user_id": user.ID})
}

func login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find the user by email
	var user models.User
	if err := DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check the password
	if err := user.CheckPassword(req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is missing"})
			return
		}

		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Optionally, you can fetch the user from the database using claims.UserID
		c.Set("user_id", claims.UserID) // Store user ID in context if needed
		c.Next()
	}
}

// ProtectedHandler Example protected route
func ProtectedHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Protected resource accessed", "user_id": userID})
}
