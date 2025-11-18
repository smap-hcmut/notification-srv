package postgres

import (
	"context"
	"database/sql"

	"smap-api/internal/model"
	"smap-api/internal/sqlboiler"
	"smap-api/internal/user/repository"
	"smap-api/pkg/paginator"
	postgresPkg "smap-api/pkg/postgre"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
)

func (r *implRepository) Detail(ctx context.Context, sc model.Scope, id string) (model.User, error) {
	mods, err := r.buildDetailQuery(ctx, id)
	if err != nil {
		r.l.Errorf(ctx, "internal.user.repository.postgres.Detail.buildDetailQuery: %v", err)
		return model.User{}, err
	}

	usr, err := sqlboiler.Users(mods...).One(ctx, r.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.User{}, repository.ErrNotFound
		}
		r.l.Errorf(ctx, "internal.user.repository.postgres.Detail.One: %v", err)
		return model.User{}, err
	}

	return *model.NewUserFromDB(usr), nil
}

func (r *implRepository) List(ctx context.Context, sc model.Scope, opts repository.ListOptions) ([]model.User, error) {
	mods, err := r.buildListQuery(ctx, opts)
	if err != nil {
		r.l.Errorf(ctx, "internal.user.repository.postgres.List.buildListQuery: %v", err)
		return nil, err
	}

	usrs, err := sqlboiler.Users(mods...).All(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "internal.user.repository.postgres.List.All: %v", err)
		return nil, err
	}

	res := make([]model.User, len(usrs))
	for i, u := range usrs {
		res[i] = *model.NewUserFromDB(u)
	}

	return res, nil
}

func (r *implRepository) Create(ctx context.Context, sc model.Scope, opts repository.CreateOptions) (model.User, error) {
	if opts.User.ID != "" {
		if err := postgresPkg.IsUUID(opts.User.ID); err != nil {
			r.l.Errorf(ctx, "internal.user.repository.postgres.Create.IsUUID: %v", err)
			return model.User{}, err
		}
	}

	dbUsr := opts.User.ToDBUser()
	if err := dbUsr.Insert(ctx, r.db, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "internal.user.repository.postgres.Create.Insert: %v", err)
		return model.User{}, err
	}

	usr, err := sqlboiler.Users(
		sqlboiler.UserWhere.ID.EQ(dbUsr.ID),
	).One(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "internal.user.repository.postgres.Create.Reload: %v", err)
		return model.User{}, err
	}

	return *model.NewUserFromDB(usr), nil
}

func (r *implRepository) Update(ctx context.Context, sc model.Scope, opts repository.UpdateOptions) (model.User, error) {
	if err := postgresPkg.IsUUID(opts.User.ID); err != nil {
		r.l.Errorf(ctx, "internal.user.repository.postgres.Update.IsUUID: %v", err)
		return model.User{}, err
	}

	_, err := sqlboiler.Users(
		sqlboiler.UserWhere.ID.EQ(opts.User.ID),
		sqlboiler.UserWhere.DeletedAt.IsNull(),
	).One(ctx, r.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.User{}, repository.ErrNotFound
		}
		r.l.Errorf(ctx, "internal.user.repository.postgres.Update.Find: %v", err)
		return model.User{}, err
	}

	dbUsr := opts.User.ToDBUser()
	rows, err := dbUsr.Update(ctx, r.db, boil.Infer())
	if err != nil {
		r.l.Errorf(ctx, "internal.user.repository.postgres.Update.Update: %v", err)
		return model.User{}, err
	}

	if rows == 0 {
		return model.User{}, repository.ErrNotFound
	}

	usr, err := sqlboiler.Users(
		sqlboiler.UserWhere.ID.EQ(opts.User.ID),
	).One(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "internal.user.repository.postgres.Update.Reload: %v", err)
		return model.User{}, err
	}

	return *model.NewUserFromDB(usr), nil
}

func (r *implRepository) GetOne(ctx context.Context, sc model.Scope, opts repository.GetOneOptions) (model.User, error) {
	mods, err := r.buildGetOneQuery(ctx, opts)
	if err != nil {
		r.l.Errorf(ctx, "internal.user.repository.postgres.GetOne.buildGetOneQuery: %v", err)
		return model.User{}, err
	}

	usr, err := sqlboiler.Users(mods...).One(ctx, r.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.User{}, repository.ErrNotFound
		}
		r.l.Errorf(ctx, "internal.user.repository.postgres.GetOne.One: %v", err)
		return model.User{}, err
	}

	return *model.NewUserFromDB(usr), nil
}

func (r *implRepository) Get(ctx context.Context, sc model.Scope, opts repository.GetOptions) ([]model.User, paginator.Paginator, error) {
	mods, err := r.buildGetQuery(ctx, opts, opts.PaginateQuery)
	if err != nil {
		r.l.Errorf(ctx, "internal.user.repository.postgres.Get.buildGetQuery: %v", err)
		return nil, paginator.Paginator{}, err
	}

	cntMods := []qm.QueryMod{
		sqlboiler.UserWhere.DeletedAt.IsNull(),
	}
	if len(opts.Filter.IDs) > 0 {
		cntMods = append(cntMods, sqlboiler.UserWhere.ID.IN(opts.Filter.IDs))
	}

	total, err := sqlboiler.Users(cntMods...).Count(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "internal.user.repository.postgres.Get.Count: %v", err)
		return nil, paginator.Paginator{}, err
	}

	usrs, err := sqlboiler.Users(mods...).All(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "internal.user.repository.postgres.Get.All: %v", err)
		return nil, paginator.Paginator{}, err
	}

	res := make([]model.User, len(usrs))
	for i, u := range usrs {
		res[i] = *model.NewUserFromDB(u)
	}

	opts.PaginateQuery.Adjust()
	pag := paginator.Paginator{
		Total:       total,
		Count:       int64(len(res)),
		PerPage:     opts.PaginateQuery.Limit,
		CurrentPage: opts.PaginateQuery.Page,
	}

	return res, pag, nil
}

func (r *implRepository) Delete(ctx context.Context, sc model.Scope, id string) error {
	if err := postgresPkg.IsUUID(id); err != nil {
		r.l.Errorf(ctx, "internal.user.repository.postgres.Delete.IsUUID: %v", err)
		return err
	}

	usr, err := sqlboiler.Users(
		sqlboiler.UserWhere.ID.EQ(id),
		sqlboiler.UserWhere.DeletedAt.IsNull(),
	).One(ctx, r.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return repository.ErrNotFound
		}
		r.l.Errorf(ctx, "internal.user.repository.postgres.Delete.Find: %v", err)
		return err
	}

	usr.DeletedAt = null.TimeFrom(r.clock())
	rows, err := usr.Update(ctx, r.db, boil.Whitelist(sqlboiler.UserColumns.DeletedAt))
	if err != nil {
		r.l.Errorf(ctx, "internal.user.repository.postgres.Delete.Update: %v", err)
		return err
	}

	if rows == 0 {
		return repository.ErrNotFound
	}

	return nil
}
