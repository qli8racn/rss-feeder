# フェーズ 11：設定ファイルによる ANTHROPIC_API_KEY 管理

## 概要

`rss-agent` が利用する `ANTHROPIC_API_KEY` を、環境変数だけでなく `internal/config/config.yml`
（リポジトリ非管理）で管理できるようにする。

## 背景・目的

- これまでは `ANTHROPIC_API_KEY` 環境変数を毎回シェルにセットする必要があった
- `internal/config/config.yml` に API キーを保存しておき、`rss-agent` 実行時に自動で読み込めるようにする
- 設定ファイルは機密情報を含むため Git 管理対象外とし、テンプレート（`config.example.yml`）のみコミットする

## 受け入れ条件

- `internal/config/config.example.yml` を用意する（`anthropic_api_key` のサンプル値を記載）
- `internal/config/config.yml` は `.gitignore` に追加し、Git の追跡対象外にする
- セットアップ時、`config.yml` が存在しなければ `config.example.yml` をコピーして生成する
- `internal/config` パッケージで `config.yml` を読み込み、`Config` 構造体（`AnthropicAPIKey` フィールド）にマッピングする
  - `config.yml` が存在しない場合はゼロ値の `Config` を返す（エラーにしない）
- `rss-agent`（`cmd/agent/main.go`）起動時に `config.Load()` を実行し、`AnthropicAPIKey` が設定されていれば
  `ANTHROPIC_API_KEY` 環境変数にセットしてから各エージェントを初期化する
  - 既存の「環境変数 `ANTHROPIC_API_KEY` を直接セットして実行する」運用も変わらず動作する
