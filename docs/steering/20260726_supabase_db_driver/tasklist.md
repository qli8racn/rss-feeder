# タスクリスト：Supabase（Postgres）DBドライバ対応

## docs/steering

- [x] `docs/steering/20260726_supabase_db_driver/requirements.md` 作成
- [x] `docs/steering/20260726_supabase_db_driver/design.md` 作成

## 依存関係の追加

- [x] `github.com/jackc/pgx/v5`（`pgx/v5/stdlib` 含む）を `go.mod` に追加する

## internal/config

- [x] `Config` に `DB DBConfig`（`Driver`・`Supabase.DSN`）を追加する
- [x] `db.driver` 未設定時は `"sqlite"` として扱われることを確認する（後方互換）
- [x] `internal/config/config.example.yml` に `db` セクションのサンプル（コメント付き、ダミー値）を追記する

## internal/migration

- [x] `internal/migration` に Postgres向けスキーマ初期化処理（`RunPostgres` 等）を追加する（`BIGSERIAL`/`TIMESTAMPTZ`/`BOOLEAN DEFAULT FALSE` 等の方言反映）
- [x] `ADD COLUMN IF NOT EXISTS` を用いた既存カラム追加処理を実装する（SQLite版のようなエラー文字列判定は不要にする）
- [x] 既存の `migration.Run`（SQLite版）に変更が入っていないことを確認する

## internal/driver/readerpg（新規）

- [x] `internal/driver/readerpg/client.go` に `NewClient(i do.Injector) (*sql.DB, error)` を実装する（`cfg.DB.Supabase.DSN` を用いて `sql.Open("pgx", dsn)`）
- [x] `internal/driver/readerpg/feed` に `feedrepo.Repository` のPostgres実装を追加する（`ON CONFLICT ... DO UPDATE`／`RETURNING id` 等への書き換えを含む）
- [x] `internal/driver/readerpg/article` に `articlerepo.Repository` のPostgres実装を追加する（`INSERT ... ON CONFLICT DO NOTHING`、`UpdateMetadataBatch` の名前付きパラメータを位置引数へ書き換え等を含む）
- [x] `internal/driver/readerpg/auditlog` に `auditlogrepo.Repository` のPostgres実装を追加する
- [x] `internal/driver/readerpg/dbmaintenance` に `Maintainer` のPostgres実装を追加する（`Vacuum` は `VACUUM` 実行、`IntegrityCheck` はPostgresでは未サポートである旨を返す方針を実装する）

## DIコンテナの配線（4つの main.go）

- [x] `cmd/rss-feeder/main.go` に `cfg.DB.Driver` に応じたリポジトリ切り替え・`migration.Run`/`RunPostgres` の分岐を実装する
- [x] `cmd/web/main.go` に同様の分岐を実装する
- [x] `cmd/agent/main.go` に同様の分岐を実装する
- [x] `cmd/mcp/main.go` に同様の分岐を実装する

## ビルド確認

- [x] `go build -o bin/rss-feeder ./cmd/rss-feeder` が通ることを確認する
- [x] `go build -o bin/web ./cmd/web` が通ることを確認する
- [x] `GOMAXPROCS=1 GOFLAGS="-gcflags=all=-l=0" go build -p 1 -o bin/rss-agent ./cmd/agent` が通ることを確認する
- [x] `GOMAXPROCS=1 GOFLAGS="-gcflags=all=-l=0" go build -p 1 -o bin/mcp ./cmd/mcp` が通ることを確認する

## テスト

- [x] 既存のSQLite向けユニットテスト（`internal/driver/readerdb/feed`・`article`）が変更なしに通ることを確認する
- [x] `go test $(go list ./... | grep -v internal/driver/anthropic)` が通ることを確認する
- [ ] （手動）`db.driver: supabase` を設定した上で、実際のSupabaseプロジェクトに対して `add-feed`・`fetch`・`bookmark`・`enrich`・`curate`・`discover` 等の一連の操作を実行し、SQLite版と同様に動作することを確認する

## ドキュメント整備

- [x] `AGENTS.md`（または該当ドキュメント）に `db.driver` 設定・Supabase接続時のセットアップ手順を追記する
- [x] `internal/config/config.example.yml` の更新内容が実際の設定手順と一致していることを確認する

## PRレビュー対応

### Must fix

- [x] `internal/migration/postgres.go` の `audit_log.article_id` の外部キーを `ON DELETE SET NULL` に変更し、`CREATE TABLE IF NOT EXISTS` のため既存テーブルには反映されない旨をコメントで残す
- [x] `internal/driver/readerpg/article/article.go` の `Search`・`FindFiltered` の `LIKE` を `ILIKE` に書き換え、`design.md` にも追記する

### Should fix

- [x] `internal/driver/readerpg/client.go` の `NewClient` に DSN 空文字チェック・`PingContext`（タイムアウト付き）による疎通確認・コネクションプール設定（`SetMaxOpenConns`/`SetMaxIdleConns`/`SetConnMaxLifetime`）を追加する
- [x] `internal/config/config.go` の `Load()` で `db.driver` の値を検証し、`""`・`"sqlite"`・`"supabase"` 以外はエラーにする（`DBConfig.Validate`）。`config_test.go` にテストケースを追加する
- [x] `AGENTS.md`・`internal/config/config.example.yml` に Supabase の Session pooler(5432)/Transaction pooler(6543) の違いと `default_query_exec_mode=simple_protocol` の注記、TIMESTAMPTZ/DATETIME の表示差異の注意点を追記する
- [x] `internal/driver/readerpg/{feed,article}` に `//go:build pg_integration` タグ付きの統合テストを追加する（`TEST_POSTGRES_DSN` 未設定時は `t.Skip`。通常の `go build`/`go test`（タグなし）はコンパイル対象外であることを確認済み）

### Nits

- [x] `internal/migration` に `RunFor(cfg *config.Config, db *sql.DB) error` を追加し、4つの `cmd/*/main.go` の migration 呼び分けをこの1関数呼び出しに集約する
- [x] `internal/driver/readerpg/dbmaintenance/dbmaintenance.go` の `Vacuum` で対象テーブル（`articles, feeds, audit_log`）を明示する
- [x] `internal/driver/readerpg/article/article.go` の `published_at DESC` 固定の箇所（`FindAll`・`FindUnread`・`FindBookmarked`・`FetchLatest`・`Search`）に `NULLS LAST` を明示する。`FindFiltered` はソート順（ASC/DESC）が動的なため、SQLiteのデフォルト（ASC→NULLS FIRST、DESC→NULLS LAST）に合わせてorderDirに応じて`NULLS FIRST`/`NULLS LAST`を出し分ける（`category`・`publisher`等NULLになりうる列でのソートに影響するため）
- [x] `internal/driver/readerdb/{article,auditlog,dbmaintenance}` に `var _ XxxRepository = (*repository)(nil)` のインターフェース充足アサーションを追加する（`readerpg` 側と対称にする）
- [x] `IntegrityCheck`（`internal/driver/readerpg/dbmaintenance/dbmaintenance.go`）は現状維持（対応不要）

## PRレビュー対応（2巡目）

### Should fix

- [x] `.github/workflows/ci-go.yml` に `go vet -tags pg_integration ./internal/driver/readerpg/...` のステップを追加し、統合テストが実DBなしで型チェックされるようにする
- [x] `AGENTS.md` の Test セクションに `TEST_POSTGRES_DSN`・`-tags pg_integration`・`-p 1`（article/feedパッケージが同じDBをTRUNCATEするため並列実行不可）を追記する
- [x] `internal/driver/readerpg/article/article.go` の `FindWithoutSummary` に `NULLS LAST` を追加し、他5箇所と揃える
- [x] 4つの `cmd/*/main.go` の `*config.Config`・`*sql.DB` の取得を `do.MustInvoke` から `do.Invoke` + 明示的なエラーハンドリングに変更し、DSN誤り等の接続エラーがpanicではなくエラーメッセージとして表示されるようにする

### Nits

- [x] `internal/driver/readerpg/client.go` の `NewClient` で `PingContext` 失敗時に `db.Close()` を呼ぶ
- [x] `AGENTS.md` に文字列カラムのソート順（SQLite=`BINARY`／Supabase=`en_US.UTF-8`相当）の差異は既知としてPostgres側を正とする旨を追記する
- [x] `internal/driver/readerpg/article/article_pg_integration_test.go` の `INSERT INTO feeds (id, ...)` を明示的なid指定からRETURNING経由の取得に変更し、`feeds_id_seq` を進める
- [x] `AGENTS.md` に `cmd/agent` が本ブランチから起動時マイグレーションを実行するようになった旨（意図的な変更）を追記する
- [x] `AGENTS.md` のTransaction pooler回避策に `default_query_exec_mode=exec` も選択肢として併記する
- [ ] `audit --article-id` に存在しない記事IDを渡した場合のFK違反（`.claude/hooks/audit-log.sh` は `|| true` で握り潰しているため実害は小さい）は対応不要と判断（現状維持）

## 今後のタスク（本フェーズではスコープ外・別フェーズで着手）

- [ ] ローカル環境でSelf-hosted Supabase（`supabase` CLI / docker compose）を使えるようにする
  - クラウド版と同じ `db.driver: supabase` + DSN方式で接続先をローカルのSelf-hosted Supabaseに向けられる想定だが、
    ローカル起動手順（`supabase start` 等）・devcontainer側のDocker-in-Docker要否・リソース消費（コンテナ数）の検討が必要なため、
    別の `docs/steering/` フェーズとして requirements.md/design.md を改めて起こすこと。
