package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// PreferenceInput は preference ツールの入力。
type PreferenceInput struct {
	// Confirm はユーザーへの明示的な同意確認なしに true を渡してはならない
	// （ANTHROPIC_API_KEYによる追加課金が発生するため、ツールの description で同意確認を必須化している）。
	Confirm bool `json:"confirm" jsonschema:"true の場合のみ実行する。ANTHROPIC_API_KEYによる追加課金が発生するため、ユーザーへの明示的な同意確認なしにtrueを渡してはならない"`
}

// PreferenceOutput は preference ツールの出力。
type PreferenceOutput struct {
	Analysis string `json:"analysis"`
}

// PreferenceTool はブックマーク済み記事から趣向を分析する preference ツールのハンドラを構築する。
// confirm != true の場合は usecase を呼び出さずにエラーを返す。
func PreferenceTool(uc *usecase.PreferenceUsecase) mcpsdk.ToolHandlerFor[PreferenceInput, PreferenceOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, input PreferenceInput) (*mcpsdk.CallToolResult, PreferenceOutput, error) {
		if err := requireConfirm(input.Confirm); err != nil {
			return nil, PreferenceOutput{}, err
		}
		analysis, err := uc.Execute(ctx)
		if err != nil {
			return nil, PreferenceOutput{}, err
		}
		return nil, PreferenceOutput{Analysis: analysis}, nil
	}
}
