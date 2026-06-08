# タスクリスト：お気に入り登録

## 実装タスク

- [x] Article.ToggleBookmark() 実装（`internal/domain/article.go`）
- [x] bookmark usecase 実装（`internal/usecase/bookmark.go`）
- [x] bookmark handler 実装（`internal/adapter/handler/bookmark.go`）
- [x] `rss-feeder check-article <id>` サブコマンド実装（ID 存在確認用）
- [x] PreToolUse Hook スクリプトへの bookmark 対応（`.claude/hooks/validate-state.sh`）
- [x] audit-log.sh への bookmark 記録対応（`--article-id` 引数）

## テスト

- [x] domain ユニットテスト（`internal/domain/article_test.go`）
  - ToggleBookmark() の状態遷移確認
- [x] bookmark usecase ユニットテスト（`internal/usecase/bookmark_test.go`）
  - トグル動作・存在しない ID のエラーハンドリング
