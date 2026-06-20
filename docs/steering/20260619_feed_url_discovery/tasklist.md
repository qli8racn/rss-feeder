# タスクリスト：フィードURL自動探索

## 共通ドライバ

- [x] `golang.org/x/net` を `go.mod` で直接依存に変更（`golang.org/x/net/html` を使用。`findFeedLink` 実装時に対応）
- [x] `internal/adapter/driver/htmlfetch/htmlfetch.go` 新規作成（`Fetcher` interface）
- [x] `internal/driver/htmlfetch/htmlfetch.go` 新規作成（タイムアウト15秒の実装。レスポンスサイズ上限5MBも追加）
- [x] `internal/driver/htmlfetch/htmlfetch_test.go` 新規作成（`httptest` 統合テスト）

## Claude を直接呼ぶ側（`cmd/agent` ・ `cmd/rss-feeder` 共通の Go パッケージ）

- [x] `internal/adapter/driver/anthropic/discover_feed.go` 新規作成（`FeedDiscoveryAgent` interface）
- [x] `internal/driver/anthropic/discover_feed.go` 新規作成（`claude-haiku-4-5`、`<head>` 抽出・truncate）
- [x] `internal/usecase/discover_feed.go` 新規作成（`DiscoverFeedUsecase`）
- [x] `internal/usecase/discover_feed_test.go` 新規作成
- [x] `internal/adapter/handler/agent/discover_feed.go` 新規作成（`discover-feed <url>` コマンド。`rss-agent` 単独実行用）
- [x] `cmd/agent/main.go` に DI 登録・コマンド追加（`adapterrss.RSSReader` ・ `htmlfetch.Fetcher` の DI 登録も追加）

## resolve_feed_url（`cmd/web` ・ `cmd/rss-feeder` 共通）

- [x] `internal/usecase/resolve_feed_url.go` 新規作成（`ResolveFeedURLUsecase`・`findFeedLink` 純粋関数・`ErrFeedNotFound`。ステップ3呼び出しに30秒タイムアウトを設定、各ステップの失敗はログに残す）
- [x] `internal/usecase/resolve_feed_url_test.go` 新規作成
- [x] `internal/adapter/handler/cli/add_feed.go` 修正（`ResolveFeedURLUsecase` → `AddFeedUsecase` の順で呼び出す）
- [x] `internal/adapter/handler/web/feed.go` 修正（`ResolveFeedURLUsecase` → `AddFeedUsecase` の順で呼び出す。タイムアウトを 504 に、それ以外の解決失敗を 400 にマッピング。`addFeedTimeout`=80秒を `AddFeedHandler` 内で設定）

## cmd/web 側（サブプロセス経由、同時実行数制限）

- [x] `internal/adapter/driver/feeddiscovery/feeddiscovery.go` 新規作成（`Agent` interface、`ErrAgentUnavailable`）
- [x] `internal/driver/feeddiscovery/subprocess.go` 新規作成（セマフォで同時実行数を制限した上で `exec.CommandContext` で `bin/rss-agent discover-feed` 実行）
- [x] `internal/driver/feeddiscovery/subprocess_test.go` 新規作成（同時実行数の上限挙動を含む）
- [x] `cmd/web/main.go` に DI 登録・`--rss-agent-path`（デフォルト `bin/rss-agent`）・`--feed-discovery-concurrency`（デフォルト2）フラグ追加

## cmd/rss-feeder 側（インプロセス、Anthropic SDK に直接依存）※2026-06-20 に下記「アーキテクチャ変更」で撤回

- [x] `cmd/rss-feeder/main.go` に `htmlfetch.Fetcher` ・ `adapteranthropic.FeedDiscoveryAgent` ・ `DiscoverFeedUsecase` を DI 登録（`config.Load()` で `ANTHROPIC_API_KEY` を設定する処理も `cmd/agent/main.go` から移植）
- [x] `cmd/rss-feeder/main.go` 内に `DiscoverFeedUsecase` を `feeddiscovery.Agent` として渡すための小さなアダプタ型（`discoverFeedAgent`）を定義（専用パッケージは作らない。コンポジションルートのため層違反にならない。`var _ feeddiscovery.Agent = (*discoverFeedAgent)(nil)` でコンパイル時にインターフェース充足を確認）
- [x] `cmd/rss-feeder/main.go` に `ResolveFeedURLUsecase` を DI 登録し `add-feed` サブコマンドに渡す（`--rss-agent-path` 等のフラグは不要）

## アーキテクチャ変更：cmd/rss-feeder もサブプロセス経由に統一（2026-06-20）

Anthropic SDK への直接依存を `cmd/agent` に一本化する方針に変更したため、上記「cmd/rss-feeder 側（インプロセス）」を撤回し、
`cmd/web` と同じサブプロセス実装に差し替える（`docs/steering/20260619_feed_url_discovery/design.md` 改訂箇所を参照）。

- [ ] `cmd/rss-feeder/main.go` から `adapteranthropic`・`driveranthropic`・`config.SetupAnthropicAPIKey()` 呼び出し・`discoverFeedAgent` アダプタ型・`discoverFeedUC` を削除
- [ ] `cmd/rss-feeder/main.go` に `--rss-agent-path`（デフォルト `bin/rss-agent`）フラグを追加し、`driverfeeddiscovery.NewSubprocessAgent(*rssAgentPath, 1)` を構築して `ResolveFeedURLUsecase` に渡す
- [ ] `go build ./...` / `go vet ./...` / `go test ./...` で確認

## OpenAPI・ドキュメント

- [x] `docs/openapi.yaml` の `POST /api/feeds` に `504` レスポンスを追加（`components/responses/GatewayTimeout` を新規定義）
- [x] `go generate ./internal/adapter/handler/web/openapi/...` で型再生成（`GatewayTimeout` 型を追加）
- [x] `cd web/frontend && npm run generate:api` で型再生成（フロントエンドの手書きコードは変更不要。`tsc --noEmit`・`npm run test` で確認済み）
- [x] `docs/design.md` のコマンド一覧・パッケージ構成・依存ライブラリ表を更新（`/api/feeds` 系エンドポイント・`discover-feed` コマンド・新規ファイルツリー・`golang.org/x/net` 等を追加。フィード管理UIフェーズで未反映だった `/api/feeds` 系も合わせて追記）

## 確認

- [x] `go build ./...` / `go vet ./...` / `go test ./...`
- [x] `go build -o bin/rss-agent ./cmd/agent` `go build -o bin/web ./cmd/web` `go build -o bin/rss-feeder ./cmd/rss-feeder`
- [x] `curl` で `POST /api/feeds`（`cmd/web`）の実機能確認（ローカルでテスト起動し検証。完了後サーバーは停止済み）
  - [x] RSS/Atomフィードの直接URL（`https://hnrss.org/frontpage`）→ステップ1で即解決
  - [x] `<link rel="alternate">` を持つ通常サイトURL（`https://www.theverge.com/`）→ステップ2で相対URL（`/rss/index.xml`）を絶対URLに解決
  - [x] 標準探索では見つからないURL（`https://qiita.com`）→ステップ3でClaudeが呼ばれる（`$0.0206`課金を確認）ことを確認。
    今回は推測URL（`/feed.atom`）が実際には404で存在せず、再検証で弾かれて400になった（「Claudeの推測が常に正しいわけではない」を再検証ステップが正しく防いでいることの確認にもなった）
  - [ ] 標準探索では見つからないが Claude が実際に正しいフィードURLを発見できる想定のURLでの成功パターン（未確認。上記の通り簡単には見つからなかったため保留）
  - [ ] フィードが存在しないURL（全ステップ失敗 → 400）の単独確認（上記qiita.comのケースで結果的に確認済みだが、フィード自体が存在しないケースは未確認）
  - [ ] `bin/rss-agent` を一時的にリネームした状態での通常サイトURL追加（`ErrAgentUnavailable`経路）
  - [ ] `--feed-discovery-concurrency` の上限を超える同時リクエストの挙動
- [ ] `bin/rss-feeder add-feed <url>` での同様の確認（`bin/rss-agent` 削除状態でのインプロセスAI探索）

> 残りの未確認項目は、外部サイトへの追加アクセスとAPIコストを伴うため保留した（ユーザー判断）。
> 本番運用前に必要であれば手動で確認すること。
