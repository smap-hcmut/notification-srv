package usecase

import (
	"smap-api/internal/user"
	"smap-api/internal/user/repository"
	pkgLog "smap-api/pkg/log"
)

type usecase struct {
	l    pkgLog.Logger
	repo repository.Repository
}

func New(l pkgLog.Logger, repo repository.Repository) user.UseCase {
	return &usecase{
		l:    l,
		repo: repo,
	}
}
