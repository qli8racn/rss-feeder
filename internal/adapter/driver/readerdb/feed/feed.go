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

// Repository の各メソッドは userID でスコープする（フィードはユーザーごとに
// 別行として管理されるため）。
type Repository interface {
	Save(ctx context.Context, feed domain.Feed, userID int64) (int64, error)
	FindByURL(ctx context.Context, url string, userID int64) (*domain.Feed, error)
	Register(ctx context.Context, url string, userID int64) error
	ListAll(ctx context.Context, userID int64) ([]domain.Feed, error)
	Remove(ctx context.Context, id int64, userID int64) error
}
