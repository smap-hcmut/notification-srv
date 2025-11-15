package repository

import (
	"context"

	"smap-api/internal/model"
	"smap-api/pkg/paginator"
)

//go:generate mockery --name Repository
type Repository interface {
	Get(ctx context.Context, sc model.Scope, opts GetOptions) ([]model.User, paginator.Paginator, error)
	Detail(ctx context.Context, sc model.Scope, id string) (model.User, error)
	List(ctx context.Context, sc model.Scope, opts ListOptions) ([]model.User, error)
	Create(ctx context.Context, sc model.Scope, opts CreateOptions) (model.User, error)
	Update(ctx context.Context, sc model.Scope, opts UpdateOptions) (model.User, error)
	GetOne(ctx context.Context, sc model.Scope, opts GetOneOptions) (model.User, error)
	Delete(ctx context.Context, sc model.Scope, id string) error
}
