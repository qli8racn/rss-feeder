package usecase

import (
	"context"
	"fmt"

	userrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/user"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

// ResolveUserUsecase は識別子文字列からユーザーを解決する（find-or-create）。
// 他のUsecaseと異なり、解決済みuserIDを構造体フィールドとして保持しない
// （userIDを解決すること自体がこのUsecaseの役目のため）。
// 全エントリポイントのmain.goで、DI配線・migration実行の直後・他Usecase構築より前に1回だけ呼び出す。
type ResolveUserUsecase struct {
	userRepo userrepo.Repository
}

func NewResolveUserUsecase(userRepo userrepo.Repository) *ResolveUserUsecase {
	return &ResolveUserUsecase{userRepo: userRepo}
}

// Execute は name に一致するユーザーを返す。存在しない場合は作成して返す。
// Create がUNIQUE制約違反で失敗した場合（他プロセスが同時に同名ユーザーを作成した場合の
// レース条件）は、再度 FindByName を試みてそちらの結果を採用する。
func (uc *ResolveUserUsecase) Execute(ctx context.Context, name string) (*domain.User, error) {
	u, err := uc.userRepo.FindByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("ユーザー %q の検索に失敗しました: %w", name, err)
	}
	if u != nil {
		return u, nil
	}

	u, err = uc.userRepo.Create(ctx, name)
	if err != nil {
		// UNIQUE制約違反によるレース条件を吸収する: 他プロセスが同時に同名ユーザーを
		// 作成していた場合、そのユーザーを検索して返す。
		if existing, findErr := uc.userRepo.FindByName(ctx, name); findErr == nil && existing != nil {
			return existing, nil
		}
		return nil, fmt.Errorf("ユーザー %q の作成に失敗しました: %w", name, err)
	}
	return u, nil
}
