# タスクリスト：記事メタデータ拡充（出版元・サムネイル・要約・カテゴリ）

## 実装タスク

- [ ] `internal/migration/migration.go` に4カラム追加（`CREATE TABLE` 定義 + 既存DB向け `ALTER TABLE`、重複エラーは無視）
- [ ] `internal/domain/article.go` に `Publisher` / `ThumbnailURL` / `Summary` / `Category` を追加
- [ ] `internal/driver/rss/reader.go`
  - [ ] `extractThumbnail(item *gofeed.Item) string` 追加（media:thumbnail → enclosure(image/*) → itunes:image の優先順）
- [ ] `internal/usecase/fetch.go` で `Publisher`（フィードタイトル）・`ThumbnailURL` を `Article` にセット
- [ ] `internal/adapter/driver/readerdb/article/article.go`
  - [ ] `Repository` インターフェースに `UpdateEnrichment` / `FindWithoutSummary` を追加
- [ ] `internal/driver/readerdb/article/article.go`
  - [ ] `Save` の INSERT 文に `publisher` / `thumbnail_url` を追加
  - [ ] `scanArticle` / `scanArticleWithFeed` に `publisher` / `thumbnail_url` / `summary` / `category` を追加
  - [ ] `UpdateEnrichment` 実装
  - [ ] `FindWithoutSummary` 実装
- [ ] `internal/driver/anthropic/enrich.go` 新規作成（`EnrichAgent` / `EnrichOptions{Limit, Force}`）
- [ ] `cmd/agent/main.go` に `enrich` サブコマンド追加（`--limit` / `--force`）
- [ ] `internal/adapter/handler/serve.go` の Article DTO に4フィールド追加
- [ ] `internal/adapter/handler/list.go` / `search.go` の表示を `Publisher` に変更（未設定時は `FeedURL` フォールバック）

## テスト

- [ ] `internal/driver/rss/reader_test.go`: `extractThumbnail` の各優先順位パターン
- [ ] `internal/driver/readerdb/article/article_test.go`: `Save` で publisher/thumbnail_url が保存されること、`UpdateEnrichment` / `FindWithoutSummary` の統合テスト
- [ ] `internal/usecase/fetch_test.go`: `Publisher` / `ThumbnailURL` が `Article` にセットされること

## フォローアップ（このフェーズ外）

- [ ] フロントエンドでのサムネイル表示・カテゴリ表示
- [ ] カテゴリ別フィルタ・一覧コマンド
- [ ] 既存記事への `publisher` / `thumbnail_url` バックフィル
