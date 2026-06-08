# タスクリスト：取得済み記事の一覧表示

## 実装タスク

- [x] list usecase 実装（`internal/usecase/list.go`）
  - 未読のみ・全件・お気に入りのみ の 3モード
- [x] 表示後に `read = 1` へ自動更新処理
- [x] list handler 実装・テーブル出力フォーマット（`internal/adapter/handler/list.go`）
- [x] `--all` / `--bookmarked` フラグ追加

## テスト

- [x] list usecase ユニットテスト（`internal/usecase/list_test.go`）
  - 未読フィルタ・全件・お気に入りフィルタが正しく委譲されることを確認
