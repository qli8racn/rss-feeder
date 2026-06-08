# 設計：フィード管理の SQLite 移行

## 対象コマンド

```bash
rss-feeder add-feed <url>
rss-feeder list-feeds
rss-feeder remove-feed <id>
```

## 処理フロー

```
add-feed <url>
  └─ handler/add_feed.go
       └─ usecase/add_feed.go
            └─ feedrepo.Repository.Register()
                 └─ driver/readerdb/feed/feed.go
                      └─ INSERT INTO feeds ON CONFLICT DO NOTHING
                           └─ RowsAffected() == 0 → ErrAlreadyExists

list-feeds
  └─ handler/list_feeds.go
       └─ usecase/list_feeds.go
            └─ feedrepo.Repository.ListAll()
                 └─ driver/readerdb/feed/feed.go
                      └─ SELECT id, feed_url, COALESCE(title,''), ... ORDER BY created_at

remove-feed <id>
  └─ handler/remove_feed.go
       └─ usecase/remove_feed.go
            └─ feedrepo.Repository.Remove()
                 └─ driver/readerdb/feed/feed.go
                      └─ DELETE FROM feeds WHERE id = ?
                           └─ RowsAffected() == 0 → ErrNotFound
```

## エラー定義

| sentinel | 定義場所 | 意味 |
|---------|---------|------|
| `ErrAlreadyExists` | `internal/adapter/driver/readerdb/feed/feed.go` | 同一 URL が既に登録済み |
| `ErrNotFound` | `internal/adapter/driver/readerdb/feed/feed.go` | 指定 ID が存在しない |

## 関連ファイル

### 追加ファイル

| ファイル | 役割 |
|---------|------|
| `internal/usecase/add_feed.go` | フィード登録ユースケース |
| `internal/usecase/list_feeds.go` | フィード一覧ユースケース |
| `internal/usecase/remove_feed.go` | フィード削除ユースケース |
| `internal/adapter/handler/add_feed.go` | `add-feed` cobra コマンド |
| `internal/adapter/handler/list_feeds.go` | `list-feeds` cobra コマンド |
| `internal/adapter/handler/remove_feed.go` | `remove-feed` cobra コマンド |
| `internal/usecase/add_feed_test.go` | add_feed ユースケースユニットテスト |
| `internal/usecase/list_feeds_test.go` | list_feeds ユースケースユニットテスト |
| `internal/usecase/remove_feed_test.go` | remove_feed ユースケースユニットテスト |

### 変更ファイル

| ファイル | 変更内容 |
|---------|---------|
| `internal/adapter/driver/readerdb/feed/feed.go` | `Register` / `ListAll` / `Remove` をインターフェースに追加、sentinel エラー定義 |
| `internal/driver/readerdb/feed/feed.go` | 上記メソッドを実装、コンパイル時インターフェース適合チェック追加 |
| `internal/driver/readerdb/feed/feed_test.go` | `Register` / `ListAll` / `Remove` の統合テスト追加 |
| `internal/adapter/handler/fetch.go` | `feedrepo.Repository` 直接依存を `*ListFeedsUsecase` に変更 |
| `cmd/rss-feeder/main.go` | 3コマンド登録、file ドライバ削除 |
| `.claude/skills/add-feed/SKILL.md` | `feeds.txt` 書き込みから `bin/rss-feeder add-feed` 実行に変更 |

### 削除ファイル

| ファイル | 理由 |
|---------|------|
| `feeds.txt` | DB 管理に移行済み |
| `internal/adapter/driver/file/feeds_reader.go` | 不要になった file ドライバ interface |
| `internal/driver/file/feeds_reader.go` | 不要になった file ドライバ実装 |

## アーキテクチャ上の注意点

- `fetch` ハンドラは `feedrepo.Repository` を直接受け取らず `*ListFeedsUsecase` を受け取る（adapter → driver 直接依存の禁止）
- `list_feeds.go` で定義した `msgNoFeeds` 定数を `fetch.go` と共有することで DRY を維持
