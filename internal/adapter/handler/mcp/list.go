package mcp

import (
	"context"
	"errors"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// listPerPage は MCP には CLI 同様ページネーションの概念がないため、実用上「全件」とみなせる
// 十分大きな値を指定する（internal/adapter/handler/cli の cliListPerPage と同じ考え方）。
const listPerPage = 10000

// ErrListAllAndBookmarkedExclusive は all・bookmarked が同時に true で渡された場合に返す
// （CLI の list コマンドが MarkFlagsMutuallyExclusive("all", "bookmarked") で両立を禁止しているのと同じ意味論）。
var ErrListAllAndBookmarkedExclusive = errors.New("all と bookmarked は同時に true にできません")

// ListInput は list ツールの入力。
type ListInput struct {
	All        bool   `json:"all,omitempty" jsonschema:"既読・未読を問わず全記事を対象にする。bookmarkedとの同時指定は不可"`
	Bookmarked bool   `json:"bookmarked,omitempty" jsonschema:"ブックマーク済みの記事のみに絞り込む。allとの同時指定は不可"`
	Category   string `json:"category,omitempty" jsonschema:"指定したカテゴリの記事のみに絞り込む"`
}

// ListOutput は list ツールの出力。
type ListOutput struct {
	Articles []ArticleDTO `json:"articles"`
}

// ListTool は保存済み記事一覧を返す list ツールのハンドラを構築する。
// all・bookmarked のいずれも指定しない場合は未読記事のみを返す（CLI と同じデフォルト挙動）。
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

		articles, _, err := uc.ExecuteFiltered(ctx, usecase.ListFilterOptions{
			Mode:     mode,
			Category: input.Category,
			Sort:     "published_at",
			Order:    "desc",
			PerPage:  listPerPage,
		})
		if err != nil {
			return nil, ListOutput{}, err
		}
		return nil, ListOutput{Articles: toArticleDTOs(articles)}, nil
	}
}
