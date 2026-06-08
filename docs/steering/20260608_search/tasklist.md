# タスクリスト：記事の検索

## 実装タスク

- [ ] ArticleRepository に `Search(keyword string, bookmarkedOnly bool) ([]Article, error)` 追加
  - `internal/adapter/driver/readerdb/article/article.go`（interface）
  - `internal/driver/readerdb/article/article.go`（SQL 実装）
- [ ] search usecase 実装（`internal/usecase/search.go`）
- [ ] search handler 実装（`internal/adapter/handler/search.go`）
  - `--bookmarked` フラグ
  - 0件時「該当記事が見つかりませんでした」出力
- [ ] main.go へ search コマンド登録（`cmd/rss-feeder/main.go`）

## テスト

- [ ] search usecase ユニットテスト（`internal/usecase/search_test.go`）
  - キーワード一致・0件・`--bookmarked` フィルタの確認
- [ ] driver 統合テスト（`internal/driver/readerdb/article/article_test.go`）
  - LIKE 検索の SQL 動作確認
