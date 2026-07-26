# タスクリスト：MCPサーバー利用者（クライアント）単位でのフィード管理

## docs/steering

- [x] `docs/steering/20260726_mcp_user_management/requirements.md` 作成
- [x] `docs/steering/20260726_mcp_user_management/design.md` 作成

## DBスキーマ・マイグレーション

- [x] `rss-feeder-db/schema.sql` に `users` テーブル・`feeds.user_id`・`UNIQUE(user_id, feed_url)` を反映する（新規DB作成時の正とする定義）
- [x] `rss-feeder-db/20260726_add_user_management.sh` を新設し、`users` テーブル作成・デフォルトユーザー作成・`feeds.user_id` 追加・既存データのデフォルトユーザーへの紐付け・`UNIQUE(user_id, feed_url)` へのテーブル再作成を行う
- [x] `internal/migration/migration.go` に `addUserManagement(db)` を追加し、`Run()` から呼び出す（アプリ起動時の自動マイグレーション）
- [x] マイグレーションの冪等性（複数回実行しても安全であること）を確認する（`TestRun_IsIdempotent` 等）
- [x] マイグレーション前後でのデータ損失有無を確認するテスト（既存フィード・記事がすべてデフォルトユーザーに紐付くこと）を追加する（`TestRun_ExistingDataMigratesToDefaultUser`）

補足: 実装時に、`articles.url` が元々グローバルなUNIQUE制約だったため、`feeds` が「1ユーザー1購読=1行」に
なったことで複数ユーザーが同じ外部フィードURLを購読すると2人目以降の記事保存がブロックされる問題が
判明した。design.md には明記されていなかったが、`articles` も `UNIQUE(feed_id, url)` へテーブル再作成
する対応を追加した（`rss-feeder-db/20260726_add_user_management.sh`・`internal/migration/migration.go`
の両方に反映済み）。design.md の追記要否は @pm に確認すること。

## ドメイン・リポジトリ層

- [x] `internal/domain/user.go` に `User` 構造体・`DefaultUserName` 定数を追加する
- [x] `internal/adapter/driver/readerdb/user/user.go` に `Repository` インターフェース（`FindByName`・`Create`）を追加する
- [x] `internal/driver/readerdb/user/user.go` に `Repository` の実装を追加する
- [x] `internal/adapter/driver/readerdb/feed/feed.go` の `Repository` インターフェースの各メソッドに `userID` 引数を追加する
- [x] `internal/driver/readerdb/feed/feed.go` の実装を `userID` でスコープするSQLに変更する（`Save`・`FindByURL`・`Register`・`ListAll`・`Remove`）
- [x] `internal/adapter/driver/readerdb/article/article.go` の `Repository` インターフェースの各メソッドに `userID` 引数を追加する
- [x] `internal/driver/readerdb/article/article.go` の実装を `userID` でスコープするSQL（`feed_id IN (SELECT id FROM feeds WHERE user_id = ?)`）に変更する
- [x] 上記リポジトリの既存テスト（`*_test.go`）をuserIDスコープに対応させて更新する（他ユーザーのデータが見えない・操作できないことを確認するテストを追加）

補足: design.md の記載では `Search`・`DistinctCategories` は userID スコープ対象の列挙から漏れていたが、
要件（記事検索・カテゴリ一覧もユーザーごとに分離）を満たすため実装では両メソッドにも `userID` を追加した。

## Usecase層

- [x] `internal/usecase/resolve_user.go` に `ResolveUserUsecase`（find-or-createでユーザーを解決する）を追加する
- [x] `list`・`search`・`list_categories`・`list_feeds`・`bookmark`・`mark_read`・`add_feed`・`remove_feed`・`fetch`・`backfill_metadata`・`reset`・`check_article`・`check_bookmarked` の各Usecaseコンストラクタに `userID` を追加し、内部のrepo呼び出しに反映する
- [x] `internal/driver/anthropic/preference.go`・`curate.go`・`discover.go` の各Agent実装のコンストラクタに `userID` を追加し、内部のrepo呼び出し（`FindBookmarked`・`ListAll`）に反映する（DIコンテナには `usecase.UserID` ではなく既存の `*domain.User` を登録する方式を採用。理由は各`main.go`の実装メモを参照）
- [x] 上記Usecase・Agentの既存テストをuserIDスコープに対応させて更新する（Agent側は新規に `agent_userid_test.go` を追加）

## `cmd/mcp` の変更

- [x] `cmd/mcp/main.go` に `--user-id` フラグ（デフォルト値 `default`）を追加する
- [x] migration実行の直後に `ResolveUserUsecase.Execute(ctx, *userIDFlag)` を呼び出し、解決済みuserIDを取得する
- [x] DIコンテナに解決済みuserIDを登録し（`do.ProvideValue(i, user)` で `*domain.User` を登録）、各Usecase・Agentのコンストラクタ呼び出しに反映する
- [x] `go build -o bin/mcp ./cmd/mcp` が通ることを確認する

## `cmd/rss-feeder`・`cmd/web`・`cmd/agent` の変更

- [x] `cmd/rss-feeder/main.go` で `ResolveUserUsecase.Execute(ctx, domain.DefaultUserName)` を呼び出し、各Usecaseコンストラクタに反映する
- [x] `cmd/web/main.go` で同様の対応を行う
- [x] `cmd/agent/main.go` で同様の対応を行う（`preference`・`curate`・`discover`・`enrich`・`summarize` の各Agent）。あわせて `cmd/agent/main.go` に欠けていた `migration.Run(db)` 呼び出しを追加した
- [x] CLI・Web UIの既存コマンド・APIレスポンスが変更前と同一の挙動になることを確認する（ハンドラ層は無変更。`go build`・`go test` で確認）

## 動作確認

- [x] `cmd/mcp --user-id alice` と `cmd/mcp --user-id bob` を別々に起動し、それぞれのフィード・記事・ブックマークが完全に分離されることを確認する（ユニットテストでの検証に加え、実バイナリでの起動確認も実施。ユーザーレコードがそれぞれ独立に作成されることを確認済み）
- [x] 既存の `reader.db`（マイグレーション前のデータ）に対してマイグレーションを実行し、CLI/Web UIから見て既存フィード・記事がすべて閲覧できることを確認する（デフォルトユーザーへの紐付け確認。`TestRun_ExistingDataMigratesToDefaultUser` で自動テスト化）
- [x] 同一の `--user-id` で複数回起動しても同じユーザーとして扱われる（フィードが重複作成されない）ことを確認する（実バイナリでの起動確認・`TestResolveUserUsecase_ReturnsExistingUser`・`TestRun_IsIdempotent` で確認）

## ドキュメント整備

- [x] `AGENTS.md` の `MCP Server（cmd/mcp）` 節に `--user-id` フラグの説明を追記する
- [x] `claude_desktop_config.json` の設定例に `--user-id` の指定方法を追記する
