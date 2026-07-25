package usecase

import (
	"context"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
)

type CurateUsecase struct {
	agent adapteranthropic.CurateAgent
}

func NewCurateUsecase(agent adapteranthropic.CurateAgent) *CurateUsecase {
	return &CurateUsecase{agent: agent}
}

func (uc *CurateUsecase) Execute(ctx context.Context, opts adapteranthropic.CurateOptions) (string, error) {
	return uc.agent.Run(ctx, opts)
}
