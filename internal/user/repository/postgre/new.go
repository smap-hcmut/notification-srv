package postgres

import (
	"database/sql"
	"time"

	"smap-api/internal/user/repository"
	pkgLog "smap-api/pkg/log"
)

type implRepository struct {
	l     pkgLog.Logger
	db    *sql.DB
	clock func() time.Time
}

var _ repository.Repository = &implRepository{}

func New(l pkgLog.Logger, db *sql.DB) *implRepository {
	return &implRepository{
		l:     l,
		db:    db,
		clock: time.Now,
	}
}
