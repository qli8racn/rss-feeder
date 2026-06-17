package usecase

import (
	"context"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
)

type EnrichUsecase struct {
	agent adapteranthropic.EnrichAgent
}

func NewEnrichUsecase(agent adapteranthropic.EnrichAgent) *EnrichUsecase {
	return &EnrichUsecase{agent: agent}
}

func (uc *EnrichUsecase) Execute(ctx context.Context, opts adapteranthropic.EnrichOptions) (int, error) {
	return uc.agent.Run(ctx, opts)
}
