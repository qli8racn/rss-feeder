package mcp

import (
	"context"
	"errors"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// ErrListAllAndBookmarkedExclusive は all・bookmarked が同時に true で渡された場合に返す
// （CLI の list コマンドが MarkFlagsMutuallyExclusive("all", "bookmarked") で両立を禁止しているのと同じ意味論）。
var ErrListAllAndBookmarkedExclusive = errors.New("all と bookmarked は同時に true にできません")

// ListInput は list ツールの入力。
type ListInput struct {
	All        bool   `json:"all,omitempty" jsonschema:"既読・未読を問わず全記事を対象にする。bookmarkedとの同時指定は不可"`
	Bookmarked bool   `json:"bookmarked,omitempty" jsonschema:"ブックマーク済みの記事のみに絞り込む。allとの同時指定は不可"`
	Category   string `json:"category,omitempty" jsonschema:"指定したカテゴリの記事のみに絞り込む"`
	Limit      int    `json:"limit,omitempty" jsonschema:"1ページあたりの取得件数（省略時は50、上限200）"`
	Page       int    `json:"page,omitempty" jsonschema:"取得するページ番号（1始まり、省略時は1）"`
}

// ListOutput は list ツールの出力。
type ListOutput struct {
	Articles []ArticleDTO `json:"articles"`
	// Total は絞り込み条件に一致する記事の総数。Articles との差分から、この呼び出しで
	// 取得しきれなかった記事が残っているかどうかを LLM が判断できるようにする。
	Total int64 `json:"total"`
}

// ListTool は保存済み記事一覧を返す list ツールのハンドラを構築する。
// all・bookmarked のいずれも指定しない場合は未読記事のみを返す（CLI と同じデフォルト挙動）。
// CLI・Web UI と異なり、一覧表示によって記事が既読になる副作用は起こさない（読み取り専用）。
func ListTool(uc *usecase.ListUsecase) mcpsdk.ToolHandlerFor[ListInput, ListOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, input ListInput) (*mcpsdk.CallToolResult, ListOutput, error) {
		if input.All && input.Bookmarked {
			return nil, ListOutput{}, ErrListAllAndBookmarkedExclusive
		}

		mode := usecase.ListModeUnread
		switch {
		case input.All:
			mode = usecase.ListModeAll
		case input.Bookmarked:
			mode = usecase.ListModeBookmarked
		}

		articles, total, err := uc.ExecuteFiltered(ctx, usecase.ListFilterOptions{
			Mode:           mode,
			Category:       input.Category,
			Sort:           "published_at",
			Order:          "desc",
			Page:           resolvePage(input.Page),
			PerPage:        resolvePerPage(input.Limit),
			SkipMarkAsRead: true,
		})
		if err != nil {
			return nil, ListOutput{}, err
		}
		return nil, ListOutput{Articles: toArticleDTOs(articles), Total: total}, nil
	}
}
