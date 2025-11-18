package model

import (
	"time"

	"smap-api/internal/sqlboiler"

	"github.com/aarondl/null/v8"
)

// User represents a user entity in the domain layer.
// This is a safe type model that doesn't depend on database-specific types.
type User struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	FullName     *string    `json:"full_name,omitempty"`
	PasswordHash *string    `json:"password_hash,omitempty"`
	AvatarURL    *string    `json:"avatar_url,omitempty"`
	IsActive     *bool      `json:"is_active,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

// NewUserFromDB converts a SQLBoiler User model to a domain User model.
// It safely handles null values from the database.
func NewUserFromDB(dbUser *sqlboiler.User) *User {
	user := &User{
		ID:        dbUser.ID,
		Username:  dbUser.Username,
		CreatedAt: dbUser.CreatedAt.Time,
		UpdatedAt: dbUser.UpdatedAt.Time,
	}

	// Handle nullable fields safely
	if dbUser.FullName.Valid {
		user.FullName = &dbUser.FullName.String
	}
	if dbUser.PasswordHash.Valid {
		user.PasswordHash = &dbUser.PasswordHash.String
	}
	if dbUser.AvatarURL.Valid {
		user.AvatarURL = &dbUser.AvatarURL.String
	}
	if dbUser.IsActive.Valid {
		user.IsActive = &dbUser.IsActive.Bool
	}
	if dbUser.DeletedAt.Valid {
		user.DeletedAt = &dbUser.DeletedAt.Time
	}

	return user
}

// ToDBUser converts a domain User model to a SQLBoiler User model for database operations.
// Note: This is typically used in repository layer, not in domain logic.
func (u *User) ToDBUser() *sqlboiler.User {
	dbUser := &sqlboiler.User{
		ID:       u.ID,
		Username: u.Username,
	}

	// Convert nullable fields
	if u.FullName != nil {
		dbUser.FullName = null.StringFrom(*u.FullName)
	}
	if u.PasswordHash != nil {
		dbUser.PasswordHash = null.StringFrom(*u.PasswordHash)
	}
	if u.AvatarURL != nil {
		dbUser.AvatarURL = null.StringFrom(*u.AvatarURL)
	}
	if u.IsActive != nil {
		dbUser.IsActive = null.BoolFrom(*u.IsActive)
	}
	if u.DeletedAt != nil {
		dbUser.DeletedAt = null.TimeFrom(*u.DeletedAt)
	}

	return dbUser
}
