# 設計書（機能設計 + 技術仕様）

## コマンド・API 一覧

### rss-feeder（CLI）

```
rss-feeder <command> [flags]
```

| コマンド | 概要 | フェーズ |
|--------|------|---------|
| `add-feed <url>` | RSS フィード URL を DB に登録 | 8 |
| `list-feeds` | 登録済みフィード一覧を表示 | 8 |
| `remove-feed <id>` | フィードを削除（関連記事も連動削除） | 8 |
| `fetch` | 登録済みフィードを DB から読み込み、記事を取得して DB に保存 | 2・3 |
| `list` | DB 保存済みの記事を一覧表示 | 4 |
| `bookmark <id>` | 記事のお気に入りをトグル（登録/解除） | 5 |
| `reset` | お気に入り以外の記事を削除 | 6 |
| `search <keyword>` | キーワードで記事を全文検索 | 7 |

各コマンドの詳細仕様は `docs/steering/` 以下の各フェーズディレクトリを参照。

### cmd/web（Web API）

```bash
go build -o bin/web ./cmd/web
./bin/web [--port <port>] [--static-dir <dir>]
```

| メソッド・パス | 概要 | フェーズ |
|--------------|------|---------|
| `GET /api/articles` | 記事一覧（`mode`/`category`/`sort`/`order`/`page`/`per_page` クエリ対応） | 9 |
| `GET /api/articles/search` | キーワード全文検索（`q`/`bookmarked`/`category`/`sort`/`order`/`page`/`per_page` クエリ対応） | 9 |
| `POST /api/articles/{id}/bookmark` | 記事のお気に入りをトグル（`audit_log` 記録含む） | 9 |
| `GET /api/categories` | 設定済みカテゴリの一覧（DISTINCT） | 9 |
| `POST /api/articles/fetch` | 登録済み全フィードを取得して DB に保存（CLI の `fetch` と同じ `FetchUsecase.ExecuteAll` を呼び出す） | 9 |
| `/*`（メソッド不問） | `--static-dir` で指定したディレクトリ配下の静的ファイル配信（フロントエンドビルド成果物。`r.Handle` で登録） | 9 |

詳細仕様は `docs/steering/20260614_web_view/` を参照（`POST /api/articles/fetch` はフェーズ完了後に追加されたため当該ドキュメントには未記載）。
ハンドラの実体は `internal/adapter/handler/web/`（`ListArticlesHandler` 等）で、ルート登録自体は `cmd/web/main.go` が行う。

フロントエンド（`web/frontend/`、Vite + React + TypeScript）は `npm run build` で `web/static/` に出力する。

#### API 仕様（OpenAPI）とコード生成

`cmd/web` の JSON API 仕様は `docs/openapi.yaml`（OpenAPI 3.0）で一元管理する。エンドポイント追加・変更時は
まず `docs/openapi.yaml` を更新し、以下のコマンドで Go・フロントエンド双方の型を再生成する。

```bash
go generate ./internal/adapter/handler/web/openapi/...   # internal/adapter/handler/web/openapi/types.gen.go を再生成
cd web/frontend && npm run generate:api                  # src/api/schema.gen.ts を再生成
```

- Go 側は `oapi-codegen`（`go tool oapi-codegen`、`go.mod` の `tool` ディレクティブで管理）を使い、
  `internal/adapter/handler/web/openapi/config.yaml` の設定で型（`models: true`）のみを生成する。
  ルーティング・ハンドラ実装（chi）は既存のまま変更しない。生成された `openapi.Article` / `openapi.PagedArticles` /
  `openapi.FetchResult` / `openapi.Error` を `internal/adapter/handler/web/` 配下のハンドラが DTO として利用する。
- フロントエンド側は `openapi-typescript` を使い、`web/frontend/src/api/schema.gen.ts` に型定義のみを生成する。
  既存の `web/frontend/src/api.ts`（fetch ラッパー）・`web/frontend/src/types.ts`（手書き型）の実装は変更しない。
- 生成ファイル（`*.gen.go` / `*.gen.ts`）はリポジトリにコミットし、コード生成ツールが無い環境でも
  `go build` / `npm run build` がそのまま通る状態を維持する。
- 詳細は `docs/steering/20260617_openapi_codegen/` を参照。

### rss-agent（CLI）

```
rss-agent <command> [flags]
```

| コマンド | フラグ | 概要 |
|------------|----------------------------------------|------------------------|
| `summarize` | `--feed <url>`, `--limit <n>`（デフォルト 10） | 最新記事を AI で要約 |
| `preference` | — | ブックマーク済み記事から趣向を分析 |
| `enrich` | `--limit <n>`（デフォルト 10）, `--force` | 記事に要約・カテゴリを付与してDBに保存 |

`ANTHROPIC_API_KEY` が必要（`internal/config/config.yml` の `anthropic_api_key` または環境変数）。
`cmd/agent/main.go` がエントリポイントで、起動時に `internal/config.Load()` を呼び、
`config.yml` に値があれば `ANTHROPIC_API_KEY` 環境変数にセットする。詳細は
`docs/steering/20260615_config_apikey/` を参照。

> `ANTHROPIC_API_KEY` は claude.ai（コンシューマー向けチャット）のログイン情報ではなく、
> Claude Console（https://console.anthropic.com）の「API Keys」から発行する API キーを使用する。

各エージェントのモデル選定方針：単純な要約・分類タスク（`summarize`/`enrich`）は
`claude-haiku-4-5`、ブックマークの傾向分析という難易度の高いタスク（`preference`）は
`claude-opus-4-8` + adaptive thinking を使用する。コスト・実行時間のチューニング方針・
詳細は `docs/steering/20260617_agent_cost_tuning/` を参照。

#### ビルド

CGO（mattn/go-sqlite3）と Anthropic SDK を同時にリンクするためメモリ使用量が大きい。
OOM が発生する場合はパッケージ並列数を制限する：

```bash
GOMAXPROCS=1 GOFLAGS="-gcflags=all=-l=0" go build -p 1 -o bin/rss-agent ./cmd/agent
```

---

## パッケージ構成（技術仕様）

### 層の依存方向

```
adapter(handler/cli, handler/web, handler/agent) → usecase → domain
driver                                            → adapter(interface)
```

`usecase` と `domain` は外側の実装（SQL・HTTP）を知らない。`driver` が `adapter` のインターフェースを実装することで依存を逆転させる。

### ディレクトリ構成

```
rss-feeder/
├── cmd/
│   ├── rss-feeder/
│   │   └── main.go                              # Composition Root・samber/do コンテナ構築・サブコマンド登録
│   ├── web/
│   │   └── main.go                              # Composition Root・samber/do コンテナ構築・chi router 構築（ルート定義）
│   └── agent/
│       └── main.go                              # Composition Root・samber/do コンテナ構築・サブコマンド登録（rss-agent）
├── internal/
│   ├── domain/
│   │   ├── article.go                           # Article エンティティ・ToggleBookmark() など
│   │   ├── audit_log.go                         # AuditLog エンティティ
│   │   └── feed.go                              # Feed エンティティ
│   ├── config/
│   │   ├── config.go                            # config.yml 読み込み（viper, ANTHROPIC_API_KEY 等）
│   │   ├── config.example.yml                   # config.yml のテンプレート
│   │   └── config.yml                           # API キー等のローカル設定（gitignore）
│   ├── migration/
│   │   └── migration.go                         # DB スキーマ作成（Run(*sql.DB) error）
│   ├── usecase/
│   │   ├── fetch.go                             # 取得・重複チェック・保存の orchestration
│   │   ├── list.go                              # 記事一覧取得ロジック
│   │   ├── bookmark.go                          # お気に入りトグルロジック
│   │   ├── reset.go                             # 非お気に入り記事削除ロジック
│   │   ├── search.go                            # キーワード全文検索ロジック
│   │   ├── add_feed.go                          # フィード登録ロジック
│   │   ├── list_feeds.go                        # フィード一覧取得ロジック
│   │   ├── remove_feed.go                       # フィード削除ロジック
│   │   ├── audit.go                             # audit_log 記録ロジック
│   │   ├── check_article.go                     # 記事 ID 存在確認ロジック
│   │   ├── check_bookmarked.go                  # お気に入り件数確認ロジック
│   │   ├── maintenance.go                       # DB VACUUM・整合性チェック
│   │   ├── summarize.go                         # 記事要約（SummarizeAgent への薄いラッパー）
│   │   ├── preference.go                        # 趣向分析（PreferenceAgent への薄いラッパー）
│   │   └── enrich.go                            # 要約・カテゴリ付与（EnrichAgent への薄いラッパー）
│   ├── adapter/
│   │   ├── driver/
│   │   │   ├── readerdb/
│   │   │   │   ├── article/
│   │   │   │   │   └── article.go              # ArticleRepository interface
│   │   │   │   ├── auditlog/
│   │   │   │   │   └── auditlog.go             # AuditLogRepository interface
│   │   │   │   ├── dbmaintenance/
│   │   │   │   │   └── dbmaintenance.go        # DBMaintenance interface
│   │   │   │   └── feed/
│   │   │   │       └── feed.go                 # FeedRepository interface（ErrAlreadyExists・ErrNotFound 定義）
│   │   │   ├── rss/
│   │   │   │   └── rss_reader.go               # RSSReader interface
│   │   │   └── anthropic/
│   │   │       ├── summarize.go                # SummarizeAgent interface・SummarizeOptions
│   │   │       ├── preference.go                # PreferenceAgent interface
│   │   │       └── enrich.go                    # EnrichAgent interface・EnrichOptions
│   │   └── handler/
│   │       ├── cli/                             # rss-feeder（cobra）向けハンドラ
│   │       │   ├── fetch.go                     # cobra コマンド → ListFeedsUsecase + FetchUsecase 呼び出し
│   │       │   ├── list.go
│   │       │   ├── bookmark.go
│   │       │   ├── reset.go
│   │       │   ├── search.go                    # search サブコマンド（--bookmarked フラグ）
│   │       │   ├── add_feed.go                  # add-feed サブコマンド
│   │       │   ├── list_feeds.go                # list-feeds サブコマンド（msgNoFeeds 定数定義）
│   │       │   ├── remove_feed.go                # remove-feed サブコマンド
│   │       │   ├── table.go                      # 記事一覧テーブル描画ヘルパー（printArticleTable）
│   │       │   ├── audit.go                      # audit サブコマンド（Hook 経由で呼び出し）
│   │       │   ├── check_article.go              # check-article サブコマンド（Hook 経由で呼び出し）
│   │       │   ├── check_bookmarked.go          # check-bookmarked サブコマンド（Hook 経由で呼び出し）
│   │       │   └── maintenance.go                # maintenance サブコマンド（Hook 経由で呼び出し）
│   │       ├── web/                             # cmd/web（HTTP JSON API）向けハンドラ（ルート定義自体は cmd/web/main.go が持つ）
│   │       │   ├── response.go                   # writeJSON・writeJSONError 共通ヘルパー
│   │       │   ├── article.go                    # ListArticlesHandler・SearchArticlesHandler・BookmarkArticleHandler（DTO は openapi パッケージの生成型）
│   │       │   ├── category.go                   # ListCategoriesHandler
│   │       │   ├── fetch.go                       # FetchLatestHandler（最新フィード取得）
│   │       │   └── openapi/                       # docs/openapi.yaml から生成された型（oapi-codegen、DO NOT EDIT）
│   │       │       ├── config.yaml                # oapi-codegen 設定（models のみ生成）
│   │       │       ├── generate.go                # go:generate ディレクティブ
│   │       │       └── types.gen.go                # Article・PagedArticles・FetchResult・Error 等
│   │       └── agent/                            # rss-agent（cobra）向けハンドラ
│   │           ├── summarize.go                   # summarize サブコマンド（--feed/--limit フラグ）
│   │           ├── preference.go                  # preference サブコマンド
│   │           └── enrich.go                      # enrich サブコマンド（--limit/--force フラグ）
│   └── driver/
│       ├── readerdb/                            # reader.db への接続・リポジトリ実装
│       │   ├── client.go                        # DB 接続（sql.Open のみ）
│       │   ├── article/
│       │   │   └── article.go                  # ArticleRepository 実装（SQL 文はここ）
│       │   ├── auditlog/
│       │   │   └── auditlog.go                 # AuditLogRepository 実装
│       │   ├── dbmaintenance/
│       │   │   └── dbmaintenance.go            # DBMaintenance 実装（VACUUM・整合性チェック）
│       │   └── feed/
│       │       └── feed.go                     # FeedRepository 実装
│       ├── rss/
│       │   └── reader.go                       # RSSReader 実装（gofeed による HTTP Fetch）
│       └── anthropic/                           # Claude API 連携（エージェント機能。adapter/driver/anthropic の各 interface を実装）
│           ├── loop.go                          # エージェントループ（runAgentLoop・toArticleJSONList）
│           ├── preference.go                    # preferenceAgent（PreferenceAgent 実装）
│           ├── summarize.go                     # summarizeAgent（SummarizeAgent 実装）
│           └── enrich.go                        # enrichAgent（EnrichAgent 実装、DB保存）
├── reader.db                                    # SQLite データベース（gitignore）
├── go.mod
└── go.sum
```

---

## 依存ライブラリ

| ライブラリ | 用途 | 選定理由 |
|----------|------|---------|
| `github.com/spf13/cobra` | CLI サブコマンド管理 | 複数サブコマンドの構造化に適している |
| `github.com/mmcdole/gofeed` | RSS/Atom パース | RSS 2.0・Atom 両対応、メンテ活発 |
| `github.com/mattn/go-sqlite3` | SQLite ドライバ | CGO 使用。devcontainer に GCC あり・純 Go 版は コンパイル時メモリ不足のため除外 |
| `github.com/samber/do/v2` | DI コンテナ | CLI 向きのシンプルな API、コード生成不要 |
| `github.com/anthropics/anthropic-sdk-go` | Claude API クライアント | エージェント機能・記事要約で使用 |
| `github.com/go-chi/chi/v5` | HTTP ルーター | Web ビュー（`cmd/web`）のルーティング・ミドルウェア |
| `github.com/go-chi/cors` | CORS ミドルウェア | figma-mcp 製フロントエンドを別ポートで開発する際の CORS 対応 |
| `github.com/spf13/viper` | 設定ファイル読み込み | `config.yml` から `ANTHROPIC_API_KEY` 等を読み込む |
| `github.com/oapi-codegen/oapi-codegen/v2`（tool） | OpenAPI → Go 型生成 | `docs/openapi.yaml` から `internal/adapter/handler/web/openapi` の型を生成。`go.mod` の `tool` ディレクティブで管理し、ランタイム依存にはしない |

---

## Hook 実装仕様

Claude Code が Bash ツールで `rss-feeder` コマンドを実行する際にフックを挿入する。

### settings.json 設定

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": ".claude/hooks/validate-state.sh",
            "timeout": 10
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": ".claude/hooks/audit-log.sh",
            "timeout": 5,
            "async": true
          }
        ]
      }
    ],
    "Stop": [
      {
        "matcher": ".*",
        "hooks": [
          {
            "type": "command",
            "command": ".claude/hooks/session-cleanup.sh"
          }
        ]
      }
    ]
  }
}
```

### Hook スクリプトの役割

Hook スクリプトは `sqlite3` コマンドを直接呼ばず、**Go バイナリのサブコマンド**を通じて DB を操作する。

| スクリプト | タイミング | 処理内容 |
|----------|----------|---------|
| `validate-state.sh` | PreToolUse | `rss-feeder bookmark <id>` 実行前に `rss-feeder check-article <id>` で ID 存在確認。`rss-feeder reset` 実行前に `rss-feeder check-bookmarked` でお気に入り件数を表示 |
| `audit-log.sh` | PostToolUse | `fetch` / `bookmark` / `reset` 実行後に `rss-feeder audit --action=<action> [--article-id=<id>]` で audit_log に記録 |
| `session-cleanup.sh` | Stop | `rss-feeder maintenance` で VACUUM 実行と整合性チェック |

---

## DI 構成（samber/do）

`main.go` に 1 つの `do.Injector` を置き、driver 層のプロバイダーを登録する。
usecase 層は `do.MustInvoke` でリポジトリを取り出して直接構築し、handler/cli・handler/web・handler/agent に渡す。
`cmd/agent/main.go` も同じ構造で、`internal/driver/anthropic` の各 Agent を `do.Provide` で登録し、`internal/usecase/{summarize,preference,enrich}.go` 経由で `handler/agent` のコマンドに渡す。

```go
// cmd/rss-feeder/main.go
func main() {
    i := do.New()

    // driver 層をコンテナに登録
    do.Provide(i, readerdb.NewClient)               // *sql.DB
    do.Provide(i, dbrepoarticle.NewRepository)      // articlerepo.Repository
    do.Provide(i, dbrepofeed.NewRepository)         // feedrepo.Repository
    do.Provide(i, driverrss.NewReader)              // adapterrss.RSSReader
    do.Provide(i, dbrepoauditlog.NewRepository)     // auditlogrepo.Repository
    do.Provide(i, dbrepodbmaint.NewMaintainer)      // dbmaintrepo.Maintainer

    // migration
    db := do.MustInvoke[*sql.DB](i)
    if err := migration.Run(db); err != nil {
        log.Fatalf("migration failed: %v", err)
    }

    // usecase 層を直接構築（依存はコンテナから取り出す）
    fetchUC           := usecase.NewFetchUsecase(
        do.MustInvoke[articlerepo.Repository](i),
        do.MustInvoke[feedrepo.Repository](i),
        do.MustInvoke[adapterrss.RSSReader](i),
    )
    listUC            := usecase.NewListUsecase(do.MustInvoke[articlerepo.Repository](i))
    bookmarkUC        := usecase.NewBookmarkUsecase(do.MustInvoke[articlerepo.Repository](i))
    resetUC           := usecase.NewResetUsecase(do.MustInvoke[articlerepo.Repository](i))
    searchUC          := usecase.NewSearchUsecase(do.MustInvoke[articlerepo.Repository](i))
    checkArticleUC    := usecase.NewCheckArticleUsecase(do.MustInvoke[articlerepo.Repository](i))
    checkBookmarkedUC := usecase.NewCheckBookmarkedUsecase(do.MustInvoke[articlerepo.Repository](i))
    auditUC           := usecase.NewAuditUsecase(do.MustInvoke[auditlogrepo.Repository](i))
    maintenanceUC     := usecase.NewMaintenanceUsecase(do.MustInvoke[dbmaintrepo.Maintainer](i))
    addFeedUC         := usecase.NewAddFeedUsecase(do.MustInvoke[feedrepo.Repository](i))
    listFeedsUC       := usecase.NewListFeedsUsecase(do.MustInvoke[feedrepo.Repository](i))
    removeFeedUC      := usecase.NewRemoveFeedUsecase(do.MustInvoke[feedrepo.Repository](i))

    // cobra コマンド組み立て
    root := &cobra.Command{Use: "rss-feeder", Short: "RSS フィードを取得・管理する CLI ツール"}
    root.AddCommand(
        cli.NewFetchCommand(fetchUC),
        cli.NewListCommand(listUC),
        cli.NewBookmarkCommand(bookmarkUC),
        cli.NewResetCommand(resetUC),
        cli.NewSearchCommand(searchUC),
        cli.NewCheckArticleCommand(checkArticleUC),
        cli.NewCheckBookmarkedCommand(checkBookmarkedUC),
        cli.NewAuditCommand(auditUC),
        cli.NewMaintenanceCommand(maintenanceUC),
        cli.NewAddFeedCommand(addFeedUC),
        cli.NewListFeedsCommand(listFeedsUC),
        cli.NewRemoveFeedCommand(removeFeedUC),
    )

    if err := root.Execute(); err != nil {
        os.Exit(1)
    }
}
```

### プロバイダーの書き方（例: dbrepoarticle.NewRepository）

`do.Provide` に渡す関数は `(do.Injector, error)` を返す形にする。
依存は `do.MustInvoke[T]` で取り出す。

```go
// internal/driver/readerdb/article/article.go
func NewRepository(i do.Injector) (articlerepo.Repository, error) {
    db := do.MustInvoke[*sql.DB](i)
    return &repository{db: db}, nil
}
```

---

## データフロー（フェーズ 3 保存時）

Hook は Claude Code が Bash ツールで `rss-feeder` を実行するタイミングで発火する。
Go バイナリの内部 SQLite 操作には直接関与しない。

```
Claude Code が Bash ツールで "rss-feeder fetch" を実行
  │
  ├─ [PreToolUse Hook] validate-state.sh
  │    └─ fetch はスキップ（bookmark / reset のみ検証対象）
  │
  ├─ rss-feeder fetch（Go バイナリ実行）
  │    └─ adapter/handler/cli/fetch.go
  │         ├─ usecase/list_feeds.go  # DB からフィード URL 取得
  │         └─ usecase/fetch.go       # 重複チェック・保存指示
  │              ├─ driver/rss/reader.go               # HTTP GET → gofeed パース
  │              └─ driver/readerdb/article/article.go # INSERT OR IGNORE
  │
  └─ [PostToolUse Hook] audit-log.sh
       └─ rss-feeder audit --action=fetch で audit_log に記録
```

---

## テスト戦略

`adapter`（handler）はテスト対象外。`domain`・`usecase`・`driver` の 3 層をテストする。

### domain（純粋なユニットテスト）

外部依存なし。エンティティのメソッドのみをテストする。

```
internal/domain/
  article_test.go   # ToggleBookmark() の状態遷移、MarkAsRead() など
```

```go
// 例: article_test.go
func TestArticle_ToggleBookmark(t *testing.T) {
    a := domain.Article{Bookmarked: false}
    a.ToggleBookmark()
    if !a.Bookmarked {
        t.Error("expected Bookmarked to be true after first toggle")
    }
    a.ToggleBookmark()
    if a.Bookmarked {
        t.Error("expected Bookmarked to be false after second toggle")
    }
}
```

---

### usecase（モックを使ったユニットテスト）

`adapter` のインターフェースをモック化し、ビジネスロジックだけを検証する。

```
internal/usecase/
  fetch_test.go            # 新規記事のみ保存される・重複はスキップされる
  list_test.go             # 未読フィルタ・全件・お気に入りフィルタが正しく委譲される
  list_categories_test.go  # カテゴリ一覧（DISTINCT）取得ロジック
  bookmark_test.go         # トグル動作・存在しない ID のエラーハンドリング
  reset_test.go            # お気に入り記事が削除対象に含まれないことの確認
  search_test.go           # キーワード一致・0件・bookmarked フィルタ・エラーハンドリング
  add_feed_test.go         # フィード登録ロジック
  list_feeds_test.go       # フィード一覧取得ロジック
  remove_feed_test.go      # フィード削除ロジック
  audit_test.go            # audit_log への記録ロジック
  check_article_test.go    # 記事 ID 存在確認ロジック
  check_bookmarked_test.go # お気に入り件数確認ロジック
  maintenance_test.go      # VACUUM・整合性チェックロジック
  summarize_test.go        # SummarizeAgent への委譲・エラー伝播
  preference_test.go       # PreferenceAgent への委譲・エラー伝播
  enrich_test.go           # EnrichAgent への委譲・エラー伝播
```

```go
// 例: fetch_test.go（モック使用）
func TestFetchUsecase_SkipsDuplicates(t *testing.T) {
    repo := &mockArticleRepo{existing: map[string]bool{"https://example.com/1": true}}
    uc := NewFetchUsecase(repo, &mockFeedRepo{}, &mockRSSReader{
        articles: []domain.Article{{URL: "https://example.com/1", Title: "Test"}},
    })

    result, err := uc.Execute(context.Background(), []string{"https://feed.example.com"})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result.TotalSaved() != 0 {
        t.Errorf("Saved: got %d, want 0", result.TotalSaved())
    }
    if result.TotalSkipped() != 1 {
        t.Errorf("Skipped: got %d, want 1", result.TotalSkipped())
    }
}
```

---

### driver（インメモリ SQLite を使った統合テスト）

実際の SQLite（インメモリ）と HTTP モックサーバーを使い、実装の正確性を検証する。

```
internal/driver/
  readerdb/
    article/
      article_test.go  # INSERT・重複スキップ・UPDATE・DELETE・LIKE 検索の SQL 動作確認
    feed/
      feed_test.go     # feeds テーブル操作
  rss/
    reader_test.go     # httptest.NewServer でモック RSS を返し、パースを検証
```

```go
// 例: article_test.go（インメモリ SQLite）
func TestArticleRepository_Save_Duplicate(t *testing.T) {
    ctx := context.Background()
    r := newRepo(t)

    r.Save(ctx, makeArticle("https://example.com/1"))
    err := r.Save(ctx, makeArticle("https://example.com/1"))
    if !errors.Is(err, articlerepo.ErrDuplicate) {
        t.Errorf("expected ErrDuplicate, got %v", err)
    }
}
```
