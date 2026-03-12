package usecase

import (
	"notification-srv/internal/alert"

	"github.com/smap-hcmut/shared-libs/go/discord"
	"github.com/smap-hcmut/shared-libs/go/log"
)

type implUseCase struct {
	logger  log.Logger
	discord discord.IDiscord
}

func New(logger log.Logger, discord discord.IDiscord) alert.UseCase {
	return &implUseCase{
		logger:  logger,
		discord: discord,
	}
}
