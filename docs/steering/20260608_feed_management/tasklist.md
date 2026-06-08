# タスクリスト：フィード管理の SQLite 移行

## 実装タスク

- [x] `feedrepo.Repository` インターフェースに `Register` / `ListAll` / `Remove` を追加
  - `internal/adapter/driver/readerdb/feed/feed.go`
  - sentinel エラー `ErrAlreadyExists` / `ErrNotFound` 定義
- [x] driver 実装（`internal/driver/readerdb/feed/feed.go`）
  - `Register`: `INSERT ... ON CONFLICT DO NOTHING` + `RowsAffected()` チェック
  - `ListAll`: `SELECT ... ORDER BY created_at` + `COALESCE(title, '')`
  - `Remove`: `DELETE WHERE id = ?` + `RowsAffected()` チェック
  - コンパイル時インターフェース適合チェック追加
- [x] add_feed usecase 実装（`internal/usecase/add_feed.go`）
- [x] list_feeds usecase 実装（`internal/usecase/list_feeds.go`）
- [x] remove_feed usecase 実装（`internal/usecase/remove_feed.go`）
- [x] add-feed handler 実装（`internal/adapter/handler/add_feed.go`）
- [x] list-feeds handler 実装（`internal/adapter/handler/list_feeds.go`）
- [x] remove-feed handler 実装（`internal/adapter/handler/remove_feed.go`）
- [x] fetch ハンドラの依存を `feedrepo.Repository` → `*ListFeedsUsecase` に変更
- [x] main.go に 3 コマンド登録・file ドライバ依存削除
- [x] Claude Code `add-feed` スキルを `bin/rss-feeder add-feed` 実行に変更

## 削除タスク

- [x] `feeds.txt` 削除
- [x] `internal/adapter/driver/file/feeds_reader.go` 削除
- [x] `internal/driver/file/feeds_reader.go` 削除

## テスト

- [x] usecase ユニットテスト（mock ベース）
  - `internal/usecase/add_feed_test.go`（正常系・重複・DB エラー）
  - `internal/usecase/list_feeds_test.go`（正常系・空・DB エラー）
  - `internal/usecase/remove_feed_test.go`（正常系・未存在・DB エラー）
- [x] driver 統合テスト（in-memory SQLite）
  - `internal/driver/readerdb/feed/feed_test.go`
  - `Register`・`ListAll`・`Remove` の各 happy path・error path を追加
