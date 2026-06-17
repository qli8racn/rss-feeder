package usecase

import (
	"context"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
)

type SummarizeUsecase struct {
	agent adapteranthropic.SummarizeAgent
}

func NewSummarizeUsecase(agent adapteranthropic.SummarizeAgent) *SummarizeUsecase {
	return &SummarizeUsecase{agent: agent}
}

func (uc *SummarizeUsecase) Execute(ctx context.Context, opts adapteranthropic.SummarizeOptions) (string, error) {
	return uc.agent.Run(ctx, opts)
}
