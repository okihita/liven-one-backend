package utils

import (
	"log"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"time"
)

var jwtSecret []byte

func init() {
	// Attempt to load .env file. This is useful for local development.
	// In a containerized environment (like Docker/Cloud Run),
	// environment variables are typically set directly by the platform.
	err := godotenv.Load()
	if err != nil {
		// Don't make this fatal. It's okay if .env is not found in a container.
		log.Printf("Warning: .env file not found or error loading: %v. Relying on OS environment variables.", err)
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// This is a critical failure if the secret isn't set by any means
		log.Fatal("FATAL: JWT_SECRET environment variable is not set.")
	}
	jwtSecret = []byte(secret)
}

type Claims struct {
	UserID   uint   `json:"user_id"`
	UserType string `json:"user_type"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uint, userType string) (string, error) {

	claims := Claims{
		UserID:   userID,
		UserType: userType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)), // Token expires in 24 hours
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "GaruruCannonIssuer",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString(jwtSecret)

	if err != nil {
		return "", err
	}

	return ss, nil
}

func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	return claims, nil
}
