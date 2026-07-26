# rss-feeder

Go 製 RSS リーダー CLI。Claude Code のエージェント・Hooks・SQLite 連携を学習するプロジェクト。

## 概要

DB に登録した RSS フィードから記事を取得・保存し、一覧表示・検索・ブックマーク管理を行う CLI ツール。
AI エージェント機能（`rss-agent`）で記事要約や読書傾向の分析も可能。

## 要件

- Go 1.24 以上
- GCC（`mattn/go-sqlite3` の CGO ビルドに必要）
- `ANTHROPIC_API_KEY`（`rss-agent` 使用時のみ。`config.yml` または環境変数で設定）
  - claude.ai（コンシューマー向けチャット）のログイン情報ではなく、Claude Console
    （https://console.anthropic.com）の「API Keys」から発行する API キーを使用する

## セットアップ

```bash
go mod tidy
go build -o bin/rss-feeder ./cmd/rss-feeder
go build -o bin/web        ./cmd/web
GOMAXPROCS=1 GOFLAGS="-gcflags=all=-l=0" go build -o bin/rss-agent -p 1 ./cmd/agent
```

DB（`rss-feeder-db/reader.db`）は初回起動時に自動作成される。

`rss-agent` を使う場合は `internal/config/config.yml` を作成し、`ANTHROPIC_API_KEY` を設定する。

```bash
cp internal/config/config.example.yml internal/config/config.yml
# internal/config/config.yml の anthropic_api_key に API キーを設定する
```

`internal/config/config.yml` は `.gitignore` 対象（機密情報を含むため Git 管理対象外）。

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
| `bin/rss-agent enrich [--limit <n>] [--force]` | 記事に要約・カテゴリを付与してDBに保存 |

## Web UI

ブラウザで記事を閲覧するための JSON API + フロントエンドを `cmd/web` で提供する（CLI とは別バイナリ）。

```bash
cd web/frontend && npm ci && npm run build  # web/static/ にビルド成果物を出力
bin/web [--port 8080] [--static-dir web/static]
```

`GET /api/articles`・`GET /api/articles/search`・`POST /api/articles/{id}/bookmark`・`GET /api/categories`・`POST /api/articles/fetch` を提供する。API の詳細は [機能設計・技術仕様](docs/design.md) を参照。

## MCPサーバーとして使う（Claude Desktop連携）

`cmd/mcp` をビルドしてClaude Desktopに登録すると、チャット上の自然言語からこのプロジェクトの
フィード管理・記事閲覧を操作できる。ビルド方法・`claude_desktop_config.json` への登録方法は
[AGENTS.md](AGENTS.md#mcp-servercmdmcp) を参照（DevContainer環境での注意点もそこに記載）。

登録すると、Claude Desktopのツール/コネクタ一覧に `rss-feeder` が表示され、以下のツールが使えるようになる。

| ツール | 説明 | 備考 |
|-------|------|------|
| `rss_list` | 保存済み記事を一覧表示（デフォルトは未読のみ。all/bookmarked・categoryで絞り込み可。all・bookmarkedは同時指定不可） | 読み取り専用（閲覧しても既読にならない）。`limit`/`page`でページネーションし、`total`で総件数を返す（デフォルト50件・上限200件） |
| `rss_search` | キーワードで記事を全文検索 | 読み取り専用。`limit`/`page`/`total`は`rss_list`と同様 |
| `rss_categories` | 記事に付与済みのカテゴリ一覧を表示 | 読み取り専用 |
| `rss_list_feeds` | 登録済みRSSフィード一覧を表示 | 読み取り専用 |
| `rss_bookmark` | 指定した記事IDのブックマークをトグル | |
| `rss_mark_read` | 指定した記事IDを既読にする | `rss_list`が既読化しないため、既読管理をしたい場合に使う |
| `rss_add_feed` | フィードURL（またはそのサイトURL）をDBに登録し、直後に記事を取得 | 数秒〜数十秒かかる場合あり |
| `rss_fetch` | 登録済みの全フィードを取得してDBに保存 | 数秒〜数十秒かかる場合あり |
| `rss_remove_feed` | 指定フィードと関連記事を完全削除 | 破壊的操作。`confirm`必須・実行前に必ず同意確認 |
| `rss_enrich` | 記事にAIで要約・カテゴリを付与してDBに保存 | `ANTHROPIC_API_KEY`課金発生。`confirm`・`limit`必須 |
| `rss_preference` | ブックマーク済み記事から読書傾向を分析 | 読み取り専用だが課金発生。`confirm`必須 |

Claude Desktopのチャットでは、例えば以下のように話しかけると各ツールが呼び出される。

- 「最近ブックマークした記事を教えて」→ `rss_list`（bookmarked絞り込み）
- 「Goに関する記事を検索して」→ `rss_search`
- 「この記事は既読にして」→ `rss_mark_read`
- 「新しいフィード https://example.com/feed.xml を登録して」→ `rss_add_feed`
- 「登録済みの全フィードを更新して」→ `rss_fetch`
- 「〇〇フィードはもう読まないから削除して」→ `rss_remove_feed`（実行前にClaudeから確認が入る）
- 「未要約の記事に要約をつけて。5件まで」→ `rss_enrich`（課金発生の確認が入る）
- 「自分の読書傾向を分析して」→ `rss_preference`（課金発生の確認が入る）

`rss_remove_feed`・`rss_enrich`・`rss_preference` は実行前にClaude側から同意確認が入る設計なので、
「はい」等で明示的に同意しない限り実行されない。

## テスト

```bash
go test ./internal/domain/...
go test ./internal/usecase/...
go test $(go list ./internal/driver/... | grep -v internal/driver/anthropic)
```

## アーキテクチャ

クリーンアーキテクチャ風の層構造で、依存方向は外側から内側への一方向。

```
adapter(handler/cli, handler/web) -> usecase -> domain
driver                            -> adapter(interface)
```

| 層 | 役割 |
|----|------|
| `domain` | エンティティ（Article・Feed・AuditLog） |
| `usecase` | ビジネスロジック（重複チェック・ブックマークトグルなど） |
| `adapter` | usecase インターフェース定義・cobra ハンドラー（`handler/cli`）・HTTP ハンドラー（`handler/web`） |
| `driver` | SQLite・HTTP・ファイル・Anthropic API の実装 |

DI には `samber/do` を使用。`cmd/rss-feeder/main.go`・`cmd/web/main.go`・`cmd/agent/main.go` がそれぞれ独立した Composition Root。

## 主な依存ライブラリ

| ライブラリ | 用途 |
|----------|------|
| `github.com/spf13/cobra` | CLI サブコマンド管理 |
| `github.com/mmcdole/gofeed` | RSS/Atom パース |
| `github.com/mattn/go-sqlite3` | SQLite ドライバ（CGO） |
| `github.com/samber/do/v2` | DI コンテナ |
| `github.com/anthropics/anthropic-sdk-go` | Claude API クライアント |
| `github.com/spf13/viper` | `config.yml` の読み込み（`ANTHROPIC_API_KEY` 等） |
| `github.com/go-chi/chi/v5` | Web UI（`cmd/web`）の HTTP ルーター |
| `github.com/go-chi/cors` | Web UI の CORS ミドルウェア |

## Hooks

Claude Code が Bash ツールで `rss-feeder` コマンドを実行する際に自動発火する。

| スクリプト | タイミング | 処理 |
|----------|----------|------|
| `.claude/hooks/validate-state.sh` | PreToolUse | `bookmark` / `reset` 実行前の ID 確認・件数表示 |
| `.claude/hooks/audit-log.sh` | PostToolUse | `fetch` / `bookmark` / `reset` 後に `audit_log` へ記録 |
| `.claude/hooks/session-cleanup.sh` | Stop | DB の VACUUM と整合性チェック |

## DB 確認

```bash
sqlite3 rss-feeder-db/reader.db "SELECT * FROM articles LIMIT 5"
sqlite3 rss-feeder-db/reader.db "SELECT * FROM audit_log ORDER BY timestamp DESC LIMIT 10"
```

## ドキュメント

- [プロダクト要求定義書](docs/product-requirements.md)
- [機能設計・技術仕様](docs/design.md)

## License

[MIT](LICENSE)
