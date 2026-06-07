package article

import (
	"context"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

type Repository interface {
	Save(ctx context.Context, article domain.Article) error
	FindAll(ctx context.Context) ([]domain.Article, error)
	FindUnread(ctx context.Context) ([]domain.Article, error)
	FindBookmarked(ctx context.Context) ([]domain.Article, error)
	FindByID(ctx context.Context, id int64) (*domain.Article, error)
	Update(ctx context.Context, article domain.Article) error
	DeleteNonBookmarked(ctx context.Context) (int64, error)
	CountNonBookmarked(ctx context.Context) (int64, error)
}
