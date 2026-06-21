package anthropic

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// messageCreator は anthropic.MessageService.New を抽象化したインターフェース。
// summarize/preference/enrich の各エージェントが共有し、テストで実APIを呼ばずに
// fakeへ差し替えられるようにするために用意する。
type messageCreator interface {
	New(ctx context.Context, body anthropic.MessageNewParams, opts ...option.RequestOption) (*anthropic.Message, error)
}

// maxAPIRetries は429（レート制限）・5xx・接続エラー時にSDKが自動でリトライする最大回数。
// SDKは標準でRetry-After（またはRetry-After-Ms）ヘッダーに従った指数バックオフで
// リトライする仕組みを内蔵しているため、アプリ側でバックオフ処理を自前実装する必要はない
// （参照: anthropic-sdk-go/internal/requestconfig/requestconfig.go の shouldRetry/retryDelay）。
// SDKのデフォルト値（2回）はenrichの並列バッチ実行（同時に複数リクエストが飛ぶ）で
// レート制限に達しやすいことを踏まえ、新規記事の自動enrich等で人手によるリトライが
// 難しい場面でも自然に吸収できるよう、デフォルトより手厚く設定する。
const maxAPIRetries = 5

// newAnthropicClient は全エージェント共通のリトライ設定を適用した anthropic.Client を返す。
func newAnthropicClient() anthropic.Client {
	return anthropic.NewClient(option.WithMaxRetries(maxAPIRetries))
}

// modelPricing は 1M トークンあたりの USD 単価（入力・出力）。
// 価格改定時はここを更新する。
var modelPricing = map[string]struct{ Input, Output float64 }{
	anthropic.ModelClaudeHaiku4_5:  {Input: 1.00, Output: 5.00},
	anthropic.ModelClaudeSonnet4_6: {Input: 3.00, Output: 15.00},
	anthropic.ModelClaudeOpus4_8:   {Input: 5.00, Output: 25.00},
}

// estimateCostUSD は Usage から概算費用（USD）を計算する。
// cache_read は通常入力の約0.1倍、cache_creation は約1.25倍として概算する。
// 単価が登録されていないモデルは 0 を返す。
func estimateCostUSD(model string, u anthropic.Usage) float64 {
	price, ok := modelPricing[model]
	if !ok {
		return 0
	}
	input := float64(u.InputTokens) + float64(u.CacheCreationInputTokens)*1.25 + float64(u.CacheReadInputTokens)*0.1
	output := float64(u.OutputTokens)
	return (input*price.Input + output*price.Output) / 1_000_000
}

// addUsage は2つの Usage の各トークン数を加算した結果を返す。
// runAgentLoop でのツール呼び出しループ全体のトークン使用量を合算するために使う。
func addUsage(total, delta anthropic.Usage) anthropic.Usage {
	total.InputTokens += delta.InputTokens
	total.OutputTokens += delta.OutputTokens
	total.CacheCreationInputTokens += delta.CacheCreationInputTokens
	total.CacheReadInputTokens += delta.CacheReadInputTokens
	return total
}

// logUsage はトークン使用量・概算費用・実行時間を標準エラー出力に1行で記録する。
// 標準出力はコマンドの主たる出力（要約結果など）に使うため、ログはそちらを汚さない標準エラー出力に出す。
func logUsage(model string, usage anthropic.Usage, elapsed time.Duration) {
	fmt.Fprintf(os.Stderr, "[rss-agent] model=%s input=%d output=%d cost=$%.4f elapsed=%s\n",
		model, usage.InputTokens, usage.OutputTokens, estimateCostUSD(model, usage), elapsed.Round(time.Millisecond))
}

// runAgentLoopWithUsageLog は runAgentLoop を実行し、完了後にトークン使用量・概算費用・
// 実行時間を logUsage で標準エラー出力に記録する。preference.go/summarize.go の
// 計測用ボイラープレート（計測開始・Usage集計・defer ログ出力）を共通化するためのラッパー。
func runAgentLoopWithUsageLog(ctx context.Context, client messageCreator, params anthropic.MessageNewParams, handler toolHandler) (string, error) {
	start := time.Now()
	result, usage, err := runAgentLoop(ctx, client, params, handler)
	logUsage(params.Model, usage, time.Since(start))
	return result, err
}
