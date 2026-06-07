package feed

import (
	"context"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

type Repository interface {
	Save(ctx context.Context, feed domain.Feed) (int64, error)
	FindByURL(ctx context.Context, url string) (*domain.Feed, error)
}
