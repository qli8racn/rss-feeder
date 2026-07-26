package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// ListFeedsInput は list-feeds ツールの入力（引数なし）。
type ListFeedsInput struct{}

// ListFeedsOutput は list-feeds ツールの出力。
type ListFeedsOutput struct {
	Feeds []FeedDTO `json:"feeds"`
}

// ListFeedsTool は登録済みフィード一覧を返す list-feeds ツールのハンドラを構築する。
func ListFeedsTool(uc *usecase.ListFeedsUsecase) mcpsdk.ToolHandlerFor[ListFeedsInput, ListFeedsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ ListFeedsInput) (*mcpsdk.CallToolResult, ListFeedsOutput, error) {
		feeds, err := uc.Execute(ctx)
		if err != nil {
			return nil, ListFeedsOutput{}, err
		}
		return nil, ListFeedsOutput{Feeds: toFeedDTOs(feeds)}, nil
	}
}
