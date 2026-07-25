# 要件：構造化ログ基盤の導入

## 背景・目的

現在、`internal/driver/anthropic/` 配下の各ドライバ（`enrich.go`・`curate.go`・`discover.go`・`usage.go`）では `fmt.Fprintf(os.Stderr, "[command] message\n")` 形式のプレーンテキストログを使用しており、以下の課題がある。

- フォーマット・出力先がハードコードされており、環境ごとに切り替えられない
- 構造化されていないため、ログ集約ツール（Loki 等）でのクエリが困難
- グローバルな暗黙的出力に依存しており、テストでの検証がしにくい

`log/slog`（Go 1.21 以降の標準ライブラリ）を使った構造化ログ基盤を整備し、今後のログ追加が一貫したインターフェースで行えるようにする。

## 要件

### 設定項目（`config.yml` への追加）

- `log.output`：ログの出力先を指定する
  - `stderr`（デフォルト）: 標準エラー出力
  - `stdout`: 標準出力
  - ファイルパス文字列（例: `logs/agent.log`）: 指定パスのファイルに追記
- `log.format`：ログのフォーマットを指定する
  - `text`（デフォルト）: `slog.TextHandler` によるキーバリュー形式
  - `json`: `slog.JSONHandler` による JSON 形式

### DI 注入

- `*slog.Logger` を DI コンテナ（`samber/do/v2`）経由で提供する
- 各 anthropic ドライバの構造体フィールドとして注入する（グローバル変数・`slog.SetDefault` は使用しない）
- `NewXxxAgent` コンストラクタで `do.MustInvoke[*slog.Logger]` により取得する

### 既存ログの置き換え

- `enrich.go` の `fmt.Fprintf(os.Stderr, ...)` 呼び出しを `logger.Info(...)` / `logger.Error(...)` に置き換える
- `usage.go` の `logUsage` 関数内の `fmt.Fprintf(os.Stderr, ...)` を `logger.Info(...)` に置き換える
- `curate.go`・`discover.go` 内の `fmt.Fprintf(os.Stderr, ...)` 呼び出しを `logger` 経由に置き換える

## 制約・前提

- Go バージョン: 1.24.3（`log/slog` は 1.21 で標準化済みのため追加依存不要）
- 外部ロギングライブラリ（zerolog・zap・logrus 等）は導入しない
- ログレベルは `Info` / `Error` の2段階を基本とし、デバッグレベルは今フェーズのスコープ外
- 設定ファイルのパースは既存の `internal/config/` の仕組みに乗せる
- `slog.Logger` の提供は DI コンテナの provider として実装し、`cmd/agent/main.go` の起動時に一度だけ構築する

## 完了条件

- `config.yml` に `log.output` / `log.format` を追加でき、起動時に反映される
- `*slog.Logger` が DI コンテナから取得でき、各 anthropic ドライバに注入される
- `internal/driver/anthropic/` 内の `fmt.Fprintf(os.Stderr, ...)` 呼び出しがすべて `slog.Logger` 経由に置き換わっている
- `go build -p 1 -o bin/rss-agent ./cmd/agent` が通る
- `log.format: json` を設定した場合、出力が JSON 形式になる
- `log.output` にファイルパスを指定した場合、そのファイルにログが書き込まれる
