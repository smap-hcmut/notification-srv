package main

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	// Secret key (must match JWT_SECRET_KEY in .env)
	secretKey := "my-super-secret-jwt-key-for-testing-change-in-production"

	// Create claims
	claims := jwt.MapClaims{
		"sub":   "user123",                             // User ID
		"email": "test@example.com",                    // Optional
		"exp":   time.Now().Add(24 * time.Hour).Unix(), // Expires in 24 hours
		"iat":   time.Now().Unix(),
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secretKey))

	fmt.Println("Generated JWT Token:")
	fmt.Println(tokenString)
	fmt.Println("\nTo test WebSocket connection:")
	fmt.Printf("go run tests/client_example.go %s\n", tokenString)
	fmt.Println("\nOr in browser/Postman:")
	fmt.Printf("ws://localhost:8081/ws?token=%s\n", tokenString)
}
