package user

import (
	"context"

	"smap-api/internal/model"
)

//go:generate mockery --name UseCase
type UseCase interface {
	Detail(ctx context.Context, sc model.Scope, id string) (UserOutput, error)
	DetailMe(ctx context.Context, sc model.Scope) (UserOutput, error)
	List(ctx context.Context, sc model.Scope, ip ListInput) ([]model.User, error)
	Get(ctx context.Context, sc model.Scope, ip GetInput) (GetUserOutput, error)
	UpdateProfile(ctx context.Context, sc model.Scope, ip UpdateProfileInput) (UserOutput, error)
	Create(ctx context.Context, sc model.Scope, ip CreateInput) (UserOutput, error)
	GetOne(ctx context.Context, sc model.Scope, ip GetOneInput) (model.User, error)
	Update(ctx context.Context, sc model.Scope, ip UpdateInput) (UserOutput, error)
	Delete(ctx context.Context, sc model.Scope, id string) error
	Dashboard(ctx context.Context, sc model.Scope, ip DashboardInput) (UsersDashboardOutput, error)
}
