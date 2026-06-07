package rss

import (
	"context"

	adapterrss "github.com/qli8racn/rss-feeder/internal/adapter/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/samber/do/v2"
)

type reader struct{}

func NewReader(_ do.Injector) (adapterrss.RSSReader, error) {
	return &reader{}, nil
}

func (r *reader) Fetch(_ context.Context, _ string) (string, []domain.Article, error) {
	return "", nil, nil
}
