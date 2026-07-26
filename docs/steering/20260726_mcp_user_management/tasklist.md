# タスクリスト：MCPサーバー利用者（クライアント）単位でのフィード管理

## docs/steering

- [ ] `docs/steering/20260726_mcp_user_management/requirements.md` 作成
- [ ] `docs/steering/20260726_mcp_user_management/design.md` 作成

## DBスキーマ・マイグレーション

- [ ] `rss-feeder-db/schema.sql` に `users` テーブル・`feeds.user_id`・`UNIQUE(user_id, feed_url)` を反映する（新規DB作成時の正とする定義）
- [ ] `rss-feeder-db/20260726_add_user_management.sh` を新設し、`users` テーブル作成・デフォルトユーザー作成・`feeds.user_id` 追加・既存データのデフォルトユーザーへの紐付け・`UNIQUE(user_id, feed_url)` へのテーブル再作成を行う
- [ ] `internal/migration/migration.go` に `addUserManagement(db)` を追加し、`Run()` から呼び出す（アプリ起動時の自動マイグレーション）
- [ ] マイグレーションの冪等性（複数回実行しても安全であること）を確認する
- [ ] マイグレーション前後でのデータ損失有無を確認するテスト（既存フィード・記事がすべてデフォルトユーザーに紐付くこと）を追加する

## ドメイン・リポジトリ層

- [ ] `internal/domain/user.go` に `User` 構造体・`DefaultUserName` 定数を追加する
- [ ] `internal/adapter/driver/readerdb/user/user.go` に `Repository` インターフェース（`FindByName`・`Create`）を追加する
- [ ] `internal/driver/readerdb/user/user.go` に `Repository` の実装を追加する
- [ ] `internal/adapter/driver/readerdb/feed/feed.go` の `Repository` インターフェースの各メソッドに `userID` 引数を追加する
- [ ] `internal/driver/readerdb/feed/feed.go` の実装を `userID` でスコープするSQLに変更する（`Save`・`FindByURL`・`Register`・`ListAll`・`Remove`）
- [ ] `internal/adapter/driver/readerdb/article/article.go` の `Repository` インターフェースの各メソッドに `userID` 引数を追加する
- [ ] `internal/driver/readerdb/article/article.go` の実装を `JOIN feeds ON ... AND feeds.user_id = ?` でスコープするSQLに変更する
- [ ] 上記リポジトリの既存テスト（`*_test.go`）をuserIDスコープに対応させて更新する

## Usecase層

- [ ] `internal/usecase/resolve_user.go` に `ResolveUserUsecase`（find-or-createでユーザーを解決する）を追加する
- [ ] `list`・`search`・`list_categories`・`list_feeds`・`bookmark`・`mark_read`・`add_feed`・`remove_feed`・`fetch`・`backfill_metadata`・`reset`・`check_article`・`check_bookmarked` の各Usecaseコンストラクタに `userID` を追加し、内部のrepo呼び出しに反映する
- [ ] `internal/driver/anthropic/preference.go`・`curate.go`・`discover.go` の各Agent実装のコンストラクタに `userID` を追加し、内部のrepo呼び出し（`FindBookmarked`・`ListAll`）に反映する
- [ ] 上記Usecase・Agentの既存テストをuserIDスコープに対応させて更新する

## `cmd/mcp` の変更

- [ ] `cmd/mcp/main.go` に `--user-id` フラグ（デフォルト値 `default`）を追加する
- [ ] migration実行の直後に `ResolveUserUsecase.Execute(ctx, *userIDFlag)` を呼び出し、解決済みuserIDを取得する
- [ ] DIコンテナに解決済みuserIDを登録し（専用型 `usecase.UserID` 等）、各Usecase・Agentのコンストラクタ呼び出しに反映する
- [ ] `go build -o bin/mcp ./cmd/mcp` が通ることを確認する

## `cmd/rss-feeder`・`cmd/web`・`cmd/agent` の変更

- [ ] `cmd/rss-feeder/main.go` で `ResolveUserUsecase.Execute(ctx, domain.DefaultUserName)` を呼び出し、各Usecaseコンストラクタに反映する
- [ ] `cmd/web/main.go` で同様の対応を行う
- [ ] `cmd/agent/main.go` で同様の対応を行う（`preference`・`curate`・`discover`・`enrich`・`summarize` の各Agent）
- [ ] CLI・Web UIの既存コマンド・APIレスポンスが変更前と同一の挙動になることを確認する（ハンドラ層は無変更）

## 動作確認

- [ ] `cmd/mcp --user-id alice` と `cmd/mcp --user-id bob` を別々に起動し、それぞれのフィード・記事・ブックマークが完全に分離されることを確認する
- [ ] 既存の `reader.db`（マイグレーション前のデータ）に対してマイグレーションを実行し、CLI/Web UIから見て既存フィード・記事がすべて閲覧できることを確認する（デフォルトユーザーへの紐付け確認）
- [ ] 同一の `--user-id` で複数回起動しても同じユーザーとして扱われる（フィードが重複作成されない）ことを確認する

## ドキュメント整備

- [ ] `AGENTS.md` の `MCP Server（cmd/mcp）` 節に `--user-id` フラグの説明を追記する
- [ ] `claude_desktop_config.json` の設定例に `--user-id` の指定方法を追記する
