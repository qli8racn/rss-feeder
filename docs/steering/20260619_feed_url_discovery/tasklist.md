# タスクリスト：フィードURL自動探索

## 共通ドライバ

- [ ] `golang.org/x/net` を `go.mod` で直接依存に変更（`golang.org/x/net/html` を使用。`findFeedLink` 実装時に対応）
- [x] `internal/adapter/driver/htmlfetch/htmlfetch.go` 新規作成（`Fetcher` interface）
- [x] `internal/driver/htmlfetch/htmlfetch.go` 新規作成（タイムアウト15秒の実装。レスポンスサイズ上限5MBも追加）
- [x] `internal/driver/htmlfetch/htmlfetch_test.go` 新規作成（`httptest` 統合テスト）

## Claude を直接呼ぶ側（`cmd/agent` ・ `cmd/rss-feeder` 共通の Go パッケージ）

- [ ] `internal/adapter/driver/anthropic/discover_feed.go` 新規作成（`FeedDiscoveryAgent` interface）
- [ ] `internal/driver/anthropic/discover_feed.go` 新規作成（`claude-haiku-4-5`、`<head>` 抽出・truncate）
- [ ] `internal/usecase/discover_feed.go` 新規作成（`DiscoverFeedUsecase`）
- [ ] `internal/usecase/discover_feed_test.go` 新規作成
- [ ] `internal/adapter/handler/agent/discover_feed.go` 新規作成（`discover-feed <url>` コマンド。`rss-agent` 単独実行用）
- [ ] `cmd/agent/main.go` に DI 登録・コマンド追加

## resolve_feed_url（`cmd/web` ・ `cmd/rss-feeder` 共通）

- [ ] `internal/usecase/resolve_feed_url.go` 新規作成（`ResolveFeedURLUsecase`・`findFeedLink` 純粋関数・`ErrFeedNotFound`。ステップ3呼び出しに30秒タイムアウトを設定、各ステップの失敗はログに残す）
- [ ] `internal/usecase/resolve_feed_url_test.go` 新規作成
- [ ] `internal/adapter/handler/cli/add_feed.go` 修正（`ResolveFeedURLUsecase` → `AddFeedUsecase` の順で呼び出す）
- [ ] `internal/adapter/handler/web/feed.go` 修正（同上、`ErrFeedNotFound` を 400 に、タイムアウトを 504 にマッピング）

## cmd/web 側（サブプロセス経由、同時実行数制限）

- [ ] `internal/adapter/driver/feeddiscovery/feeddiscovery.go` 新規作成（`Agent` interface、`ErrAgentUnavailable`）
- [ ] `internal/driver/feeddiscovery/subprocess.go` 新規作成（セマフォで同時実行数を制限した上で `exec.CommandContext` で `bin/rss-agent discover-feed` 実行）
- [ ] `internal/driver/feeddiscovery/subprocess_test.go` 新規作成（同時実行数の上限挙動を含む）
- [ ] `cmd/web/main.go` に DI 登録・`--rss-agent-path`（デフォルト `bin/rss-agent`）・`--feed-discovery-concurrency`（デフォルト2）フラグ追加・`AddFeedHandler` 呼び出しに80秒の `context.WithTimeout` を追加

## cmd/rss-feeder 側（インプロセス、Anthropic SDK に直接依存）

- [ ] `cmd/rss-feeder/main.go` に `htmlfetch.Fetcher` ・ `adapteranthropic.FeedDiscoveryAgent` ・ `DiscoverFeedUsecase` を DI 登録
- [ ] `cmd/rss-feeder/main.go` 内に `DiscoverFeedUsecase` を `feeddiscovery.Agent` として渡すための小さなアダプタ型を定義（専用パッケージは作らない。コンポジションルートのため層違反にならない）
- [ ] `cmd/rss-feeder/main.go` に `ResolveFeedURLUsecase` を DI 登録し `add-feed` サブコマンドに渡す（`--rss-agent-path` 等のフラグは不要）

## OpenAPI・ドキュメント

- [ ] `docs/openapi.yaml` の `POST /api/feeds` に `504` レスポンスを追加
- [ ] `go generate ./internal/adapter/handler/web/openapi/...` で型再生成
- [ ] `cd web/frontend && npm run generate:api` で型再生成（フロントエンドの手書きコードは変更不要の想定）
- [ ] `docs/design.md` のコマンド一覧・パッケージ構成・依存ライブラリ表を更新

## 確認

- [ ] `go build ./...` / `go vet ./...` / `go test ./...`
- [ ] `go build -o bin/rss-agent ./cmd/agent` `go build -o bin/web ./cmd/web` `go build -o bin/rss-feeder ./cmd/rss-feeder`
- [ ] `curl` で `POST /api/feeds`（`cmd/web`）に以下を入力し、それぞれ期待通りの結果になることを確認
  - RSS/Atomフィードの直接URL（ステップ1で即解決）
  - `<link rel="alternate">` を持つ通常サイトURL（ステップ2で解決）
  - 標準探索では見つからないが Claude なら見つかる想定のURL（ステップ3で解決、`bin/rss-agent` 必要）
  - フィードが存在しないURL（全ステップ失敗 → 400）
  - `bin/rss-agent` を一時的にリネームした状態での通常サイトURL追加（ステップ3が`ErrAgentUnavailable`で失敗し全滅 → 400、エラーで落ちないことを確認）
  - `--feed-discovery-concurrency` の上限を超える数のステップ3対象リクエストを同時に投げ、後続リクエストが空きを待ってから処理されること（タイムアウトしてもクラッシュしないこと）を確認
- [ ] `bin/rss-feeder add-feed <url>` で同様のURLパターンを確認（`bin/rss-agent` を削除した状態でも、インプロセス実装のため通常サイトURLのAI探索が機能することを確認）
