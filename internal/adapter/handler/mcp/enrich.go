package mcp

import (
	"context"
	"errors"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// ErrEnrichLimitRequired は limit に1以上の値が指定されなかった場合に返す
// （無制限実行による課金の暴走を防ぐため、limit を必須引数としている）。
var ErrEnrichLimitRequired = errors.New("limit には1以上の値を指定してください")

// maxEnrichLimit・maxEnrichBatchSize・maxEnrichConcurrency は、confirm による同意確認1回で
// 際限なく課金・API負荷が発生するのを防ぐためのサーバー側の上限クランプ。LLM が limit を必須で
// 指定しても、その値自体に上限がなければ暴走防止の意図が実効化しないため、ここで頭打ちにする。
const (
	maxEnrichLimit       = 100
	maxEnrichBatchSize   = 100
	maxEnrichConcurrency = 5
)

func clampMax(v, upper int) int {
	if v > upper {
		return upper
	}
	return v
}

// EnrichInput は enrich ツールの入力。
type EnrichInput struct {
	// Limit は処理件数の上限（必須）。無制限実行を避けるため省略を許可しない。
	Limit int `json:"limit" jsonschema:"処理する記事件数の上限。無制限実行を避けるため必ず1以上の値を指定すること（サーバー側で100件に上限クランプされる）"`
	// Confirm はユーザーへの明示的な同意確認なしに true を渡してはならない
	// （ANTHROPIC_API_KEYによる追加課金が発生するため、ツールの description で同意確認を必須化している）。
	Confirm     bool   `json:"confirm" jsonschema:"true の場合のみ実行する。ANTHROPIC_API_KEYによる追加課金が発生するため、ユーザーへの明示的な同意確認なしにtrueを渡してはならない"`
	Force       bool   `json:"force,omitempty" jsonschema:"要約済みの記事も含め、最新記事を対象に再処理する"`
	FeedURL     string `json:"feed_url,omitempty" jsonschema:"対象を絞り込むフィードURL（省略時は全フィード対象）"`
	BatchSize   int    `json:"batch_size,omitempty" jsonschema:"1回のAPI呼び出しで処理する記事数（省略時はデフォルト値を使う）"`
	Concurrency int    `json:"concurrency,omitempty" jsonschema:"バッチの最大同時実行数（省略時はデフォルト値を使う）"`
}

// EnrichOutput は enrich ツールの出力。
type EnrichOutput struct {
	Processed int `json:"processed"`
}

// EnrichTool は記事に要約・カテゴリを付与する enrich ツールのハンドラを構築する。
// confirm != true、または limit が1未満の場合は usecase を呼び出さずにエラーを返す。
func EnrichTool(uc *usecase.EnrichUsecase) mcpsdk.ToolHandlerFor[EnrichInput, EnrichOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, input EnrichInput) (*mcpsdk.CallToolResult, EnrichOutput, error) {
		if err := requireConfirm(input.Confirm); err != nil {
			return nil, EnrichOutput{}, err
		}
		if input.Limit < 1 {
			return nil, EnrichOutput{}, ErrEnrichLimitRequired
		}

		n, err := uc.Execute(ctx, adapteranthropic.EnrichOptions{
			Limit:       clampMax(input.Limit, maxEnrichLimit),
			Force:       input.Force,
			FeedURL:     input.FeedURL,
			BatchSize:   clampMax(input.BatchSize, maxEnrichBatchSize),
			Concurrency: clampMax(input.Concurrency, maxEnrichConcurrency),
		})
		if err != nil {
			return nil, EnrichOutput{}, err
		}
		return nil, EnrichOutput{Processed: n}, nil
	}
}
