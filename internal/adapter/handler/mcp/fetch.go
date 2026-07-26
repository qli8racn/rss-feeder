package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// FetchInput は fetch ツールの入力（引数なし）。
type FetchInput struct{}

// FetchOutput は fetch ツールの出力。
type FetchOutput struct {
	Saved   int `json:"saved"`
	Skipped int `json:"skipped"`
	Errors  int `json:"errors"`
}

// FetchTool は登録済み全フィードを取得する fetch ツールのハンドラを構築する。
func FetchTool(uc *usecase.FetchUsecase) mcpsdk.ToolHandlerFor[FetchInput, FetchOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ FetchInput) (*mcpsdk.CallToolResult, FetchOutput, error) {
		result, err := uc.ExecuteAll(ctx)
		if err != nil && len(result.Feeds) == 0 {
			// フィード一覧の取得自体に失敗した場合のみエラーとして返す。個々のフィード取得の
			// 失敗は saved/skipped/errors の件数で表現するため、その場合の err は無視してよい
			// （internal/adapter/handler/web の FetchLatestHandler と同じ方針）。
			return nil, FetchOutput{}, err
		}
		return nil, FetchOutput{
			Saved:   result.TotalSaved(),
			Skipped: result.TotalSkipped(),
			Errors:  result.TotalErrors(),
		}, nil
	}
}
