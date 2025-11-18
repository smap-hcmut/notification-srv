package repository

import (
	"smap-api/internal/model"
	"smap-api/pkg/paginator"
)

// Filter contains filtering options for user queries.
type Filter struct {
	IDs      []string
	Username string
	IsActive *bool
	Search   string // For searching in username or full_name
}

// CreateOptions contains options for creating a user.
type CreateOptions struct {
	User model.User
}

// UpdateOptions contains options for updating a user.
// Only non-nil fields will be updated.
type UpdateOptions struct {
	User model.User
}

// GetOneOptions contains options for getting a single user.
type GetOneOptions struct {
	Username string
	ID       string
}

// ListOptions contains options for listing users.
type ListOptions struct {
	Filter Filter
}

// GetOptions contains options for paginated user listing.
type GetOptions struct {
	Filter        Filter
	PaginateQuery paginator.PaginateQuery
}
