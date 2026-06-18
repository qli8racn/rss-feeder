# 設計：エージェント系コマンドの再確認とリファクタリング

## 1. 現状の振り返り（フェーズ開始時点）

| ファイル | モデル | Thinking | 件数上限 | 備考 |
|---|---|---|---|---|
| `enrich.go` | `claude-haiku-4-5` | なし | `opts.Limit`（既定 10）、本文 2000 文字に切り詰め | 単発リクエスト（ツール未使用） |
| `summarize.go` | `claude-haiku-4-5` | なし | `opts.Limit`（既定 10） | `fetch_articles` ツール経由 |
| `preference.go` | `claude-opus-4-8` | `adaptive` | `fetch_bookmarked` の `limit`（既定・上限 50） | 趣向分析という性質上、モデル・Thinking は意図的に維持 |

`summarize`/`enrich` は単純な要約・分類タスクのため Haiku 4.5 に変更済み。`preference` は
ブックマークの傾向を洞察する分析タスクであり、要約系より難易度が高いと判断し Opus 4.8 +
adaptive thinking を維持する。本フェーズではこの判断が妥当か再確認しつつ、可視化（費用・実行時間）
を主目的とする。

## 2. ANTHROPIC_API_KEY の取得元の明記

`README.md` の前提条件セクションと `AGENTS.md` のセットアップ手順に以下を追記する。

```markdown
> `ANTHROPIC_API_KEY` は claude.ai（コンシューマー向けチャット）のログイン情報ではなく、
> Claude Console（https://console.anthropic.com）の「API Keys」から発行する API キーを使用する。
```

`internal/config/config.example.yml` のコメントにも同様の注記を追加する。

## 3. 費用・実行時間の計測

### 3.1 料金テーブル

`internal/driver/anthropic/usage.go`（新規）に、使用するモデルの USD/1M トークン単価を保持する。

```go
package anthropic

// modelPricing は 1M トークンあたりの USD 単価（入力・出力）。
// 価格改定時はここを更新する。
var modelPricing = map[string]struct{ Input, Output float64 }{
    "claude-haiku-4-5": {Input: 1.00, Output: 5.00},
    "claude-sonnet-4-6": {Input: 3.00, Output: 15.00},
    "claude-opus-4-8":   {Input: 5.00, Output: 25.00},
}

// estimateCostUSD は Usage から概算費用（USD）を計算する。
// cache_read は通常入力の約 0.1 倍、cache_creation は約 1.25 倍として概算する。
func estimateCostUSD(model string, u anthropic.Usage) float64 {
    price, ok := modelPricing[model]
    if !ok {
        return 0
    }
    input := float64(u.InputTokens) + float64(u.CacheCreationInputTokens)*1.25 + float64(u.CacheReadInputTokens)*0.1
    output := float64(u.OutputTokens)
    return (input*price.Input + output*price.Output) / 1_000_000
}
```

### 3.2 実行時間計測と出力フォーマット

各エージェントの `Run()` の冒頭で `start := time.Now()` を取り、`defer` で1行のサマリーを
**標準エラー出力**に出す（標準出力は要約結果などコマンドの主たる出力に使うため汚さない）。

```go
defer func() {
    fmt.Fprintf(os.Stderr, "[rss-agent] model=%s input=%d output=%d cost=$%.4f elapsed=%s\n",
        model, usage.InputTokens, usage.OutputTokens, estimateCostUSD(model, usage), time.Since(start).Round(time.Millisecond))
}()
```

- `enrich.go`：単発リクエストのため `resp.Usage` をそのまま使う。
- `summarize.go`/`preference.go`：`runAgentLoop`（`loop.go`）がツール呼び出しで複数回
  `Messages.New` を呼ぶため、ループ内で `Usage` を加算して `runAgentLoop` の戻り値に含める
  必要がある。`runAgentLoop` の戻り値を `(string, anthropic.Usage, error)` に拡張する。

### 3.3 `runAgentLoop` の変更点（概要）

```go
func runAgentLoop(ctx context.Context, client anthropic.Client, params anthropic.MessageNewParams, handler toolHandler) (string, anthropic.Usage, error) {
    var total anthropic.Usage
    messages := params.Messages
    for {
        params.Messages = messages
        resp, err := client.Messages.New(ctx, params)
        if err != nil {
            return "", total, err
        }
        total.InputTokens += resp.Usage.InputTokens
        total.OutputTokens += resp.Usage.OutputTokens
        total.CacheCreationInputTokens += resp.Usage.CacheCreationInputTokens
        total.CacheReadInputTokens += resp.Usage.CacheReadInputTokens
        // ...既存のツール呼び出し処理...
    }
}
```

呼び出し元（`summarize.go`/`preference.go`）はこの加算済み `Usage` を使ってコスト・実行時間を
出力する。

## 4. 表示の既定動作

フラグでの切り替えは行わず、常に標準エラー出力へ1行出すデフォルト動作とする
（CI・cron 実行時のログにも残るようにするため）。標準出力（要約結果など）には影響しない。
