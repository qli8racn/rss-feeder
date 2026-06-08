# タスクリスト：記事の取得と標準出力

## 実装タスク

- [x] RSSReader interface 定義（`internal/adapter/driver/rss/rss_reader.go`）
- [x] gofeed による RSS 2.0 / Atom パース実装（`internal/driver/rss/reader.go`）
- [x] fetch usecase 実装（`internal/usecase/fetch.go`）
- [x] fetch handler 実装・出力フォーマット（`internal/adapter/handler/fetch.go`）
- [x] フィードアクセス失敗時のエラーハンドリングと続行処理

## テスト

- [x] RSSReader ユニットテスト（`internal/driver/rss/reader_test.go`）
  - httptest.NewServer でモック RSS フィードを返す
  - RSS 2.0 / Atom のパース確認
- [x] fetch usecase ユニットテスト（`internal/usecase/fetch_test.go`）
  - 新規記事の保存・重複スキップの確認
