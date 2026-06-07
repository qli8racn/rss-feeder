package feed

import (
	"context"
	"database/sql"

	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/samber/do/v2"
)

type repository struct {
	db *sql.DB
}

func NewRepository(i do.Injector) (feedrepo.Repository, error) {
	return &repository{db: do.MustInvoke[*sql.DB](i)}, nil
}

func (r *repository) Save(_ context.Context, _ domain.Feed) (int64, error)        { return 1, nil }
func (r *repository) FindByURL(_ context.Context, _ string) (*domain.Feed, error) { return nil, nil }
