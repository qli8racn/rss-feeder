package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// CategoriesInput は categories ツールの入力（引数なし）。
type CategoriesInput struct{}

// CategoriesOutput は categories ツールの出力。
type CategoriesOutput struct {
	Categories []string `json:"categories"`
}

// CategoriesTool は記事に付与済みのカテゴリ一覧を返す categories ツールのハンドラを構築する。
func CategoriesTool(uc *usecase.ListCategoriesUsecase) mcpsdk.ToolHandlerFor[CategoriesInput, CategoriesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ CategoriesInput) (*mcpsdk.CallToolResult, CategoriesOutput, error) {
		categories, err := uc.Execute(ctx)
		if err != nil {
			return nil, CategoriesOutput{}, err
		}
		return nil, CategoriesOutput{Categories: categories}, nil
	}
}
