package article

import (
	"context"
	"database/sql"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/samber/do/v2"
)

type repository struct {
	db *sql.DB
}

func NewRepository(i do.Injector) (articlerepo.Repository, error) {
	return &repository{db: do.MustInvoke[*sql.DB](i)}, nil
}

func (r *repository) Save(_ context.Context, _ domain.Article) error               { return nil }
func (r *repository) FindAll(_ context.Context) ([]domain.Article, error)          { return nil, nil }
func (r *repository) FindUnread(_ context.Context) ([]domain.Article, error)       { return nil, nil }
func (r *repository) FindBookmarked(_ context.Context) ([]domain.Article, error)   { return nil, nil }
func (r *repository) FindByID(_ context.Context, _ int64) (*domain.Article, error) { return nil, nil }
func (r *repository) Update(_ context.Context, _ domain.Article) error             { return nil }
func (r *repository) DeleteNonBookmarked(_ context.Context) (int64, error)         { return 0, nil }
func (r *repository) CountNonBookmarked(_ context.Context) (int64, error)          { return 0, nil }
