package usecase

import (
	"context"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
)

type PreferenceUsecase struct {
	agent adapteranthropic.PreferenceAgent
}

func NewPreferenceUsecase(agent adapteranthropic.PreferenceAgent) *PreferenceUsecase {
	return &PreferenceUsecase{agent: agent}
}

func (uc *PreferenceUsecase) Execute(ctx context.Context) (string, error) {
	return uc.agent.Run(ctx)
}
