# 設計：Supabase（Postgres）DBドライバ対応

## 現状構造の調査結果

- `internal/adapter/driver/readerdb/{feed,article,auditlog}` に **リポジトリインターフェース**（`Repository`）と、ドメイン非依存のエラー型（`ErrAlreadyExists`・`ErrNotFound`・`ErrDuplicate`等）が定義済み。
- `internal/driver/readerdb/{feed,article,auditlog}` が上記インターフェースの **SQLite実装**。いずれも `*sql.DB`（`database/sql`）を `do.Injector` から受け取るコンストラクタ（`NewRepository(i do.Injector)`）を持つ。
- `internal/driver/readerdb/client.go` の `NewClient(_ do.Injector) (*sql.DB, error)` が `sql.Open("sqlite3", dsn)` でSQLite接続を作る唯一の箇所（DSNはWALモード・busy_timeout付き）。`NewInMemoryDB()` はテスト・インメモリ用。
- `internal/driver/readerdb/dbmaintenance` はSQLite固有の `VACUUM`・`PRAGMA integrity_check` を実行する。
- `internal/migration/migration.go` の `Run(db *sql.DB) error` が唯一のスキーマ初期化処理。`CREATE TABLE IF NOT EXISTS`（`AUTOINCREMENT`・`DATETIME`・`BOOLEAN DEFAULT 0`等のSQLite方言）を一括実行した後、`addArticleColumns` で `ALTER TABLE ... ADD COLUMN` を1文ずつ実行し、SQLiteが `ADD COLUMN IF NOT EXISTS` をサポートしないため `duplicate column name` エラーを無視する、という「起動時に毎回実行する簡易マイグレーション」方式。専用のマイグレーションツール（golang-migrate等）は使用していない。
- SQL文はプレースホルダ `?`（位置引数）で統一されている。ただし `internal/driver/readerdb/article/article.go` の `UpdateMetadataBatch` のみ `sql.Named`（SQLiteドライバの名前付きパラメータ `:name` 記法）を使用している。
- DIコンテナ（`samber/do/v2`）の配線は `cmd/rss-feeder`・`cmd/web`・`cmd/agent`・`cmd/mcp` の4つの `main.go` それぞれで個別に行っている（共通の `internal/di` パッケージは存在しない）。いずれも `do.Provide(i, readerdb.NewClient)` → 各リポジトリの `do.Provide` → `migration.Run(do.MustInvoke[*sql.DB](i))` という同一パターン。
- `internal/config/config.go` の `Config` は現状 `AnthropicAPIKey`・`Log` のみ。`viper` で `internal/config/config.yml` を読み込む。DB関連の設定項目はまだ存在しない。
- 既存テスト（`internal/driver/readerdb/feed/feed_test.go`・`article/article_test.go`）は `sql.Open("sqlite3", ":memory:")` + `migration.Run(db)` を直接呼び出しており、SQLite決め打ちである。

## 全体方針

既にリポジトリ層がインターフェース化されている（`internal/adapter/driver/readerdb/{feed,article,auditlog}.Repository`）ため、**SQL方言を吸収する共通レイヤーを新設するのではなく、Postgres向けの実装をSQLite向けの実装と完全に分離した「別実装」として追加し、DIコンテナで注入先を切り替える**方針を採る。

```
Claude Desktop / CLI / Web UI
        │
   internal/usecase（変更なし）
        │
   internal/adapter/driver/readerdb/{feed,article,auditlog}.Repository（既存インターフェース、変更なし）
        │
        ├─ internal/driver/readerdb/{feed,article,auditlog}   … SQLite実装（既存、変更なし）
        └─ internal/driver/readerpg/{feed,article,auditlog}   … Postgres実装（新規追加）
                │
        internal/config.Config.DB.Driver の値に応じて
        cmd/*/main.go が上記どちらの NewRepository を do.Provide するか分岐する
```

### この方針を採る理由（比較検討）

| 方式 | 概要 | 長所 | 短所 |
|---|---|---|---|
| **A: 実装を完全分離（採用）** | SQLite実装はそのまま維持し、Postgres専用の実装パッケージ（`internal/driver/readerpg/...`）を新規追加。DIで注入を切り替える | 既存コード（SQLite実装・既存テスト）に一切手を入れない。SQL文はそれぞれの方言で素直に書けるため学習用途として理解しやすい。実装の切り分けが明確でレビューしやすい | SQL文・スキャン処理を2重に保守する必要がある（機能追加のたびに両実装へ反映が必要） |
| B: `sqlx` の `Rebind` でプレースホルダのみ吸収 | SQL文字列は `?` のまま書き、`sqlx.DB.Rebind()` で接続時のドライバに応じて `$1` 形式等へ変換 | プレースホルダの違いだけなら書き換え不要 | `sql.Named`（`UpdateMetadataBatch`）等の名前付きパラメータはドライバ間で互換性の保証がなく別途対応が必要。型差異（`BOOLEAN`/`DATETIME`等）は別途吸収コードが要る。依存追加（`jmoiron/sqlx`）の割に恩恵が限定的 |
| C: SQLビルダー導入（`Masterminds/squirrel`等） | SQL文をビルダーAPI経由で組み立て、対象方言に応じてプレースホルダ形式を出し分ける | 一定の抽象化ができる | 既存の生SQLをすべて書き直すコストが大きく、学習目的の小規模プロジェクトには過剰 |
| D: ORM導入（`gorm`等） | ORMがSQL生成・方言差異を吸収 | 移植コストは小さくなりうる | 既存の生SQL・素朴な `database/sql` ベースの設計方針から大きく逸脱し、既存コードの大部分の書き直しが必要。学習目的（SQLite/Postgresの差異を理解する）にも合わない |

方式Aは保守コストの重複というデメリットはあるものの、対象がリポジトリ3つ・SQL文の総量も限定的であること、既存のSQLite資産・テストに影響を与えないこと、Postgres特有の書き方を素直に学べることから採用する。

## リポジトリ層の実装方針

- 新規パッケージ `internal/driver/readerpg/{feed,article,auditlog}` を追加し、それぞれ既存と同じ `adapter` 側インターフェース（`feedrepo.Repository`・`articlerepo.Repository`・`auditlogrepo.Repository`）を満たす構造体を実装する。
- コンストラクタは `NewRepository(i do.Injector) (xxxrepo.Repository, error)` とし、`do.MustInvoke[*sql.DB](i)` で接続を受け取る点はSQLite実装と同一にする（`*sql.DB` という型自体は `database/sql` 標準の抽象なので、Postgresドライバを `database/sql` 経由（`database/sql` 互換ドライバ）で使う限り、リポジトリのシグネチャ自体に変更は不要）。
- SQL文の主な書き換えポイント：
  - プレースホルダ: `?` → `$1, $2, ...`
  - `INSERT ... ON CONFLICT(feed_url) DO UPDATE SET ...` はPostgresも同一構文をサポート（`ON CONFLICT` はPostgres由来の構文であり、そのまま利用可能）。
  - `INSERT OR IGNORE` → Postgresでは `INSERT ... ON CONFLICT (url) DO NOTHING` に書き換える。
  - `res.LastInsertId()`（`feed.Save`）→ Postgresの `database/sql` 経由ドライバ（後述のpgx含む）は `LastInsertId` を一般にサポートしないため、`INSERT ... RETURNING id` + `QueryRowContext(...).Scan(&id)` に書き換える。
  - `sql.Named`（`UpdateMetadataBatch`）→ 名前付きパラメータの互換性が不確実なため、位置引数（`$1, $2, $3`）ベースの素直な書き方に書き換える。
  - `BOOLEAN`・`DATETIME` 型のGo側スキャン（`sql.NullTime`・`bool`）は `database/sql` の標準型で吸収されるため、リポジトリのScanコード自体は大きな変更なしで動作する見込み（Postgresの `TIMESTAMPTZ`/`BOOLEAN` は `sql.NullTime`/`bool` に問題なくマッピングされる）。
- `internal/driver/readerpg/dbmaintenance` も追加する。ただしSQLite固有のPRAGMAはPostgresに存在しないため、以下のように役割を読み替える。
  - `Vacuum`: Postgresの `VACUUM` をそのまま実行（`VACUUM ANALYZE` にするかは実装時に検討）。
  - `IntegrityCheck`: Postgresには `PRAGMA integrity_check` に相当する組み込み機能がない。フェーズ1では「Postgres実装では未サポートである」旨を返す（空配列 + 何もしない、またはエラーではなく `[]string{"postgres: integrity check is not supported"}` のような説明的な戻り値とする）方針とし、必要であれば将来的に `pg_catalog` を用いた代替チェックを検討する。

## Postgres接続クライアントの追加（`internal/driver/readerpg/client.go` 想定）

- `NewClient(i do.Injector) (*sql.DB, error)` を新設し、`config.Config` から接続情報（DSN）を取得して `sql.Open("pgx", dsn)`（後述）で接続を作る。

### Postgresドライバの選定

| 観点 | `github.com/jackc/pgx/v5`（+ `pgx/v5/stdlib`） | `github.com/lib/pq` |
|---|---|---|
| メンテナンス状況 | 活発にメンテナンスされている | 公式に「メンテナンスモード（新機能開発なし、重大バグ修正のみ）」とアナウンス済み |
| `database/sql` との統合 | `pgx/v5/stdlib` パッケージで `database/sql` 互換ドライバとして登録可能（`sql.Open("pgx", dsn)`） | `database/sql` ドライバとしてネイティブ対応 |
| 機能 | Postgres固有機能（COPY・LISTEN/NOTIFY等）への対応が広い。将来的にpgx独自APIへの移行も可能 | 基本的なCRUD用途には十分 |
| SSL/TLS（Supabase接続で必須） | `sslmode` を含む接続文字列をサポート | 同様にサポート |

**採用**: `github.com/jackc/pgx/v5`（`pgx/v5/stdlib` 経由で `database/sql` ドライバとして利用）を採用する。メンテナンス状況・将来性を重視し、`lib/pq` は見送る。既存コードは `database/sql` ベースのままとし、`pgx` 独自APIへの全面移行はスコープ外とする（`*sql.DB` 型のシグネチャを維持することで、リポジトリ層の変更を最小化する）。

## DIコンテナでの切り替え方法

現状は4つの `main.go` それぞれで `do.Provide(i, readerdb.NewClient)` → 各SQLiteリポジトリの `do.Provide` を固定で行っているため、この部分をconfig値に応じた分岐に変更する。

```go
i := do.New()
do.Provide(i, config.NewProvider)
cfg := do.MustInvoke[*config.Config](i)

switch cfg.DB.Driver {
case "supabase":
    do.Provide(i, readerpg.NewClient)
    do.Provide(i, pgrepoarticle.NewRepository)
    do.Provide(i, pgrepofeed.NewRepository)
    do.Provide(i, pgrepoauditlog.NewRepository)
    do.Provide(i, pgrepodbmaint.NewMaintainer)
default: // "" または "sqlite"
    do.Provide(i, readerdb.NewClient)
    do.Provide(i, dbrepoarticle.NewRepository)
    do.Provide(i, dbrepofeed.NewRepository)
    do.Provide(i, dbrepoauditlog.NewRepository)
    do.Provide(i, dbrepodbmaint.NewMaintainer)
}

db := do.MustInvoke[*sql.DB](i)
if cfg.DB.Driver == "supabase" {
    if err := migration.RunPostgres(db); err != nil { ... }
} else {
    if err := migration.Run(db); err != nil { ... }
}
```

- この分岐ロジック自体は4つの `main.go`（`cmd/rss-feeder`・`cmd/web`・`cmd/agent`・`cmd/mcp`）すべてに重複して書く必要がある（既存も各 `main.go` に配線が重複しており、共通の `internal/di` パッケージは現状存在しないため、今回の変更でもその方針を踏襲する。将来的に配線の共通化を行う場合は別フェーズで検討する）。
- `do.MustInvoke[articlerepo.Repository](i)` のように **呼び出し側（usecase組み立て部分）はインターフェース型で取得**しているため、上記のリポジトリ登録の分岐以外は一切変更不要。

## スキーマ管理・マイグレーション方針

- 現状の「起動時に `CREATE TABLE IF NOT EXISTS` を実行する簡易マイグレーション」方式は踏襲し、専用マイグレーションツール（golang-migrate等）の導入はスコープ外とする（既存方針を尊重し、変更範囲を抑える）。
- `internal/migration/migration.go` の `Run` はSQLite専用として現状のまま維持し、新たに `RunPostgres(db *sql.DB) error` を同一パッケージ（または `internal/migration/postgres.go`）に追加する。Postgres向けスキーマは以下の方言差分を反映する。
  - `INTEGER PRIMARY KEY AUTOINCREMENT` → `BIGSERIAL PRIMARY KEY`（または `GENERATED ALWAYS AS IDENTITY`）
  - `DATETIME` → `TIMESTAMPTZ`
  - `BOOLEAN DEFAULT 0` → `BOOLEAN DEFAULT FALSE`
  - `ALTER TABLE ... ADD COLUMN` については、Postgres 9.6+ は `ADD COLUMN IF NOT EXISTS` をネイティブサポートするため、SQLite版のような「エラーメッセージ文字列を見て無視する」実装は不要になり、`addArticleColumns` 相当の処理はシンプルになる見込み。
- SQLite版・Postgres版のスキーマ定義は将来的に乖離しうるため、両者のカラム構成に差分が出た場合は必ず両方に反映することをレビュー観点として明記する（自動で同期する仕組みは設けない）。

## config.yml のスキーマ変更

`internal/config/config.go` の `Config` 構造体に `DB` フィールドを追加する。

```go
type DBConfig struct {
	Driver   string         `mapstructure:"driver"`   // "sqlite"（デフォルト） | "supabase"
	Supabase SupabaseConfig `mapstructure:"supabase"`
}

type SupabaseConfig struct {
	DSN string `mapstructure:"dsn"`
}
```

`config.yml` の記述例（Supabaseのプロジェクト設定画面に表示される接続文字列をそのまま貼り付けられる形を採用する）:

```yaml
anthropic_api_key: "sk-ant-xxxxxxxx"

db:
  driver: supabase   # 省略時・"sqlite" の場合は従来通り SQLite を使用
  supabase:
    dsn: "postgres://postgres.xxxxxxxx:xxxxxxxx@aws-0-xxxxx.pooler.supabase.com:5432/postgres?sslmode=require"
```

### 接続情報の持ち方の検討

| 方式 | 概要 | 長所 | 短所 |
|---|---|---|---|
| **DSN文字列一本化（採用）** | Supabaseの管理画面に表示される接続文字列をそのまま `dsn` に貼り付ける | Supabase側の表記をコピペするだけで設定が完了し、ユーザーの手間が最小 | host/port/user等を個別に上書きしたい場合に不便（今回のスコープでは不要と判断） |
| host/port/user/password/dbname/sslmode を個別項目化 | 各項目を個別のYAMLキーとして持つ | 項目単位でのバリデーション・環境変数オーバーライドがしやすい | Supabaseの接続文字列から手動で分解して転記する必要があり手間が増える |

学習目的でユーザー自身がSupabaseの接続文字列をそのまま使う想定のため、DSN文字列一本化を採用する。

`db.driver` が空文字または `"sqlite"` の場合は、`SupabaseConfig` の値を一切参照せず、既存の `dsn`（`rss-feeder-db/reader.db?...`）を使う従来の挙動を維持する。

`internal/config/config.example.yml` にも上記 `db` セクションのコメント付きサンプルを追記する（`anthropic_api_key` 同様、実際の値はダミー）。

## テスト方針

- 既存の `internal/driver/readerdb/{feed,article}` のユニットテストはSQLite実装（`sql.Open("sqlite3", ":memory:")` + `migration.Run`）に対するものであり、変更しない。SQLite側のリグレッション防止として維持する。
- Postgres実装（`internal/driver/readerpg/...`）に対する自動テストは、以下の理由から**フェーズ1では追加しない**方針とする。
  - CI環境（GitHub Actions等）からクラウド上のSupabaseプロジェクトへ接続するには秘密情報（DSN）をCIシークレットとして管理する必要があり、学習目的の個人プロジェクトの運用コストに見合わない。
  - ローカルDocker等でPostgresコンテナを都度起動してテストする方式（testcontainers-go等）も選択肢としてはあるが、依存追加・CI実行時間増加のコストに対して、今回のスコープ（個人の学習目的でのSupabase接続確認）では過剰と判断する。
- 代わりに、Postgres実装の妥当性は以下で担保する。
  - `go build`（対象パッケージ明示、`internal/driver/anthropic` を含む場合のOOM回避のビルド作法は `AGENTS.md` 準拠）でコンパイルが通ることをCI・手元で確認する。
  - `db.driver: supabase` を設定した状態で、ユーザーが手元で実際にSupabaseプロジェクトに接続し、`fetch`・`bookmark`・`enrich`・`curate` 等の一連の操作を手動で実行して動作確認する（`requirements.md` の完了条件に対応）。
- 将来的にPostgres実装の自動テストが必要になった場合は、`testcontainers-go` 等を用いたローカルPostgresコンテナでのテスト整備を別フェーズの検討課題とする。

## タスクリスト構成の方針

上記の設計内容を踏まえ、`tasklist.md` は以下の単位でタスク化する。

1. `internal/config` の変更（`DB`/`Supabase` 設定追加、`config.example.yml` 更新）
2. `internal/migration` へのPostgres向けスキーマ追加
3. `internal/driver/readerpg`（client・feed・article・auditlog・dbmaintenance）の新規実装
4. 4つの `main.go` へのDI分岐追加
5. ドキュメント整備（`AGENTS.md`・`README`相当への追記）
6. 手動動作確認（ユーザー実施）
