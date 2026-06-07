# 設計書（機能設計 + 技術仕様）

## CLI コマンド設計（機能設計）

### コマンド一覧

```
rss-feeder <command> [flags]
```

| コマンド | 概要 | フェーズ |
|--------|------|---------|
| `fetch` | feeds.txt を読み込み、記事を取得して DB に保存 | 1・2・3 |
| `list` | DB 保存済みの記事を一覧表示 | 4 |
| `bookmark <id>` | 記事のお気に入りをトグル（登録/解除） | 5 |
| `reset` | お気に入り以外の記事を削除 | 6 |

---

### `fetch`

```bash
rss-feeder fetch
```

- `feeds.txt` の URL を上から順に処理する
- 取得成功時：新規保存した記事のタイトルと URL を出力する
- 既存 URL はスキップし「スキップ: N 件」を末尾に表示する
- 取得失敗時：エラーを表示して次の URL に続行する

**出力イメージ:**
```
[1/3] https://example.com/feed.xml
  + Go 1.23 リリースノート  https://example.com/go123
  + net/http の新機能       https://example.com/nethttp
  スキップ: 5 件

[2/3] https://blog.example.org/rss
  エラー: タイムアウト (10s)

完了: 新規 2 件 / スキップ 5 件 / エラー 1 件
```

---

### `list`

```bash
rss-feeder list [flags]
```

| フラグ | 説明 |
|------|------|
| （なし） | 未読記事のみ表示（デフォルト） |
| `--all` | 全件表示 |
| `--bookmarked` | お気に入りのみ表示 |

表示した記事は `read = 1` に自動更新する（`list --all` や `list --bookmarked` でも同様）。

**出力イメージ:**
```
ID   タイトル                          公開日時              既読  お気に入り
---  --------------------------------  --------------------  ----  ----------
 42  Go 1.23 リリースノート            2024-08-13 12:00      -     -
 43  net/http の新機能                 2024-08-13 12:01      -     ★
```

---

### `bookmark <id>`

```bash
rss-feeder bookmark 42
```

- 現在 `bookmarked = 0` → `1` に変更し「お気に入りに登録しました」と表示
- 現在 `bookmarked = 1` → `0` に変更し「お気に入りを解除しました」と表示
- 存在しない ID の場合：エラーを表示して終了コード 1 で終了

---

### `reset`

```bash
rss-feeder reset [-y]
```

- `-y` なし：削除対象件数を表示して確認を求める
- `-y` あり：確認なしで即時削除する

**出力イメージ（確認あり）:**
```
お気に入り以外の記事 38 件を削除します。よろしいですか？ [y/N]: y
削除完了: 38 件（お気に入り: 5 件 を保持）
```

---

## パッケージ構成（技術仕様）

### 層の依存方向

```
adapter(handler) → usecase → domain
driver           → adapter(interface)
```

`usecase` と `domain` は外側の実装（SQL・HTTP）を知らない。`driver` が `adapter` のインターフェースを実装することで依存を逆転させる。

### ディレクトリ構成

```
rss-feeder/
├── cmd/
│   └── rss-feeder/
│       └── main.go                         # Composition Root・samber/do コンテナ構築・サブコマンド登録
├── internal/
│   ├── domain/
│   │   ├── article.go                      # Article エンティティ・ToggleBookmark() など
│   │   └── feed.go                         # Feed エンティティ
│   ├── migration/
│   │   └── migration.go                    # DB スキーマ作成（Run(*sql.DB) error）
│   ├── usecase/
│   │   ├── fetch.go                        # 取得・重複チェック・保存の orchestration
│   │   ├── list.go                         # 記事一覧取得ロジック
│   │   ├── bookmark.go                     # お気に入りトグルロジック
│   │   └── reset.go                        # 非お気に入り記事削除ロジック
│   ├── adapter/
│   │   ├── driver/
│   │   │   ├── file/
│   │   │   │   └── feeds_reader.go         # FeedsReader interface
│   │   │   └── rss/
│   │   │       └── rss_reader.go           # RSSReader interface
│   │   ├── repository/
│   │   │   ├── article/
│   │   │   │   └── article.go             # ArticleRepository interface
│   │   │   └── feed/
│   │   │       └── feed.go                # FeedRepository interface
│   │   └── handler/
│   │       ├── fetch.go                   # cobra コマンド → FetchUsecase 呼び出し
│   │       ├── list.go
│   │       ├── bookmark.go
│   │       └── reset.go
│   └── driver/
│       ├── readerdb/                       # reader.db への接続・リポジトリ実装
│       │   ├── client.go                   # DB 接続（sql.Open のみ）
│       │   ├── article/
│       │   │   └── article.go             # ArticleRepository 実装（SQL 文はここ）
│       │   └── feed/
│       │       └── feed.go                # FeedRepository 実装
│       ├── file/
│       │   └── feeds_reader.go            # FeedsReader 実装（feeds.txt 読み込み）
│       └── rss/
│           └── reader.go                  # RSSReader 実装（gofeed による HTTP Fetch）
├── feeds.txt                               # RSS フィード URL リスト（1行1URL、# コメント・空行スキップ）
├── reader.db                               # SQLite データベース（gitignore）
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
具体的なサブコマンド設計はツール実装後に確定する。

| スクリプト | タイミング | 処理内容 |
|----------|----------|---------|
| `validate-state.sh` | PreToolUse | `rss-feeder bookmark <id>` / `rss-feeder reset` 実行前に Go バイナリ経由で対象 ID の存在確認・`bookmarked=1` 記事の保護を検証 |
| `audit-log.sh` | PostToolUse | 書き込み操作後に Go バイナリ経由で `audit_log` に記録 |
| `session-cleanup.sh` | Stop | Go バイナリ経由で `VACUUM` 実行と整合性チェック |

---

## DI 構成（samber/do）

`main.go` に 1 つの `do.Injector` を置き、全レイヤーのプロバイダーを登録する。
各ハンドラーはコンテナから自分が必要な usecase だけを取り出す。

```go
// cmd/rss-feeder/main.go
func main() {
    i := do.New()

    // driver 層
    do.Provide(i, sqlite.NewClient)           // *sql.DB
    do.Provide(i, sqlite.NewArticleRepository) // adapter.ArticleRepository
    do.Provide(i, sqlite.NewFeedRepository)    // adapter.FeedRepository
    do.Provide(i, rss.NewReader)              // adapter.RSSReader

    // usecase 層（依存は do が自動解決）
    do.Provide(i, usecase.NewFetchUsecase)
    do.Provide(i, usecase.NewListUsecase)
    do.Provide(i, usecase.NewBookmarkUsecase)
    do.Provide(i, usecase.NewResetUsecase)

    // cobra コマンド組み立て
    root := &cobra.Command{Use: "rss-feeder"}
    root.AddCommand(
        handler.NewFetchCommand(do.MustInvoke[*usecase.FetchUsecase](i)),
        handler.NewListCommand(do.MustInvoke[*usecase.ListUsecase](i)),
        handler.NewBookmarkCommand(do.MustInvoke[*usecase.BookmarkUsecase](i)),
        handler.NewResetCommand(do.MustInvoke[*usecase.ResetUsecase](i)),
    )

    if err := root.Execute(); err != nil {
        os.Exit(1)
    }
}
```

### プロバイダーの書き方（例: sqlite.NewArticleRepository）

`do.Provide` に渡す関数は `(do.Injector, error)` を返す形にする。
依存は `do.Invoke[T]` で取り出す。

```go
// internal/driver/sqlite/article.go
func NewArticleRepository(i do.Injector) (adapter.ArticleRepository, error) {
    db := do.MustInvoke[*sql.DB](i)
    return &articleRepository{db: db}, nil
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
  │    └─ 必要に応じて rss-feeder <check-subcommand> で事前検証
  │
  ├─ rss-feeder fetch（Go バイナリ実行）
  │    └─ adapter/handler/fetch.go
  │         └─ usecase/fetch.go     # feeds.txt 読み込み・重複チェック・保存指示
  │              ├─ driver/rss/reader.go      # HTTP GET → gofeed パース
  │              └─ driver/sqlite/article.go  # INSERT OR IGNORE
  │
  └─ [PostToolUse Hook] audit-log.sh
       └─ rss-feeder <audit-subcommand> で audit_log に記録
```

---

## テスト戦略

`adapter`（handler）はテスト対象外。`domain`・`usecase`・`driver` の 3 層をテストする。

### domain（純粋なユニットテスト）

外部依存なし。エンティティのメソッドのみをテストする。

```
internal/domain/
  article_test.go   # ToggleBookmark() の状態遷移、MarkAsRead() など
  feed_test.go      # Feed バリデーション等
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
  fetch_test.go     # 新規記事のみ保存される・重複はスキップされる
  list_test.go      # 未読フィルタ・全件・お気に入りフィルタが正しく委譲される
  bookmark_test.go  # トグル動作・存在しない ID のエラーハンドリング
  reset_test.go     # お気に入り記事が削除対象に含まれないことの確認
```

```go
// 例: fetch_test.go（モック使用）
func TestFetchUsecase_SkipsDuplicates(t *testing.T) {
    repo := &mockArticleRepository{existing: []string{"https://example.com/1"}}
    reader := &mockRSSReader{articles: []domain.Article{{URL: "https://example.com/1"}}}
    uc := usecase.NewFetchUsecase(repo, reader)

    result, err := uc.Execute(context.Background(), []string{"https://feed.example.com"})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result.Saved != 0 {
        t.Errorf("Saved: got %d, want 0", result.Saved)
    }
    if result.Skipped != 1 {
        t.Errorf("Skipped: got %d, want 1", result.Skipped)
    }
}
```

---

### driver（インメモリ SQLite を使った統合テスト）

実際の SQLite（インメモリ）と HTTP モックサーバーを使い、実装の正確性を検証する。

```
internal/driver/
  sqlite/
    article_test.go  # INSERT・重複スキップ・UPDATE・DELETE の SQL 動作確認
    feed_test.go     # feeds テーブル操作
  rss/
    reader_test.go   # httptest.NewServer でモック RSS を返し、パースを検証
```

```go
// 例: article_test.go（インメモリ SQLite）
func TestArticleRepository_Save_SkipsDuplicate(t *testing.T) {
    db := sqlite.NewInMemoryClient(t)
    repo := sqlite.NewArticleRepository(db)

    article := domain.Article{URL: "https://example.com/1", Title: "Test"}
    if err := repo.Save(ctx, article); err != nil {
        t.Fatalf("first save failed: %v", err)
    }
    if err := repo.Save(ctx, article); err != nil { // 2回目は重複→エラーなしでスキップ
        t.Fatalf("second save (duplicate) failed: %v", err)
    }

    count, _ := repo.Count(ctx)
    if count != 1 {
        t.Errorf("Count: got %d, want 1", count)
    }
}
```
