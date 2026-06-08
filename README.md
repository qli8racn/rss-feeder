# rss-feeder

Go 製 RSS リーダー CLI。Claude Code のエージェント・Hooks・SQLite 連携を学習するプロジェクト。

## 概要

DB に登録した RSS フィードから記事を取得・保存し、一覧表示・検索・ブックマーク管理を行う CLI ツール。
AI エージェント機能（`rss-agent`）で記事要約や読書傾向の分析も可能。

## 要件

- Go 1.24 以上
- GCC（`mattn/go-sqlite3` の CGO ビルドに必要）
- `ANTHROPIC_API_KEY`（`rss-agent` 使用時のみ）

## セットアップ

```bash
go mod tidy
go build -o bin/rss-feeder ./cmd/rss-feeder
go build -o bin/rss-agent  ./cmd/agent   # メモリ不足時は -p 1 を追加
```

DB（`reader.db`）は初回起動時に自動作成される。

## RSS フィードの登録

`add-feed` コマンドでフィード URL を DB に登録する。

```bash
bin/rss-feeder add-feed https://example.com/feed.xml
bin/rss-feeder add-feed https://another.example.com/rss
```

## コマンド

### rss-feeder

| コマンド | 説明 |
|---------|------|
| `bin/rss-feeder add-feed <url>` | RSS フィード URL を DB に登録 |
| `bin/rss-feeder list-feeds` | 登録済みフィードの一覧を表示 |
| `bin/rss-feeder remove-feed <id>` | フィードを削除（関連記事も連動削除） |
| `bin/rss-feeder fetch` | 登録済みフィードを取得して DB に保存 |
| `bin/rss-feeder list` | 未読記事の一覧を表示 |
| `bin/rss-feeder list --all` | すべての記事を表示 |
| `bin/rss-feeder list --bookmarked` | ブックマーク済み記事を表示 |
| `bin/rss-feeder bookmark <id>` | 指定 ID の記事のブックマークをトグル |
| `bin/rss-feeder reset [-y]` | ブックマーク以外の記事を削除（`-y` で確認スキップ） |
| `bin/rss-feeder search <keyword>` | キーワードで記事を全文検索 |

### rss-agent

| コマンド | 説明 |
|---------|------|
| `bin/rss-agent summarize` | 最新 10 件の記事を AI で要約 |
| `bin/rss-agent summarize --feed <url> --limit <n>` | 特定フィード・件数指定で要約 |
| `bin/rss-agent preference` | ブックマーク済み記事から読書傾向を分析 |

## テスト

```bash
go test ./internal/domain/...
go test ./internal/usecase/...
go test ./internal/driver/...
```

## アーキテクチャ

クリーンアーキテクチャ風の層構造で、依存方向は外側から内側への一方向。

```
adapter(handler) -> usecase -> domain
driver           -> adapter(interface)
```

| 層 | 役割 |
|----|------|
| `domain` | エンティティ（Article・Feed・AuditLog） |
| `usecase` | ビジネスロジック（重複チェック・ブックマークトグルなど） |
| `adapter` | usecase インターフェース定義・cobra ハンドラー |
| `driver` | SQLite・HTTP・ファイル・Anthropic API の実装 |

DI には `samber/do` を使用。`cmd/rss-feeder/main.go` が Composition Root。

## 主な依存ライブラリ

| ライブラリ | 用途 |
|----------|------|
| `github.com/spf13/cobra` | CLI サブコマンド管理 |
| `github.com/mmcdole/gofeed` | RSS/Atom パース |
| `github.com/mattn/go-sqlite3` | SQLite ドライバ（CGO） |
| `github.com/samber/do/v2` | DI コンテナ |
| `github.com/anthropics/anthropic-sdk-go` | Claude API クライアント |

## Hooks

Claude Code が Bash ツールで `rss-feeder` コマンドを実行する際に自動発火する。

| スクリプト | タイミング | 処理 |
|----------|----------|------|
| `.claude/hooks/validate-state.sh` | PreToolUse | `bookmark` / `reset` 実行前の ID 確認・件数表示 |
| `.claude/hooks/audit-log.sh` | PostToolUse | `fetch` / `bookmark` / `reset` 後に `audit_log` へ記録 |
| `.claude/hooks/session-cleanup.sh` | Stop | DB の VACUUM と整合性チェック |

## DB 確認

```bash
sqlite3 reader.db "SELECT * FROM articles LIMIT 5"
sqlite3 reader.db "SELECT * FROM audit_log ORDER BY timestamp DESC LIMIT 10"
```

## ドキュメント

- [プロダクト要求定義書](docs/product-requirements.md)
- [機能設計・技術仕様](docs/design.md)
