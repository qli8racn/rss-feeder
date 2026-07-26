package mcp

import (
	"context"
	"errors"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// ErrSearchKeywordRequired は keyword が空文字で渡された場合に返す
// （CLI の search コマンドは cobra.ExactArgs(1) でキーワード省略を許さないのと同じ意味論。
// 空文字を許容すると事実上「全件返却」になり呼び出し側の意図と乖離しやすいため拒否する）。
var ErrSearchKeywordRequired = errors.New("keyword は空文字にできません")

// SearchInput は search ツールの入力。
type SearchInput struct {
	Keyword    string `json:"keyword" jsonschema:"検索キーワード（空文字は不可）"`
	Bookmarked bool   `json:"bookmarked,omitempty" jsonschema:"ブックマーク済みの記事のみを検索対象にする"`
	Category   string `json:"category,omitempty" jsonschema:"指定したカテゴリの記事のみを検索対象にする"`
	Limit      int    `json:"limit,omitempty" jsonschema:"1ページあたりの取得件数（省略時は50、上限200）"`
	Page       int    `json:"page,omitempty" jsonschema:"取得するページ番号（1始まり、省略時は1）"`
}

// SearchOutput は search ツールの出力。
type SearchOutput struct {
	Articles []ArticleDTO `json:"articles"`
	// Total は検索条件に一致する記事の総数（詳細は ListOutput.Total を参照）。
	Total int64 `json:"total"`
}

// SearchTool はキーワードで記事を検索する search ツールのハンドラを構築する。
func SearchTool(uc *usecase.SearchUsecase) mcpsdk.ToolHandlerFor[SearchInput, SearchOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, input SearchInput) (*mcpsdk.CallToolResult, SearchOutput, error) {
		if input.Keyword == "" {
			return nil, SearchOutput{}, ErrSearchKeywordRequired
		}

		articles, total, err := uc.ExecuteFiltered(ctx, usecase.SearchFilterOptions{
			Keyword:        input.Keyword,
			BookmarkedOnly: input.Bookmarked,
			Category:       input.Category,
			Sort:           "published_at",
			Order:          "desc",
			Page:           resolvePage(input.Page),
			PerPage:        resolvePerPage(input.Limit),
		})
		if err != nil {
			return nil, SearchOutput{}, err
		}
		return nil, SearchOutput{Articles: toArticleDTOs(articles), Total: total}, nil
	}
}
