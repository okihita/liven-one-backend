package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"liven-one-go/models"
	"liven-one-go/utils"
	"net/http"
	"strings"
)

var DB *gorm.DB

// RegisterRequest struct to bind registration data
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=20"`
	UserType string `json:"user_type" binding:"required,oneof=diner merchant"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
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
		context.JSON(http.StatusNotFound, gin.H{"error": "Route not found"})
	}
}

func register(context *gin.Context) {
	var req RegisterRequest
	if err := context.ShouldBindJSON(&req); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate UserType (handled by binding:"oneof=diner merchant" but better if explicit)
	if req.UserType != "diner" && req.UserType != "merchant" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user type. Must be \"diner\" or \"merchant\""})
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
	user := models.User{
		Email:    req.Email,
		UserType: req.UserType,
	}

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

	type RegisterResponse struct {
		ID       uint   `json:"id"`
		Email    string `json:"email"`
		UserType string `json:"user_type"`
	}

	response := RegisterResponse{
		ID:       user.ID,
		Email:    user.Email,
		UserType: req.UserType,
	}

	context.JSON(http.StatusCreated, gin.H{"message": "User registered successfully", "user": response})
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
	token, err := utils.GenerateToken(user.ID, user.UserType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// AuthMiddleware checks authorization and token status, ensuring it's still valid and not tampered.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is missing"})
			return
		}

		const bearerPrefix = "Bearer "
		var tokenString string

		// Strips the "Bearer " prefix in token auth header
		if strings.HasPrefix(authHeader, bearerPrefix) {
			tokenString = strings.TrimPrefix(authHeader, bearerPrefix)
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is invalid. Ensure it starts with bearer prefix."})
		}

		// Check if token string empty after stripping
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token is missing"})
			return
		}

		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
			return
		}

		c.Set("user_claims", claims)
		c.Next()
	}
}

// MerchantAccountHandler Example protected route
func MerchantAccountHandler(c *gin.Context) {
	claimsInterface, userExists := c.Get("user_claims")

	if !userExists {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "User identification not found"})
		return
	}

	userClaims, ok := claimsInterface.(*utils.Claims)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid claims type in context"})
		return
	}

	if userClaims.UserType != models.UserTypeMerchant {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user type. Must be \"" + models.UserTypeMerchant + "\""})
		return
	}

	c.JSON(http.StatusOK, userClaims)
}

func DinerAccountHandler(c *gin.Context) {
	claimsInterface, userExists := c.Get("user_claims")

	if !userExists {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "User identification not found"})
		return
	}

	userClaims, ok := claimsInterface.(*utils.Claims)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid claims type in context"})
		return
	}
	if userClaims.UserType != models.UserTypeDiner {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user type. Must be \"" + models.UserTypeDiner + "\""})
		return
	}

	c.JSON(http.StatusOK, userClaims)
}
