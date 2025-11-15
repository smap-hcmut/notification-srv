package model

import (
	"time"

	"smap-api/internal/sqlboiler"

	"github.com/aarondl/null/v8"
)

// SubscriptionStatus represents the status of a subscription.
// This is a domain enum that mirrors the database enum.
type SubscriptionStatus string

const (
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusTrialing  SubscriptionStatus = "trialing"
	SubscriptionStatusPastDue   SubscriptionStatus = "past_due"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
	SubscriptionStatusExpired   SubscriptionStatus = "expired"
)

// IsValid checks if the subscription status is valid.
func (s SubscriptionStatus) IsValid() bool {
	switch s {
	case SubscriptionStatusActive,
		SubscriptionStatusTrialing,
		SubscriptionStatusPastDue,
		SubscriptionStatusCancelled,
		SubscriptionStatusExpired:
		return true
	default:
		return false
	}
}

// String returns the string representation of the status.
func (s SubscriptionStatus) String() string {
	return string(s)
}

// Subscription represents a subscription entity in the domain layer.
// This is a safe type model that doesn't depend on database-specific types.
type Subscription struct {
	ID          string             `json:"id"`
	UserID      string             `json:"user_id"`
	PlanID      string             `json:"plan_id"`
	Status      SubscriptionStatus `json:"status"`
	TrialEndsAt *time.Time         `json:"trial_ends_at,omitempty"`
	StartsAt    time.Time          `json:"starts_at"`
	EndsAt      *time.Time         `json:"ends_at,omitempty"`
	CancelledAt *time.Time         `json:"cancelled_at,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	DeletedAt   *time.Time         `json:"deleted_at,omitempty"`
}

// NewSubscriptionFromDB converts a SQLBoiler Subscription model to a domain Subscription model.
// It safely handles null values from the database.
func NewSubscriptionFromDB(dbSub *sqlboiler.Subscription) *Subscription {
	sub := &Subscription{
		ID:        dbSub.ID,
		UserID:    dbSub.UserID,
		PlanID:    dbSub.PlanID,
		Status:    SubscriptionStatus(dbSub.Status),
		StartsAt:  dbSub.StartsAt,
		CreatedAt: dbSub.CreatedAt,
		UpdatedAt: dbSub.UpdatedAt,
	}

	// Handle nullable fields safely
	if dbSub.TrialEndsAt.Valid {
		sub.TrialEndsAt = &dbSub.TrialEndsAt.Time
	}
	if dbSub.EndsAt.Valid {
		sub.EndsAt = &dbSub.EndsAt.Time
	}
	if dbSub.CancelledAt.Valid {
		sub.CancelledAt = &dbSub.CancelledAt.Time
	}
	if dbSub.DeletedAt.Valid {
		sub.DeletedAt = &dbSub.DeletedAt.Time
	}

	return sub
}

// ToDBSubscription converts a domain Subscription model to a SQLBoiler Subscription model for database operations.
// Note: This is typically used in repository layer, not in domain logic.
func (s *Subscription) ToDBSubscription() *sqlboiler.Subscription {
	dbSub := &sqlboiler.Subscription{
		ID:        s.ID,
		UserID:    s.UserID,
		PlanID:    s.PlanID,
		Status:    sqlboiler.SubscriptionStatus(s.Status),
		StartsAt:  s.StartsAt,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}

	// Convert nullable fields
	if s.TrialEndsAt != nil {
		dbSub.TrialEndsAt = null.TimeFrom(*s.TrialEndsAt)
	}
	if s.EndsAt != nil {
		dbSub.EndsAt = null.TimeFrom(*s.EndsAt)
	}
	if s.CancelledAt != nil {
		dbSub.CancelledAt = null.TimeFrom(*s.CancelledAt)
	}
	if s.DeletedAt != nil {
		dbSub.DeletedAt = null.TimeFrom(*s.DeletedAt)
	}

	return dbSub
}
