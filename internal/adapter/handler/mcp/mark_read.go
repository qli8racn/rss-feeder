package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// MarkReadInput は mark-read ツールの入力。
type MarkReadInput struct {
	IDs []int64 `json:"ids" jsonschema:"既読にする記事IDの配列。1件以上指定すること"`
}

// MarkReadOutput は mark-read ツールの出力。
type MarkReadOutput struct {
	IDs []int64 `json:"ids"`
}

// MarkReadTool は指定した記事IDを既読にする mark-read ツールのハンドラを構築する。
// rss_list は閲覧そのものによる既読化を行わない読み取り専用ツールのため、MCP経由で
// 明示的に既読管理を行うための唯一の手段としてこのツールを設ける。
func MarkReadTool(uc *usecase.MarkReadUsecase) mcpsdk.ToolHandlerFor[MarkReadInput, MarkReadOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, input MarkReadInput) (*mcpsdk.CallToolResult, MarkReadOutput, error) {
		if err := uc.Execute(ctx, input.IDs); err != nil {
			return nil, MarkReadOutput{}, err
		}
		return nil, MarkReadOutput{IDs: input.IDs}, nil
	}
}
