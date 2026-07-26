package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// BookmarkInput は bookmark ツールの入力。
type BookmarkInput struct {
	ID int64 `json:"id" jsonschema:"ブックマークの登録/解除をトグルする記事のID"`
}

// BookmarkOutput は bookmark ツールの出力。
type BookmarkOutput struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Bookmarked bool   `json:"bookmarked"`
}

// BookmarkTool は記事のブックマーク状態をトグルする bookmark ツールのハンドラを構築する。
func BookmarkTool(uc *usecase.BookmarkUsecase) mcpsdk.ToolHandlerFor[BookmarkInput, BookmarkOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, input BookmarkInput) (*mcpsdk.CallToolResult, BookmarkOutput, error) {
		article, err := uc.Execute(ctx, input.ID)
		if err != nil {
			return nil, BookmarkOutput{}, err
		}
		return nil, BookmarkOutput{ID: article.ID, Title: article.Title, Bookmarked: article.Bookmarked}, nil
	}
}
