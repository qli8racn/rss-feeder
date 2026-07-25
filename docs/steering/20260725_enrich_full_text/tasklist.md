# タスクリスト：enrich 本文取得改善

## docs/steering

- [x] `docs/steering/20260725_enrich_full_text/requirements.md` 作成
- [x] `docs/steering/20260725_enrich_full_text/design.md` 作成
- [x] `docs/steering/20260725_enrich_full_text/tasklist.md` 作成

## 実装

- [x] `internal/driver/anthropic/enrich.go`
  - `enrichAgent` に `fetcher adapterhtmlfetch.Fetcher` フィールド追加
  - `NewEnrichAgent` に `do.MustInvoke[adapterhtmlfetch.Fetcher](i)` 追加
  - `maxFetchConcurrency = 5` 定数追加
  - `fetchFullContent(ctx, articles) []domain.Article` メソッド追加
  - `extractTextFromHTML(html string) string` 関数追加
  - `Run` メソッド内の goroutine で `fetchFullContent` を呼び出す
- [x] `go.mod` — `goquery` を direct 依存に昇格

## テスト・確認

- [x] `go build -p 1 -o bin/rss-agent ./cmd/agent` でビルドが通ること
- [x] `bin/rss-agent enrich --force --limit 3` を実行し、従来より詳細な要約が生成されること
