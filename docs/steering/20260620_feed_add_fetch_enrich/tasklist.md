# タスクリスト：フィード追加時の自動取得・要約

## enrich のフィード絞り込み対応

- [ ] `internal/adapter/driver/anthropic/enrich.go` の `EnrichOptions` に `FeedURL string` フィールドを追加
- [ ] `internal/driver/anthropic/enrich.go` の `Run` 内 `Force` 分岐を `a.repo.FetchLatest(ctx, opts.Limit, opts.FeedURL)` に変更
- [ ] `internal/driver/anthropic/enrich_test.go` に `FeedURL` 指定時のテストケースを追加
- [ ] `internal/adapter/handler/agent/enrich.go` の `NewEnrichCommand` に `--feed` フラグを追加し `EnrichOptions.FeedURL` に渡す

## enrich 用サブプロセス機構（新設）

- [ ] `internal/adapter/driver/feedenrich/feedenrich.go` 新規作成（`Agent` interface、`ErrAgentUnavailable`）
- [ ] `internal/driver/feedenrich/subprocess.go` 新規作成（`exec.CommandContext` で `bin/rss-agent enrich --feed <url> --force --limit <n>` 実行。同時実行数制限は無し）
- [ ] `internal/driver/feedenrich/subprocess_test.go` 新規作成
- [ ] `internal/usecase/trigger_enrich.go` 新規作成（`TriggerEnrichUsecase`。`feedenrich.Agent` への薄いラッパー。
      `enrichTimeout`=30秒（`resolve_feed_url.go` の `stepThreeTimeout` と同じ考え方）を内部で `context.WithTimeout` に設定）
- [ ] `internal/usecase/trigger_enrich_test.go` 新規作成（`enrichTimeout` 経過時に `ctx.Err()` が `agent.Enrich` に
      伝播することを含めてテスト）

## cmd/rss-feeder（CLI）: 自動 fetch + 自動 enrich（サブプロセス経由）

- [ ] `internal/adapter/handler/cli/add_feed.go` の `NewAddFeedCommand` に `fetchUC`・`triggerEnrichUC` を追加し、
      登録成功後に fetch→（保存件数>0なら）enrichトリガーを呼ぶ（失敗は標準エラー出力に警告、処理は続行）
- [ ] `cmd/rss-feeder/main.go` に `--rss-agent-path`（デフォルト `bin/rss-agent`）フラグを追加し、
      `feeddiscovery.Agent`・`feedenrich.Agent` を両方サブプロセス実装で構築（Anthropic SDK 関連の import は撤去。
      `docs/steering/20260619_feed_url_discovery/tasklist.md` の「アーキテクチャ変更」と合わせて実施する）

## cmd/web: 自動 fetch のみ

- [ ] `internal/adapter/handler/web/feed.go` の `AddFeedHandler` に `fetchUC` を追加し、
      登録成功後に fetch を呼ぶ（失敗は `log.Printf` で記録、処理は続行。enrich は呼ばない）
- [ ] `cmd/web/main.go` の `web.AddFeedHandler(...)` 呼び出しに既存の `fetchUC` を追加で渡す

## 確認

- [ ] `go build ./...` / `go vet ./...` / `go test ./...`
- [ ] `bin/rss-agent enrich --force --limit 5`（`--feed` 未指定）が従来通り全フィード対象で動作することを確認（後方互換）
- [ ] `bin/rss-feeder add-feed <url>` で、フィード登録→記事取得→要約・カテゴライズ（サブプロセス経由）が連続して実行され、
      `bin/rss-feeder list` で要約・カテゴリ付きの記事が確認できることを確認
- [ ] `curl` で `POST /api/feeds` を実行し、レスポンスが既存と同じ（`Feed` DTO・`201`）であること、
      かつ記事が自動取得されること（要約・カテゴリは付与されないこと）を確認
- [ ] `bin/rss-agent` を一時的にリネームした状態で `add-feed` を実行し、フィード登録・記事取得は成功し、
      enrich失敗の警告のみ標準エラー出力に出ることを確認
