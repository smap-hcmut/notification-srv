package user

import (
	"smap-api/internal/model"
	"smap-api/pkg/paginator"
)

type CreateInput struct {
	Username string
	Password string
	FullName string
}

type UpdateProfileInput struct {
	FullName  string
	AvatarURL string
}

type UpdateInput struct {
	ID        string
	FullName  *string
	AvatarURL *string
	IsActive  *bool
}

type UserOutput struct {
	User model.User
}

type GetUserOutput struct {
	Users     []model.User
	Paginator paginator.Paginator
}

type GetOneInput struct {
	Username string
	ID       string
}

type ListInput struct {
	Filter Filter
}

type GetInput struct {
	Filter        Filter
	PaginateQuery paginator.PaginateQuery
}

type Filter struct {
	IDs []string
}

type DashboardInput struct {
	PaginateQuery paginator.PaginateQuery
}

type UsersDashboardOutput struct {
	Total  int64
	Active int64
	Growth float64
}
