package user

import (
	"context"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

// Repository はユーザーの検索・作成を行う。削除・改名はスコープ外のため提供しない。
type Repository interface {
	// FindByName は name に一致するユーザーを返す。存在しない場合は (nil, nil) を返す。
	FindByName(ctx context.Context, name string) (*domain.User, error)
	Create(ctx context.Context, name string) (*domain.User, error)
}
