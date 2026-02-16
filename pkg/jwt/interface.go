package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Validator handles JWT token validation
type Validator struct {
	secretKey []byte
}

// NewValidator creates a new JWT validator
func NewValidator(cfg Config) *Validator {
	return &Validator{
		secretKey: []byte(cfg.SecretKey),
	}
}

// ValidateToken validates a JWT token and extracts claims
func (v *Validator) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return v.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims format")
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return nil, fmt.Errorf("missing or invalid 'sub' claim")
	}

	email, _ := claims["email"].(string)

	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'exp' claim")
	}

	if time.Now().Unix() > int64(exp) {
		return nil, fmt.Errorf("token expired")
	}

	return &Claims{
		Sub:   sub,
		Email: email,
		Exp:   int64(exp),
	}, nil
}

// ExtractUserID is a convenience method to extract just the user ID
func (v *Validator) ExtractUserID(tokenString string) (string, error) {
	claims, err := v.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.Sub, nil
}
