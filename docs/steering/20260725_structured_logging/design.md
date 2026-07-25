# 設計：構造化ログ基盤の導入

## 方針

`log/slog`（Go 1.21 標準）を採用し、外部ライブラリへの依存を増やさない。
`*slog.Logger` を DI コンテナ経由で各 anthropic ドライバに注入し、グローバル変数・`slog.SetDefault` は使用しない。

## 設定スキーマ

`internal/config/config.yml` に `log` セクションを追加する。

```yaml
log:
  output: stderr     # stderr | stdout | ファイルパス（追記モード）
  format: text       # text | json
```

`Config` 構造体に `Log LogConfig` フィールドを追加する。

```go
type LogConfig struct {
    Output string `mapstructure:"output"`
    Format string `mapstructure:"format"`
}

type Config struct {
    AnthropicAPIKey string    `mapstructure:"anthropic_api_key"`
    Log             LogConfig `mapstructure:"log"`
}
```

## DI プロバイダの配置

`internal/driver/logger/logger.go` に `NewLogger(i do.Injector) (*slog.Logger, error)` を実装する。

処理フロー:
1. `do.MustInvoke[*config.Config](i)` で設定を取得
2. `config.Log.Output` に応じて `io.Writer` を決定（stderr / stdout / ファイル）
3. `config.Log.Format` に応じて `slog.TextHandler` / `slog.JSONHandler` を選択
4. ファイル出力時は `os.OpenFile` で追記モードで open し、`do.ShutdownOnStop` でクローズ登録

`cmd/agent/main.go` で以下を追加:
```go
do.Provide(i, config.NewProvider)          // *config.Config を提供
do.Provide(i, driverlogger.NewLogger)      // *slog.Logger を提供
```

## anthropic ドライバへの注入

`enrichAgent`・`curateAgent`・`discoverAgent` の各構造体に `logger *slog.Logger` フィールドを追加し、`NewXxxAgent` で `do.MustInvoke[*slog.Logger](i)` により取得する。

## ログ出力の変換方針

| 変更前 | 変更後 |
|---|---|
| `fmt.Fprintf(os.Stderr, "[enrich] 開始しています...（対象: %d件）\n", n)` | `logger.Info("開始しています...", "command", "enrich", "count", n)` |
| `fmt.Fprintf(os.Stderr, "[enrich] フルテキスト取得: %d/%d件成功\n", s, t)` | `logger.Info("フルテキスト取得", "command", "enrich", "success", s, "total", t)` |
| `fmt.Fprintf(os.Stderr, "[curate] 開始しています...\n")` | `logger.Info("開始しています...", "command", "curate")` |
| `fmt.Fprintln(os.Stderr, "[curate] 直近記事: 0件")` | `logger.Info("直近記事", "command", "curate", "count", 0)` |
| `logUsage(model, usage, elapsed)` | `logUsage(logger, model, usage, elapsed)` |

## テスト

既存テストの `enrichAgent` 初期化に `logger: slog.New(slog.NewTextHandler(io.Discard, nil))` を追加する（テスト出力を汚さないよう Discard を使う）。
