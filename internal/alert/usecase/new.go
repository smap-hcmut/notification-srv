package usecase

import (
	"notification-srv/internal/alert"
	"notification-srv/pkg/discord"
	"notification-srv/pkg/log"
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
