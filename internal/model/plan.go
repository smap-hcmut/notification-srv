package model

import (
	"time"

	"smap-api/internal/sqlboiler"

	"github.com/aarondl/null/v8"
)

// Plan represents a subscription plan entity in the domain layer.
// This is a safe type model that doesn't depend on database-specific types.
type Plan struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Code        string     `json:"code"`
	Description *string    `json:"description,omitempty"`
	MaxUsage    int        `json:"max_usage"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// NewPlanFromDB converts a SQLBoiler Plan model to a domain Plan model.
// It safely handles null values from the database.
func NewPlanFromDB(dbPlan *sqlboiler.Plan) *Plan {
	plan := &Plan{
		ID:        dbPlan.ID,
		Name:      dbPlan.Name,
		Code:      dbPlan.Code,
		MaxUsage:  dbPlan.MaxUsage,
		CreatedAt: dbPlan.CreatedAt.Time,
		UpdatedAt: dbPlan.UpdatedAt.Time,
	}

	// Handle nullable fields safely
	if dbPlan.Description.Valid {
		plan.Description = &dbPlan.Description.String
	}
	if dbPlan.DeletedAt.Valid {
		plan.DeletedAt = &dbPlan.DeletedAt.Time
	}

	return plan
}

// ToDBPlan converts a domain Plan model to a SQLBoiler Plan model for database operations.
// Note: This is typically used in repository layer, not in domain logic.
func (p *Plan) ToDBPlan() *sqlboiler.Plan {
	dbPlan := &sqlboiler.Plan{
		ID:        p.ID,
		Name:      p.Name,
		Code:      p.Code,
		MaxUsage:  p.MaxUsage,
		CreatedAt: null.TimeFrom(p.CreatedAt),
		UpdatedAt: null.TimeFrom(p.UpdatedAt),
	}

	// Convert nullable fields
	if p.Description != nil {
		dbPlan.Description = null.StringFrom(*p.Description)
	}
	if p.DeletedAt != nil {
		dbPlan.DeletedAt = null.TimeFrom(*p.DeletedAt)
	}

	return dbPlan
}
