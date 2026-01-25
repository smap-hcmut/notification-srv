package jwt

// Config holds JWT configuration
type Config struct {
	SecretKey string
}

// Claims represents the JWT claims structure
type Claims struct {
	Sub   string `json:"sub"`   // User ID
	Email string `json:"email"` // User email
	Exp   int64  `json:"exp"`   // Expiration time
}
