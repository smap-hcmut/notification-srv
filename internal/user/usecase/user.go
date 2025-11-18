package usecase

import (
	"context"

	"smap-api/internal/model"
	"smap-api/internal/user"
	"smap-api/internal/user/repository"
	"smap-api/pkg/encrypter"
	postgresPkg "smap-api/pkg/postgre"
)

func (uc *usecase) Detail(ctx context.Context, sc model.Scope, id string) (user.UserOutput, error) {
	usr, err := uc.repo.Detail(ctx, sc, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return user.UserOutput{}, user.ErrUserNotFound
		}
		uc.l.Errorf(ctx, "internal.user.usecase.Detail: %v", err)
		return user.UserOutput{}, err
	}

	return user.UserOutput{User: usr}, nil
}

func (uc *usecase) DetailMe(ctx context.Context, sc model.Scope) (user.UserOutput, error) {
	usr, err := uc.repo.Detail(ctx, sc, sc.UserID)
	if err != nil {
		if err == repository.ErrNotFound {
			return user.UserOutput{}, user.ErrUserNotFound
		}
		uc.l.Errorf(ctx, "internal.user.usecase.DetailMe: %v", err)
		return user.UserOutput{}, err
	}

	return user.UserOutput{User: usr}, nil
}

func (uc *usecase) List(ctx context.Context, sc model.Scope, ip user.ListInput) ([]model.User, error) {
	opts := repository.ListOptions{
		Filter: repository.Filter{
			IDs: ip.Filter.IDs,
		},
	}

	usrs, err := uc.repo.List(ctx, sc, opts)
	if err != nil {
		uc.l.Errorf(ctx, "internal.user.usecase.List: %v", err)
		return nil, err
	}

	return usrs, nil
}

func (uc *usecase) Get(ctx context.Context, sc model.Scope, ip user.GetInput) (user.GetUserOutput, error) {
	opts := repository.GetOptions{
		Filter: repository.Filter{
			IDs: ip.Filter.IDs,
		},
		PaginateQuery: ip.PaginateQuery,
	}

	usrs, pag, err := uc.repo.Get(ctx, sc, opts)
	if err != nil {
		uc.l.Errorf(ctx, "internal.user.usecase.Get: %v", err)
		return user.GetUserOutput{}, err
	}

	return user.GetUserOutput{
		Users:     usrs,
		Paginator: pag,
	}, nil
}

func (uc *usecase) UpdateProfile(ctx context.Context, sc model.Scope, ip user.UpdateProfileInput) (user.UserOutput, error) {
	usr, err := uc.repo.Detail(ctx, sc, sc.UserID)
	if err != nil {
		if err == repository.ErrNotFound {
			return user.UserOutput{}, user.ErrUserNotFound
		}
		uc.l.Errorf(ctx, "internal.user.usecase.UpdateProfile.Detail: %v", err)
		return user.UserOutput{}, err
	}

	usr.FullName = &ip.FullName
	if ip.AvatarURL != "" {
		usr.AvatarURL = &ip.AvatarURL
	}

	updated, err := uc.repo.Update(ctx, sc, repository.UpdateOptions{User: usr})
	if err != nil {
		uc.l.Errorf(ctx, "internal.user.usecase.UpdateProfile.Update: %v", err)
		return user.UserOutput{}, err
	}

	return user.UserOutput{User: updated}, nil
}

func (uc *usecase) Create(ctx context.Context, sc model.Scope, ip user.CreateInput) (user.UserOutput, error) {
	_, err := uc.repo.GetOne(ctx, sc, repository.GetOneOptions{Username: ip.Username})
	if err == nil {
		return user.UserOutput{}, user.ErrUserExists
	}
	if err != repository.ErrNotFound {
		uc.l.Errorf(ctx, "internal.user.usecase.Create.GetOne: %v", err)
		return user.UserOutput{}, err
	}

	hash, err := encrypter.HashPassword(ip.Password)
	if err != nil {
		uc.l.Errorf(ctx, "internal.user.usecase.Create.HashPassword: %v", err)
		return user.UserOutput{}, err
	}

	usr := model.User{
		ID:           postgresPkg.NewUUID(),
		Username:     ip.Username,
		PasswordHash: &hash,
		FullName:     &ip.FullName,
		IsActive:     boolPtr(true),
	}

	created, err := uc.repo.Create(ctx, sc, repository.CreateOptions{User: usr})
	if err != nil {
		uc.l.Errorf(ctx, "internal.user.usecase.Create: %v", err)
		return user.UserOutput{}, err
	}

	return user.UserOutput{User: created}, nil
}

func (uc *usecase) GetOne(ctx context.Context, sc model.Scope, ip user.GetOneInput) (model.User, error) {
	usr, err := uc.repo.GetOne(ctx, sc, repository.GetOneOptions{
		Username: ip.Username,
		ID:       ip.ID,
	})
	if err != nil {
		if err == repository.ErrNotFound {
			return model.User{}, user.ErrUserNotFound
		}
		uc.l.Errorf(ctx, "internal.user.usecase.GetOne: %v", err)
		return model.User{}, err
	}

	return usr, nil
}

func (uc *usecase) Update(ctx context.Context, sc model.Scope, ip user.UpdateInput) (user.UserOutput, error) {
	usr, err := uc.repo.Detail(ctx, sc, ip.ID)
	if err != nil {
		if err == repository.ErrNotFound {
			return user.UserOutput{}, user.ErrUserNotFound
		}
		uc.l.Errorf(ctx, "internal.user.usecase.Update.Detail: %v", err)
		return user.UserOutput{}, err
	}

	if ip.FullName != nil {
		usr.FullName = ip.FullName
	}
	if ip.AvatarURL != nil {
		usr.AvatarURL = ip.AvatarURL
	}
	if ip.IsActive != nil {
		usr.IsActive = ip.IsActive
	}

	updated, err := uc.repo.Update(ctx, sc, repository.UpdateOptions{User: usr})
	if err != nil {
		uc.l.Errorf(ctx, "internal.user.usecase.Update: %v", err)
		return user.UserOutput{}, err
	}

	return user.UserOutput{User: updated}, nil
}

func (uc *usecase) Delete(ctx context.Context, sc model.Scope, id string) error {
	if err := uc.repo.Delete(ctx, sc, id); err != nil {
		if err == repository.ErrNotFound {
			return user.ErrUserNotFound
		}
		uc.l.Errorf(ctx, "internal.user.usecase.Delete: %v", err)
		return err
	}

	return nil
}

func (uc *usecase) Dashboard(ctx context.Context, sc model.Scope, ip user.DashboardInput) (user.UsersDashboardOutput, error) {
	opts := repository.GetOptions{
		PaginateQuery: ip.PaginateQuery,
	}

	_, pag, err := uc.repo.Get(ctx, sc, opts)
	if err != nil {
		uc.l.Errorf(ctx, "internal.user.usecase.Dashboard: %v", err)
		return user.UsersDashboardOutput{}, err
	}

	return user.UsersDashboardOutput{
		Total:  pag.Total,
		Active: 0,
		Growth: 0,
	}, nil
}

func boolPtr(b bool) *bool {
	return &b
}
