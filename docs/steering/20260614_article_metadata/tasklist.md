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
    `size-10` の角丸サムネイル画像を追加
  - 2026-06-21: `thumbnail_url` 未設定時・画像読み込み失敗時のフォールバックとして、
    既存の`RssIcon`（`atoms/icons.tsx`）を使ったプレースホルダー枠（空状態スケルトンと同系統の
    `bg-surface-raised/30`・`border-slate-400/10`）を常時表示するよう変更。実画像はその上に
    `absolute`配置で重ね、読み込み失敗時はimgを非表示にしてプレースホルダーを見せる
  - `npx tsc -b --noEmit` / `npm run test` / `npm run build` で確認
- [x] カテゴリ別フィルタ・一覧コマンド
  - Web UI のカテゴリフィルタ（`SearchFilterBar`・`GET /api/articles?category=`）は別フェーズで実装済みだった
  - CLI 側（`rss-feeder`）には無かったため追加：`list`/`search` に `--category` フラグを追加し、
    既存の `usecase.ListUsecase.ExecuteFiltered` / `SearchUsecase.ExecuteFiltered`（Web側で既に実装済み）を
    流用（CLI 用に `cliListPerPage = 10000` で実質ページネーション無しの一覧を取得）
  - 新規 `rss-feeder categories` コマンドを追加（既存の `usecase.ListCategoriesUsecase` を CLI から呼び出す。
    元々 Web API（`GET /api/categories`）専用で CLI からは未登録だった）
  - `printArticleTable`（`internal/adapter/handler/cli/table.go`）にカテゴリ列を追加
- [x] 既存記事への `publisher` / `thumbnail_url` バックフィル
  - `articlerepo.Repository` に `UpdateMetadataBatch`（URLで記事を特定し、`publisher`/`thumbnail_url` が
    空文字の列のみ列単位で補完。既存値は上書きしない。何度実行しても安全）を追加
  - `usecase.BackfillMetadataUsecase` 新規作成：登録済み全フィードを再取得し、`UpdateMetadataBatch` で補完
    （RSSフィードは最新の一部記事しか配信しないため、フィードから外れた古い記事は対象外になる制約あり）
  - 新規 `rss-feeder backfill-metadata` コマンドを追加
  - `go build ./...` / `go vet ./...` / `go test $(go list ./... | grep -v internal/driver/anthropic)` で確認
  - 実機確認：実データで実行し、出版元が未設定だった既存記事（135件）が補完されることを確認
    （サムネイルは登録済みフィードがいずれも提供していなかったため0件のまま。仕様通りの挙動）
  - 2026-06-21: 実運用で2件の不整合を発見・修正
    1. サムネイルを提供しないフィード（今回の全フィード）に対して`backfill-metadata`を再実行すると、
       新しい値も空文字のまま`WHERE`句にマッチし続け、毎回「補完 N件」と誤報告されていた。
       `UpdateMetadataBatch`のSQLを「新しい値が空でない場合のみ」マッチするよう修正
       （`AND :publisher != ''`等を追加。パラメータも`sql.Named`に変更し可読性を確保）
    2. `migration.go`の`ALTER TABLE ... ADD COLUMN`で追加した既存行は`publisher`/`thumbnail_url`が
       空文字ではなくNULLになるため、`publisher = ''`の比較ではNULL行（実データで81件）を検出できていなかった。
       `COALESCE(publisher, '') = ''`に修正してNULLも空として扱うようにした
    - 上記2点を再現するテストを追加（`TestArticleRepository_UpdateMetadataBatch_SkipsWhenNewValueAlsoEmpty`・
      `TestArticleRepository_UpdateMetadataBatch_FillsNullColumns`）
    - 修正後もNULL記事のうち実際に補完できたのは0件だった（該当記事がいずれも2026-06-08時点の古い記事で、
      現在のRSSフィードの配信範囲外のため。ドキュメント記載済みの既知の制約であり新たな問題ではない）
