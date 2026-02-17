package jwt

import (
	"notification-srv/pkg/scope"
)

type IManager interface {
	GenerateToken(userID, email, role string, groups []string) (string, error)
	VerifyToken(tokenString string) (*Claims, error)
	SetConfig(issuer string)
	Verify(token string) (scope.Payload, error)
	CreateToken(payload scope.Payload) (string, error)
}

func New(cfg Config) (IManager, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	return &managerImpl{
		secretKey: []byte(cfg.SecretKey),
		issuer:    "",
		ttl:       defaultTTL,
	}, nil
}
