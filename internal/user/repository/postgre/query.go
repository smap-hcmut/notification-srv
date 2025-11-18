package postgres

import (
	"context"

	"smap-api/internal/sqlboiler"
	"smap-api/internal/user/repository"
	"smap-api/pkg/paginator"
	postgresPkg "smap-api/pkg/postgre"

	"github.com/aarondl/sqlboiler/v4/queries/qm"
)

func (r *implRepository) buildListQuery(ctx context.Context, opts repository.ListOptions) ([]qm.QueryMod, error) {
	mods := []qm.QueryMod{
		sqlboiler.UserWhere.DeletedAt.IsNull(),
	}

	if len(opts.Filter.IDs) > 0 {
		if err := postgresPkg.ValidateUUIDs(opts.Filter.IDs); err != nil {
			r.l.Errorf(ctx, "internal.user.repository.postgres.buildListQuery.ValidateUUIDs: %v", err)
			return nil, err
		}
		mods = append(mods, sqlboiler.UserWhere.ID.IN(opts.Filter.IDs))
	}

	return mods, nil
}

func (r *implRepository) buildDetailQuery(ctx context.Context, id string) ([]qm.QueryMod, error) {
	if err := postgresPkg.IsUUID(id); err != nil {
		r.l.Errorf(ctx, "internal.user.repository.postgres.buildDetailQuery.IsUUID: %v", err)
		return nil, err
	}

	return []qm.QueryMod{
		sqlboiler.UserWhere.ID.EQ(id),
		sqlboiler.UserWhere.DeletedAt.IsNull(),
	}, nil
}

func (r *implRepository) buildGetOneQuery(ctx context.Context, opts repository.GetOneOptions) ([]qm.QueryMod, error) {
	mods := []qm.QueryMod{
		sqlboiler.UserWhere.DeletedAt.IsNull(),
	}

	if opts.ID != "" {
		if err := postgresPkg.IsUUID(opts.ID); err != nil {
			r.l.Errorf(ctx, "internal.user.repository.postgres.buildGetOneQuery.IsUUID: %v", err)
			return nil, err
		}
		mods = append(mods, sqlboiler.UserWhere.ID.EQ(opts.ID))
	} else if opts.Username != "" {
		mods = append(mods, sqlboiler.UserWhere.Username.EQ(opts.Username))
	}

	return mods, nil
}

func (r *implRepository) buildGetQuery(ctx context.Context, opts repository.GetOptions, pq paginator.PaginateQuery) ([]qm.QueryMod, error) {
	mods := []qm.QueryMod{
		sqlboiler.UserWhere.DeletedAt.IsNull(),
	}

	if len(opts.Filter.IDs) > 0 {
		if err := postgresPkg.ValidateUUIDs(opts.Filter.IDs); err != nil {
			r.l.Errorf(ctx, "internal.user.repository.postgres.buildGetQuery.ValidateUUIDs: %v", err)
			return nil, err
		}
		mods = append(mods, sqlboiler.UserWhere.ID.IN(opts.Filter.IDs))
	}

	pq.Adjust()
	mods = append(mods,
		qm.Limit(int(pq.Limit)),
		qm.Offset(int(pq.Offset())),
		qm.OrderBy(sqlboiler.UserColumns.CreatedAt+" DESC"),
	)

	return mods, nil
}
