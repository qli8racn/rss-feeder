package feed

import (
	"context"
	"errors"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

var (
	ErrAlreadyExists = errors.New("feed already registered")
	ErrNotFound      = errors.New("feed not found")
)

type Repository interface {
	Save(ctx context.Context, feed domain.Feed) (int64, error)
	FindByURL(ctx context.Context, url string) (*domain.Feed, error)
	Register(ctx context.Context, url string) error
	ListAll(ctx context.Context) ([]domain.Feed, error)
	Remove(ctx context.Context, id int64) error
}
