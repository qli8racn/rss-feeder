# 設計：MCPサーバー利用者（クライアント）単位でのフィード管理

## 全体構成

```
cmd/mcp --user-id alice
  └─ chdirToRepoRoot() → migration.Run(db)（users テーブル新設・feeds への user_id 追加・デフォルトユーザーへの既存データ紐付け）
       └─ ResolveUserUsecase.Execute(ctx, "alice")  … users テーブルを find-or-create（upsert）
            └─ 解決済み userID（int64）を各 usecase・driver/anthropic agent の構築時に注入
                 ├─ usecase.NewListUsecase(articleRepo, userID) 等（userID を struct field として保持）
                 ├─ driveranthropic.NewPreferenceAgent(i)（内部で userID を保持し、repo呼び出し時に渡す）
                 └─ ... 既存 driver 層（readerdb 等）はメソッド引数として userID を受け取り WHERE 句で絞り込む

cmd/rss-feeder / cmd/web / cmd/agent（bin/rss-agent）
  └─ ResolveUserUsecase.Execute(ctx, DefaultUserName)  … "default" ユーザーを find-or-create
       └─ 以降は cmd/mcp と同じ配線パターンで、常に同じデフォルトユーザーIDが usecase に渡る
            → ハンドラ層（internal/adapter/handler/cli・web・agent）・利用者体験は無変更
```

`cmd/mcp` はプロセス起動時に1ユーザーに固定される（stdio transport は1接続=1プロセスのため）。`cmd/rss-feeder`・`cmd/web`・`cmd/agent` も同様に「1プロセス=1ユーザー（常に `default`）」という同じ形に揃えることで、usecase層以下は全エントリポイントで同一の設計（コンストラクタにuserIDを渡す）を共有できる。

## DBスキーマ変更

### `users` テーブルの新設

```sql
CREATE TABLE IF NOT EXISTS users (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    name       TEXT     UNIQUE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

- `name` が `--user-id` で指定される識別子文字列（例: `alice`）。UNIQUE制約により同一識別子の重複作成を防ぐ
- 既存のCLI/Web UI/agent向けに `name = "default"` の固定レコードを1件用意する（`DefaultUserName = "default"` を `internal/domain` に定数として定義し、マイグレーション・各 `main.go` の両方から参照して文字列のずれを防ぐ）
- 認証は行わないため、`name` の値自体に対するアクセス制御・検証（記号制限等）は設けない

### `feeds` テーブルへの `user_id` 追加とユニーク制約の変更

```sql
ALTER TABLE feeds ADD COLUMN user_id INTEGER REFERENCES users(id);
```

既存の `feed_url TEXT UNIQUE NOT NULL`（グローバルなユニーク制約）は、複数ユーザーが同一の外部フィードURLをそれぞれ独立に購読できるようにするため、`UNIQUE(user_id, feed_url)` の複合ユニークに変更する。SQLiteは列のユニーク制約を `ALTER TABLE` で直接変更できないため、テーブル再作成（rename→create→copy→drop→rename、いわゆる「12-step」手順）が必要になる。具体的な手順は後述の「マイグレーション設計」を参照。

**article テーブルには `user_id` を追加しない**。理由:

- `feeds` は「1ユーザー1購読=1行」の設計（後述の比較参照）にするため、`articles.feed_id` が指す `feeds` 行にはすでに `user_id` が乗っている。したがって `articles → feeds(user_id)` の JOIN だけで所有ユーザーを一意に判定でき、`articles` に列を重複して持たせる必要がない
- 記事単位の操作（`FindByID`・`Update`・`MarkAsRead` 等）はすべて `JOIN feeds ON articles.feed_id = feeds.id AND feeds.user_id = ?` を条件に加えることで、他ユーザーの記事IDを推測して操作されることを防ぐ（記事IDはユーザー横断の連番のため、この JOIN 条件がないとIDの総当たりで他ユーザーの記事を読める/更新できてしまう）

### 既存FKとの整合性についての補足

既存の `articles.feed_id REFERENCES feeds(id)` は宣言のみで、DSN（`internal/driver/readerdb/client.go`）で `PRAGMA foreign_keys` を有効化していないため実際には強制されていない。今回追加する `feeds.user_id REFERENCES users(id)` も同じ扱い（宣言のみ、アプリケーション側の実装で整合性を担保する）とし、既存の方針を踏襲する。FK強制を有効にするかどうかは今回のスコープ外とする。

### フィード単位の設計比較：ユーザーごとにフィード行を複製する方式 vs. フィード共有＋購読テーブル方式

| 観点 | 案A: `feeds.user_id` を直接追加（1ユーザー1購読=1行、採用） | 案B: フィードカタログを共有し `user_feed_subscriptions` 等の中間テーブルで購読を管理 |
|---|---|---|
| 要件との整合性 | ユーザーヒアリング内容で明示された「`feeds` テーブルへの `user_id` 外部キー追加」に直接合致する | 要件の記載（直接FK追加）から逸脱する |
| 記事データの扱い | ユーザーごとに記事も別行で保持（同じ外部URLを複数ユーザーが購読すると記事が重複保存される） | 記事はフィード単位で共有され重複しない |
| 既読/ブックマーク状態 | `articles` に直接列があるままで良い（1記事=1ユーザー分なので現状のスキーマを変えずに済む） | 既読/ブックマークはユーザーごとに異なるべき状態のため、`articles` から切り出して `user_article_states(user_id, article_id, read, bookmarked)` 等へ移す大掛かりな変更が必要 |
| fetch/enrichの重複コスト | ユーザーごとに同じ外部フィードを個別にfetch・enrich（AI課金）するため、複数ユーザーが同じ人気フィードを購読すると無駄なコストが発生し得る | フィード単位で1回fetch・enrichすれば全購読ユーザーに反映されるため効率的 |
| マイグレーションの規模 | `feeds` テーブルの再作成（ユニーク制約変更）のみで完結 | `articles` の既読/ブックマーク列を分離する再設計が必要で影響範囲が大きい |

**採用**: 案A。ユーザーからの要件定義で「`feeds` テーブルへの `user_id` 外部キー追加」が明示されており、`articles` 側の大規模なスキーマ変更（既読/ブックマークの分離）を避けられる点を優先した。fetch/enrichの重複コストは、現時点では `cmd/mcp` の想定利用者数が少数（個人〜数人規模）であることから許容範囲と判断する。将来的に多数ユーザーでの利用が発生し重複コストが問題化した場合は、案Bへの移行（記事の共有化＋ユーザー状態の分離）を再検討する。

## マイグレーション設計

`rss-feeder-db/*.sh`（シェルスクリプト、手動適用用）と `internal/migration/migration.go`（アプリ起動時の自動適用）の両方に反映する既存パターンを踏襲し、`20260726_add_user_management.sh` を新設する。

### 手順

1. `users` テーブルを作成（`CREATE TABLE IF NOT EXISTS`）
2. `name = "default"` の行が無ければ作成する（`INSERT INTO users(name) VALUES ('default') ON CONFLICT(name) DO NOTHING`）。作成後、その `id` を取得する
3. `feeds` に `user_id` 列を追加する（`ALTER TABLE feeds ADD COLUMN user_id INTEGER REFERENCES users(id)`。既存のカラム追加スクリプト（`20260614_add_article_metadata.sh`・`migration.go` の `addArticleColumns`）と同様、"duplicate column name" エラーは無視して冪等に実行できるようにする）
4. 既存の全 `feeds` 行の `user_id` が `NULL` の場合、手順2で取得した「デフォルトユーザー」の `id` で埋める（`UPDATE feeds SET user_id = ? WHERE user_id IS NULL`）。これにより既存フィード・記事はデータ損失なくデフォルトユーザーに紐付く（`articles` は `feed_id` 経由の間接紐付けのため、`articles` 自体への変更は不要）
5. `feed_url` のグローバルユニーク制約を `UNIQUE(user_id, feed_url)` に変更する（SQLiteのテーブル再作成手順）
   1. `feeds` を `feeds_old` にリネーム
   2. `user_id INTEGER NOT NULL REFERENCES users(id)`・`UNIQUE(user_id, feed_url)` を含む新しい `feeds` を作成
   3. `INSERT INTO feeds SELECT ... FROM feeds_old` で全行をコピー（手順4で全行の `user_id` が非NULLになっている前提のため `NOT NULL` 制約と矛盾しない）
   4. `articles.feed_id` は数値IDをそのまま参照しているため、`feeds` の `id` を維持する形でコピーすれば `articles` 側の変更は不要（`INSERT INTO feeds (id, user_id, feed_url, title, last_fetched, created_at) SELECT id, user_id, feed_url, title, last_fetched, created_at FROM feeds_old` のように明示的に `id` を指定してコピーする）
   5. `feeds_old` を `DROP TABLE`
   6. `CREATE INDEX IF NOT EXISTS idx_feeds_user_id ON feeds(user_id)` を追加（ユーザー単位の一覧取得を高速化するため）
6. 冪等性の確認: 手順5のテーブル再作成は一度実行すると新スキーマになるため、再実行時に「`feeds_old` が既に存在しない」「新スキーマにすでに `UNIQUE(user_id, feed_url)` が付与済み」であることを検知してスキップする（例: `PRAGMA index_list('feeds')` や `sqlite_master` の `sql` 列を見て制約の有無を確認する等、具体的な検知方法は実装時に確定する）

### `internal/migration/migration.go` への反映方針

既存の `Run(db *sql.DB) error` に、上記手順を実行する `addUserManagement(db)` を追加し、`addArticleColumns(db)` の後段で呼び出す。SQLiteのテーブル再作成（手順5）はトランザクション（`db.Begin()` / `tx.Commit()`）内で実行し、途中失敗時に中途半端な状態（`feeds_old` が残ったまま等）にならないようにする。

### `rss-feeder-db/*.sh` への反映方針

`rss-feeder-db/20260726_add_user_management.sh` を新設し、`sqlite3 "$DB"` に対して上記手順を順に実行する。手順2で作成したユーザーIDの取得は `sqlite3 "$DB" "SELECT id FROM users WHERE name='default'"` のようにサブシェルで取得し、後続のUPDATE文に埋め込む（既存の `20260614_add_article_metadata.sh` はカラム追加のみで完結するシンプルな例だが、本スクリプトは複数ステップ・データ取得を伴うため、`set -euo pipefail` を維持しつつステップごとにエラーハンドリングする）。

## `internal/usecase` 層の変更方針

### 各Usecaseの構成方針：userIDをExecute引数ではなくコンストラクタで固定する

`cmd/mcp`・`cmd/rss-feeder`・`cmd/web`・`cmd/agent` はいずれも「1プロセス=1ユーザー（`cmd/mcp` は `--user-id` の値、他は常に `default`）」というライフサイクルであり、同一プロセス内で複数ユーザーを切り替える必要がない。この前提から、以下の2案を比較する。

| 観点 | 案A: userIDをUsecaseの構造体フィールドとして構築時に固定（採用） | 案B: `Execute(ctx, userID, ...)` のように呼び出しごとの引数にする |
|---|---|---|
| 既存ハンドラ層への影響 | `internal/adapter/handler/{cli,web,agent}` の呼び出しコード（`uc.Execute(ctx, ...)`）は無変更で済む。変更が必要なのは各 `main.go` の `usecase.NewXxxUsecase(...)` 呼び出しへの引数追加のみ | ハンドラ層の全呼び出し箇所（`uc.Execute(ctx, ...)`）を含めて修正が必要になり、要件の「既存ハンドラは変更しない」という前提に反する |
| 将来の拡張性 | 同一プロセス内で複数ユーザーを切り替える用途（例: Web UIの将来的なマルチテナント化）には対応できない | 将来的なマルチテナント化に対応しやすい |
| 実装の一貫性 | 既存のUsecaseコンストラクタが既に repo・agent 等の依存を構造体フィールドとして受け取るパターン（例: `NewListUsecase(articleRepo)`）と一貫する | 既存パターンから逸脱し、依存注入と実行時引数の使い分けが分かりにくくなる |

**採用**: 案A。要件で明示された「既存ハンドラ層は変更しない」という制約に合致し、既存のコンストラクタ注入パターンとも一貫するため。将来Web UIが真のマルチテナント対応（HTTPリクエストごとに異なるユーザーを認証・解決する等）を必要とする場合は、案Bへの移行（`Execute` 引数化、あるいは `context.Context` 経由でのuserID伝播）を再検討する（本フェーズのスコープ外として記録する）。

### 影響を受けるUsecase／Repositoryの分類

1. **リポジトリを直接保持するUsecase**（`list`・`search`・`list_categories`・`list_feeds`・`bookmark`・`mark_read`・`add_feed`・`remove_feed`・`fetch`・`backfill_metadata`・`reset`・`check_article`・`check_bookmarked` 等）
   - コンストラクタに `userID int64` を追加し、構造体フィールドとして保持する
   - `Execute` 内部で repo 呼び出し時に `uc.userID` を渡す
2. **`internal/adapter/driver/anthropic` の Agent をラップするだけのUsecase**（`enrich`・`preference`・`curate`・`discover`・`summarize`）
   - Usecase自体は repo を保持していないため変更不要。userIDは委譲先の Agent 実装（`internal/driver/anthropic/*.go`）のコンストラクタに追加する（後述）
3. **リポジトリ層**（`internal/adapter/driver/readerdb/{feed,article}` のインターフェースと `internal/driver/readerdb/{feed,article}` の実装）
   - `feedrepo.Repository`: `Save`・`FindByURL`・`Register`・`ListAll`・`Remove` の各メソッドに `userID int64` 引数を追加し、SQLの `WHERE user_id = ?`（`Register`・`Save` は `INSERT` 時に `user_id` 列へ設定）に反映する
   - `articlerepo.Repository`: `FindAll`・`FindUnread`・`FindBookmarked`・`FindFiltered`・`FindByID`・`Update`・`MarkAsRead`・`DeleteNonBookmarked`・`CountNonBookmarked`・`CountBookmarked` の各メソッドに `userID int64` 引数を追加し、`JOIN feeds ON articles.feed_id = feeds.id AND feeds.user_id = ?` を付与する（`Save` は挿入対象の `feed_id` が既にユーザー固有のフィード行を指すため、引数追加は不要と判断する）
4. **`internal/driver/anthropic` の Agent実装**（`preference.go`・`curate.go`・`discover.go`。`FindBookmarked`・`ListAll` を直接呼び出している）
   - 各 `NewXxxAgent(i do.Injector)` コンストラクタで userID を受け取れるようにし（DIコンテナに `do.ProvideValue` 等で登録した解決済みuserIDを注入する、または `main.go` 側で明示的に構築し直す）、構造体フィールドとして保持したうえで、内部の repo 呼び出しに渡す

### 新設するUsecase・Repository：ユーザー解決（find-or-create）

```go
// internal/domain/user.go
type User struct {
    ID        int64
    Name      string
    CreatedAt time.Time
}

const DefaultUserName = "default"
```

```go
// internal/adapter/driver/readerdb/user/user.go
type Repository interface {
    FindByName(ctx context.Context, name string) (*domain.User, error)
    Create(ctx context.Context, name string) (*domain.User, error)
}
```

```go
// internal/usecase/resolve_user.go
type ResolveUserUsecase struct {
    userRepo userrepo.Repository
}

func (uc *ResolveUserUsecase) Execute(ctx context.Context, name string) (*domain.User, error) {
    // FindByName → 見つかれば返す。見つからなければ Create（UNIQUE制約違反時は再度 FindByName でレース条件を吸収する）
}
```

`ResolveUserUsecase` は他のUsecaseと異なり userID を保持しない（userIDを解決すること自体が役目のため）。全エントリポイントの `main.go` で、DI配線・migration実行の直後・他のUsecase構築より前に1回だけ呼び出す。

## `cmd/mcp` の `--user-id` 実装方針

- `flag.String("user-id", domain.DefaultUserName, "MCPクライアントを識別するユーザーID（例: alice）。省略時は CLI/Web UI と同じ default ユーザーとして動作する")` のように、デフォルト値を `default` とするオプショナルフラグとして追加する（未指定でも起動でき、既存の単一ユーザー運用からの移行を強制しない）
- `chdirToRepoRoot()` → `migration.Run(db)` の直後、他のUsecase・Agentを構築する前に `ResolveUserUsecase.Execute(ctx, *userIDFlag)` を呼び出し、解決済みの `*domain.User` から `userID int64` を取得する
- 取得した `userID` を `usecase.NewListUsecase(articleRepo, userID)` のように各Usecaseコンストラクタへ明示的に渡す（`main.go` 内の `do.MustInvoke` 呼び出し群と並べて書く、既存の配線スタイルを踏襲する）
- `driveranthropic.NewPreferenceAgent` 等、DIコンテナ経由で構築される Agent にも userID を渡す必要があるため、他の `int64` 値との型衝突を避ける専用型 `usecase.UserID`（`type UserID int64`）を定義し、`do.ProvideValue(i, usecase.UserID(userID))` でDIコンテナに登録する。各 `NewXxxAgent(i do.Injector)` 内では `do.MustInvoke[usecase.UserID](i)` として取得する
- MCPツールの入力スキーマ（`internal/adapter/handler/mcp` 配下のツールハンドラ）には `user_id` 引数を追加しない。ツール呼び出し時ではなくプロセス起動時に固定される値のため

## `cmd/rss-feeder`・`cmd/web`・`cmd/agent` の変更方針

いずれも `--user-id` フラグは追加しない。各 `main.go` で `migration.Run(db)` の直後に `ResolveUserUsecase.Execute(ctx, domain.DefaultUserName)` を呼び出し、常に `default` ユーザーのIDを解決したうえで、`cmd/mcp` と同じ配線パターン（Usecase・Agentのコンストラクタに解決済みuserIDを渡す）で組み立てる。

`internal/adapter/handler/cli`・`internal/adapter/handler/web`・`internal/adapter/handler/agent` 配下のハンドラ実装自体（`uc.Execute(ctx, ...)` の呼び出しコード）は変更しない。変更が生じるのは各 `main.go` のDI配線コードのみであり、CLI/Web UIの外部から見た挙動（コマンド引数・APIレスポンス）は一切変わらない。

## ログ・既存設計との整合性

- `cmd/mcp` のログ設計（`docs/steering/20260726_mcp_server/design.md` の「ログ設計」節）・CWD解決（同design.mdの「動作確認で発覚した不具合の修正」節）は本フェーズの変更対象外であり、そのまま維持する
- `internal/driver/readerdb/client.go` のDSN（WALモード・busy_timeout）は変更不要。ユーザー単位のスコープ追加はSQLクエリのWHERE句・JOIN句の変更のみで実現し、DBファイル自体は引き続き `rss-feeder-db/reader.db` を全エントリポイントで共有する

## 未解決・将来検討事項

- フィード単位の設計比較で述べた「同一外部フィードを複数ユーザーが購読した場合のfetch/enrich重複コスト」は、利用者数が増えた場合に再検討する
- `users` テーブルにはユーザーの削除・改名機能を設けない（要件のスコープ外）。誤った識別子で `cmd/mcp --user-id` を起動してしまった場合の復旧手段（該当ユーザーの `feeds.user_id` を手動で付け替える等）は今回設計しない
- FK制約（`PRAGMA foreign_keys`）を有効化するかどうかは既存踏襲で見送ったが、データ整合性を厳格化したい場合は別途検討する
