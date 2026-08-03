package usecase

import (
	"context"
	"errors"
	"fmt"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
)

// ErrMarkReadIDsRequired は ids が1件も指定されなかった場合に返す。
var ErrMarkReadIDsRequired = errors.New("ids には1件以上指定してください")

// MarkReadUsecase は指定した記事IDを既読にする。
// rss_list（MCP）は閲覧そのものを既読化のトリガーにしない読み取り専用ツールにしたため、
// MCP経由で明示的に既読管理を行うための手段として設ける。
type MarkReadUsecase struct {
	articleRepo articlerepo.Repository
	userID      int64
}

func NewMarkReadUsecase(articleRepo articlerepo.Repository, userID int64) *MarkReadUsecase {
	return &MarkReadUsecase{articleRepo: articleRepo, userID: userID}
}

func (uc *MarkReadUsecase) Execute(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return ErrMarkReadIDsRequired
	}
	if err := uc.articleRepo.MarkAsRead(ctx, ids, uc.userID); err != nil {
		return fmt.Errorf("既読化に失敗しました: %w", err)
	}
	return nil
}
