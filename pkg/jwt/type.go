package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const defaultTTL = 8 * time.Hour

type Config struct {
	SecretKey string
}

type managerImpl struct {
	secretKey []byte
	issuer    string
	ttl       time.Duration
}

type Claims struct {
	Email  string   `json:"email"`
	Role   string   `json:"role"`
	Groups []string `json:"groups,omitempty"`
	jwt.RegisteredClaims
}
