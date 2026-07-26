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

## 今後のタスク（本フェーズではスコープ外・別フェーズで着手）

- [ ] ローカル環境でSelf-hosted Supabase（`supabase` CLI / docker compose）を使えるようにする
  - クラウド版と同じ `db.driver: supabase` + DSN方式で接続先をローカルのSelf-hosted Supabaseに向けられる想定だが、
    ローカル起動手順（`supabase start` 等）・devcontainer側のDocker-in-Docker要否・リソース消費（コンテナ数）の検討が必要なため、
    別の `docs/steering/` フェーズとして requirements.md/design.md を改めて起こすこと。
