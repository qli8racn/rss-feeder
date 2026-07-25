package usecase

import (
	"context"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
)

type DiscoverUsecase struct {
	agent adapteranthropic.DiscoverAgent
}

func NewDiscoverUsecase(agent adapteranthropic.DiscoverAgent) *DiscoverUsecase {
	return &DiscoverUsecase{agent: agent}
}

func (uc *DiscoverUsecase) Execute(ctx context.Context) (string, error) {
	return uc.agent.Run(ctx)
}
