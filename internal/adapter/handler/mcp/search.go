package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// SearchInput は search ツールの入力。
type SearchInput struct {
	Keyword    string `json:"keyword" jsonschema:"検索キーワード"`
	Bookmarked bool   `json:"bookmarked,omitempty" jsonschema:"ブックマーク済みの記事のみを検索対象にする"`
	Category   string `json:"category,omitempty" jsonschema:"指定したカテゴリの記事のみを検索対象にする"`
}

// SearchOutput は search ツールの出力。
type SearchOutput struct {
	Articles []ArticleDTO `json:"articles"`
}

// SearchTool はキーワードで記事を検索する search ツールのハンドラを構築する。
func SearchTool(uc *usecase.SearchUsecase) mcpsdk.ToolHandlerFor[SearchInput, SearchOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, input SearchInput) (*mcpsdk.CallToolResult, SearchOutput, error) {
		articles, _, err := uc.ExecuteFiltered(ctx, usecase.SearchFilterOptions{
			Keyword:        input.Keyword,
			BookmarkedOnly: input.Bookmarked,
			Category:       input.Category,
			Sort:           "published_at",
			Order:          "desc",
			PerPage:        listPerPage,
		})
		if err != nil {
			return nil, SearchOutput{}, err
		}
		return nil, SearchOutput{Articles: toArticleDTOs(articles)}, nil
	}
}
