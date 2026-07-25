# タスクリスト：構造化ログ基盤の導入

## docs/steering

- [x] `docs/steering/20260725_structured_logging/requirements.md` 作成
- [x] `docs/steering/20260725_structured_logging/design.md` 作成

## 設定

- [x] `internal/config/config.go`（または既存の設定構造体）に `Log` セクション（`Output`・`Format` フィールド）を追加
- [x] `internal/config/config.yml` に `log` セクションのデフォルト値を追加

## DI プロバイダ

- [x] `*slog.Logger` を構築して DI コンテナに登録する provider を実装（`internal/driver/logger/logger.go`）
- [x] `config.Log.Output` に応じて出力先（stderr / stdout / ファイル）を切り替える
- [x] `config.Log.Format` に応じて `slog.TextHandler` / `slog.JSONHandler` を選択する

## anthropic ドライバへの注入

- [x] `internal/driver/anthropic/enrich.go` — `enrichAgent` に `logger *slog.Logger` フィールドを追加、`NewEnrichAgent` で DI 注入
- [x] `internal/driver/anthropic/curate.go` — `curateAgent` に `logger *slog.Logger` フィールドを追加、`NewCurateAgent` で DI 注入
- [x] `internal/driver/anthropic/discover.go` — `discoverAgent` に `logger *slog.Logger` フィールドを追加、`NewDiscoverAgent` で DI 注入

## 既存ログの置き換え

- [x] `enrich.go` の `fmt.Fprintf(os.Stderr, "[enrich] 開始しています...（対象: %d件）\n", ...)` を `logger.Info` に置き換え
- [x] `enrich.go` の `fmt.Fprintf(os.Stderr, "[enrich] フルテキスト取得: %d/%d件成功\n", ...)` を `logger.Info` に置き換え
- [x] `usage.go` の `logUsage` 関数を `*slog.Logger` を引数に取る形に変更し、`fmt.Fprintf(os.Stderr, ...)` を `logger.Info` に置き換え
- [x] `curate.go`・`discover.go` 内の `fmt.Fprintf(os.Stderr, ...)` 呼び出しを `logger.Info` / `logger.Error` に置き換え

## ビルド・動作確認

- [x] `go build -p 1 -o bin/rss-agent ./cmd/agent` が通ること
- [x] `log.format: json` 設定でログが JSON 形式で出力されること
- [x] `log.output` にファイルパスを設定し、ファイルへの書き込みが確認できること
- [x] `log.output: stdout` 設定で標準出力にログが出ること
