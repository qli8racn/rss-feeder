# タスクリスト：設定ファイルによる ANTHROPIC_API_KEY 管理

## 実装タスク

- [x] `internal/config/config.example.yml` を作成（`anthropic_api_key` のサンプル値）
- [x] `.gitignore` に `/internal/config/config.yml` を追加
- [x] `internal/config/config.go` を作成（viper で `config.yml` を読み込み `Config` にマッピング）
- [x] `cmd/agent/main.go` で `config.Load()` を呼び、`AnthropicAPIKey` を `ANTHROPIC_API_KEY` 環境変数にセット
- [x] `go mod tidy`（viper を直接依存に整理）

## テスト

- [x] `internal/config/config_test.go`
  - `config.yml` が存在しない場合、ゼロ値の `Config` を返す
  - `config.yml` から `anthropic_api_key` を読み込める

## ドキュメント

- [x] `AGENTS.md` のセットアップ手順に `config.yml` 生成手順を追記
- [x] `README.md` のセットアップ手順・必要要件を更新
- [x] `docs/design.md` に `internal/config` パッケージの説明を追記

## フォローアップ（このフェーズ外）

> 2026-06-20: 以下は対応しない方針に決定（ユーザー判断）。

- [ ] ~~`rss-feeder`（`cmd/rss-feeder/main.go`）側で `config.yml` を使う設定項目が増えた場合の対応~~（対応しない）
