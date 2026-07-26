# 要件：Supabase（Postgres）DBドライバ対応

## 背景・目的

現在 rss-feeder は DB として SQLite（`internal/driver/readerdb` 配下、`feed`・`article`・`auditlog` の3リポジトリ、`database/sql` + `github.com/mattn/go-sqlite3` を直接使用）のみをサポートしている。

ユーザー（プロジェクトオーナー）は学習目的でクラウド上の Supabase（Postgres）を試したいが、SQLite の選択肢は完全になくさず、設定でどちらを使うか選べるようにしたい。

## 要件

- `internal/config/config.yml` に DB ドライバ選択の設定項目（`db.driver: sqlite | supabase`）を追加する。未設定の場合は `sqlite` として扱い、現状通りの挙動（後方互換）とする。
- `db.driver: supabase` を指定した場合、Supabase（supabase.com が提供するクラウド上の Postgres プロジェクト）へ接続し、`feed`・`article`・`auditlog` の全3リポジトリが SQLite 版と同等の操作（保存・検索・更新・削除等）を提供する。
- Supabase への接続情報（Postgres接続文字列、または host/port/user/password/dbname・sslmode 等）は `config.yml` に設定する形とする。
- `cmd/rss-feeder`・`cmd/web`・`cmd/agent`・`cmd/mcp` の4エントリポイントすべてが、`config.yml` の設定のみでSQLite/Supabaseを切り替えて動作する。

## 制約・前提

- Supabase の形態はクラウド上のSupabaseプロジェクト（supabase.com、無料プラン）に接続する方式のみを対象とし、Self-hosted（Docker）は採用しない。
- ユーザー自身が Supabase 側でプロジェクトを作成し、接続情報を `config.yml` に設定する前提とする（Claude Code がクラウドアカウントを操作することはない）。
- `internal/config/config.yml` は既に Git 管理外（`.gitignore` 対象）であり、worktree・環境ごとに個別のファイルを配置する運用が `AGENTS.md` に明記済みである。そのため「ローカルマシンでは SQLite、共有の開発環境では Supabase」といった環境ごとの切り替えは、環境ごとに内容の異なる `config.yml` を配置するだけで実現できる。コード側で実行環境を自動判定するような特別な分岐は設けない。
- CI 環境からは Supabase（実クラウド）への接続ができない可能性が高い。CI・自動テストの扱いは `design.md` で検討する。

## 完了条件

- `db.driver` 未設定 or `sqlite` の場合、既存の挙動（SQLite への読み書き）に変化がないことを確認する（リグレッションがないこと）。既存のSQLite向けユニットテスト（`internal/driver/readerdb/feed`・`article`）が変更なしに通ること。
- `db.driver: supabase` を設定した状態で、ユーザーが手元のSupabaseプロジェクトに対して `fetch`・`enrich`・`curate`・`bookmark`・`preference`・`discover` 等の一連の操作を実行し、feed・article・auditlogの読み書きがSQLite版と同様に行えることを確認する（手動確認）。
- 4つのエントリポイント（`cmd/rss-feeder`・`cmd/web`・`cmd/agent`・`cmd/mcp`）すべてが `go build` に成功し、`db.driver` の設定切り替えのみで動作先DBを変更できる。
- `config.yml` のスキーマ変更内容が `internal/config/config.example.yml` に反映されている。

## スコープ外

- Self-hosted Supabase（Docker版）への対応（将来タスクとして `tasklist.md` に記載。今回のフェーズでは対応しない）
- 実行環境（CI・ローカル・共有開発環境等）をコードが自動判定してDBを切り替える機能
- SQLite・Supabase間のデータ移行（既存SQLiteデータをSupabaseへコピーするツール等）
- Supabase固有機能（Auth・Storage・Realtime・Row Level Security等）の活用
- 複数のSupabaseプロジェクト・複数DBへの同時接続
