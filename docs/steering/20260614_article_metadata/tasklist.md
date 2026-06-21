# タスクリスト：記事メタデータ拡充（出版元・サムネイル・要約・カテゴリ）

## 実装タスク

- [x] `internal/migration/migration.go` に4カラム追加（`CREATE TABLE` 定義 + 既存DB向け `ALTER TABLE`、重複エラーは無視）
- [x] `internal/domain/article.go` に `Publisher` / `ThumbnailURL` / `Summary` / `Category` を追加
- [x] `internal/driver/rss/reader.go`
  - [x] `extractThumbnail(item *gofeed.Item) string` 追加（media:thumbnail → enclosure(image/*) → itunes:image の優先順）
  - [x] `Fetch` 内で `Publisher`（フィードタイトル）・`ThumbnailURL` を `Article` にセット（usecase/fetch.go の変更は不要だった）
- [x] `internal/adapter/driver/readerdb/article/article.go`
  - [x] `Repository` インターフェースに `UpdateEnrichment` / `FindWithoutSummary` を追加
- [x] `internal/driver/readerdb/article/article.go`
  - [x] `Save` の INSERT 文に `publisher` / `thumbnail_url` を追加
  - [x] `scanArticle` / `scanArticleWithFeed` に `publisher` / `thumbnail_url` / `summary` / `category` を追加
  - [x] `UpdateEnrichment` 実装
  - [x] `FindWithoutSummary` 実装
- [x] `internal/driver/anthropic/enrich.go` 新規作成（`EnrichAgent` / `EnrichOptions{Limit, Force}`）
- [x] `cmd/agent/main.go` に `enrich` サブコマンド追加（`--limit` / `--force`）
- [x] `internal/adapter/handler/serve.go` の Article DTO に4フィールド追加
- [x] `internal/adapter/handler/list.go` / `search.go` の表示を `Publisher` に変更（`table.go` の `printArticleTable` に出版元列を追加、未設定時は `FeedURL` フォールバック）

## テスト

- [x] `internal/driver/rss/reader_test.go`: `extractThumbnail` の各優先順位パターン（`TestFetch_Thumbnail`）
- [x] `internal/driver/readerdb/article/article_test.go`: `Save` で publisher/thumbnail_url が保存されること、`UpdateEnrichment` / `FindWithoutSummary` の統合テスト
- [x] `internal/usecase/fetch_test.go`: 既存モックの interface 互換性を維持（Publisher/ThumbnailURL は reader 層のテストでカバー）

## フォローアップ（このフェーズ外）

> 2026-06-20: 一旦「対応しない」と決定したが、2026-06-21 にユーザー判断で対応する方針に変更（3件を個別に対応）。

- [x] フロントエンドでのサムネイル表示・カテゴリ表示
  - カテゴリ表示は別フェーズ（`20260619_feed_management_ui`・`20260620_design_token_cleanup`）で
    `CategoryBadge` として既に実装済みだった（本タスクリストが更新されていなかった）
  - サムネイル表示は Figma に対応デザインが存在しない（`docs/web-ui-spec.md` の「スコープ外」記載どおり、
    `ArticleListPage > ArticleTable` の実画面フレームを確認しても表示スペースはない）。
    ユーザー判断でデザインなしの簡易実装とした：`ArticleTable.tsx` でタイトル左に
    `size-10` の角丸サムネイル画像を追加（`thumbnail_url` 未設定時は非表示、画像読み込み失敗時も非表示）
  - `npx tsc -b --noEmit` / `npm run test` / `npm run build` で確認
- [ ] ~~カテゴリ別フィルタ・一覧コマンド~~（対応しない）
- [ ] ~~既存記事への `publisher` / `thumbnail_url` バックフィル~~（対応しない）
