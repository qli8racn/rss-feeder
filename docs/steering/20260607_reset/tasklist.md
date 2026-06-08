# タスクリスト：記事のリセット

## 実装タスク

- [x] reset usecase 実装（`internal/usecase/reset.go`）
  - `bookmarked = 0` の記事のみ削除
- [x] reset handler 実装・確認プロンプト（`internal/adapter/handler/reset.go`）
  - `-y` フラグによる確認スキップ
- [x] `rss-feeder check-bookmarked` サブコマンド実装（お気に入り件数表示用）
- [x] PreToolUse Hook スクリプトへの reset 対応（`.claude/hooks/validate-state.sh`）

## テスト

- [x] reset usecase ユニットテスト（`internal/usecase/reset_test.go`）
  - お気に入り記事が削除対象に含まれないことを確認
