package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// RemoveFeedInput は remove-feed ツールの入力。
type RemoveFeedInput struct {
	ID int64 `json:"id" jsonschema:"削除するフィードのID"`
	// Confirm はユーザーへの明示的な同意確認なしに true を渡してはならない
	// （破壊的操作のため、ツールの description で同意確認を必須化している）。
	Confirm bool `json:"confirm" jsonschema:"true の場合のみ削除を実行する。ユーザーへの明示的な同意確認なしにtrueを渡してはならない"`
}

// RemoveFeedOutput は remove-feed ツールの出力。
type RemoveFeedOutput struct {
	ID int64 `json:"id"`
}

// RemoveFeedTool はフィードと関連記事を完全削除する remove-feed ツールのハンドラを構築する。
// confirm != true の場合は usecase を呼び出さずにエラーを返す。
func RemoveFeedTool(uc *usecase.RemoveFeedUsecase) mcpsdk.ToolHandlerFor[RemoveFeedInput, RemoveFeedOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, input RemoveFeedInput) (*mcpsdk.CallToolResult, RemoveFeedOutput, error) {
		if err := requireConfirm(input.Confirm); err != nil {
			return nil, RemoveFeedOutput{}, err
		}
		if err := uc.Execute(ctx, input.ID); err != nil {
			return nil, RemoveFeedOutput{}, err
		}
		return nil, RemoveFeedOutput{ID: input.ID}, nil
	}
}
