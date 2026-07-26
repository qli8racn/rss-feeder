# 要件：MCPサーバー利用者（クライアント）単位でのフィード管理

## 背景・目的

`docs/steering/20260726_mcp_server/` で実装した `cmd/mcp` は、Claude Desktop 等の MCP クライアントから rss-feeder の機能（フィード管理・記事取得・AI要約など）を呼び出せるようにしたが、現状はシングルユーザー前提のままである。`internal/domain/feed.go` の `Feed` や `internal/usecase` 配下の各Usecase（`add_feed`・`list_feeds`・`remove_feed`・`list`・`search`・`list_categories`・`bookmark`・`mark_read`・`fetch`・`enrich`・`preference` 等）はユーザーという概念を一切持たず、DB（`rss-feeder-db/reader.db`）には全利用者共通の1セットのフィード・記事しか存在しない。

複数人（あるいは1人が複数の用途）で `cmd/mcp` を使い分けたい場合に、購読フィード・記事・ブックマーク・趣向分析がすべて混ざってしまうため、MCPサーバー利用者（クライアント）単位でフィードを分離管理できるようにしたい。

## 要件

- `cmd/mcp` の起動引数に `--user-id`（文字列。人間が読める識別子、例: `alice`）を追加し、プロセス起動時に固定のユーザーとして動作させる
- 同一の識別子文字列で複数回起動された場合は同一ユーザーとして扱う（初回起動時にレコードが無ければ作成するupsert方式）
- ユーザーごとに以下がすべて分離される: 購読フィード一覧、記事一覧・検索・カテゴリ一覧、ブックマーク、既読/未読状態、フィード取得（fetch）、AIエンリッチ（enrich）、趣向分析（preference）
- DBスキーマに `users` テーブルを新設し、フィードにユーザーとの紐付けを追加する
- 既存の `reader.db` にある全フィード・記事は、新規作成する「デフォルトユーザー」1件に紐付ける形でマイグレーションし、データ損失を発生させない
- `bin/rss-feeder`（CLI）・`bin/web`（Web UI）・`bin/rss-agent` は、今回ユーザー概念に対応させず、デフォルトユーザーとして暗黙的に動作し続ける（後述の制約を参照）

## 制約・前提

- MCPツールの入力スキーマ自体に `user_id` を持たせる必要はない（`--user-id` によりプロセス起動時に固定されるため、ツール呼び出しのたびに指定する情報ではない）
- `--user-id` の値はなりすまし対策のない自己申告文字列として扱う前提で設計する（認証・検証を行わない理由は「スコープ外」参照）
- `cmd/rss-feeder`・`cmd/web`・`cmd/agent`（`bin/rss-agent`）は今回ユーザー概念に対応させない。ただしDBスキーマ上フィードにユーザー紐付けを追加する以上、これら既存バイナリが問題なく動き続けられるよう、固定の「デフォルトユーザー」として暗黙的に動作させる設計が必要
- 既存の `internal/usecase` 呼び出し元（`internal/adapter/handler/cli`・`internal/adapter/handler/web`・`internal/adapter/handler/agent`）のハンドラ層自体はロジック変更を行わない前提とし、usecase層のシグネチャ変更に伴う影響は各エントリポイントの DI 配線（`main.go`）側の変更で吸収する
- Go バージョン・DIコンテナ（`samber/do/v2`）は既存同様

## スコープ外

- 認証・アクセス制御（なりすまし防止、`--user-id` の正当性検証等）
- `bin/rss-feeder`・`bin/web` のユーザー概念対応（複数ユーザーの切り替えUI・ログイン機能等）
- リモートMCPコネクタ対応（`docs/steering/20260726_mcp_server/requirements.md` で既にスコープ外と整理済み）
- ユーザーの削除・名称変更等の管理機能（作成のみをサポートする）
- ユーザー間でのフィード共有・招待等のコラボレーション機能

## 完了条件

- `users` テーブルが新設され、`feeds` テーブルがユーザーに紐付く形でスキーマ変更されている（設計は design.md 参照）
- 既存データが「デフォルトユーザー」1件に紐付くマイグレーション手順が設計されている（`rss-feeder-db/*.sh` と `internal/migration/migration.go` 双方への反映方針を含む）
- `cmd/mcp --user-id <識別子>` で起動すると、該当ユーザーのフィード・記事のみが操作対象になる設計になっている
- `cmd/rss-feeder`・`bin/web`・`bin/rss-agent` が変更前と同一のユーザー体験（デフォルトユーザーとして全フィード・記事を操作できる）を維持する設計になっている
- 上記を実現する `internal/usecase`・`internal/adapter/driver/readerdb` 層の変更方針が design.md に明記されている
