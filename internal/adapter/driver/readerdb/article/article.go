package article

import (
	"context"
	"errors"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

var ErrDuplicate = errors.New("duplicate article")

type Repository interface {
	Save(ctx context.Context, article domain.Article) error
	FindAll(ctx context.Context) ([]domain.Article, error)
	FindUnread(ctx context.Context) ([]domain.Article, error)
	FindBookmarked(ctx context.Context) ([]domain.Article, error)
	FindByID(ctx context.Context, id int64) (*domain.Article, error)
	Update(ctx context.Context, article domain.Article) error
	MarkAsRead(ctx context.Context, ids []int64) error
	DeleteNonBookmarked(ctx context.Context) (int64, error)
	CountNonBookmarked(ctx context.Context) (int64, error)
	CountBookmarked(ctx context.Context) (int64, error)
	FetchLatest(ctx context.Context, limit int, feedURL string) ([]domain.Article, error)
	Search(ctx context.Context, keyword string, bookmarkedOnly bool) ([]domain.Article, error)
}
