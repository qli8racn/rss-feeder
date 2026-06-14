# 設計：記事メタデータ拡充（出版元・サムネイル・要約・カテゴリ）

## スキーマ変更

既存 `articles` テーブルに 4 カラムを追加する。`migration.Run` は `CREATE TABLE IF NOT EXISTS` のみのため、
既存 DB には反映されない。`ALTER TABLE ... ADD COLUMN` を個別に実行し、
SQLite の `duplicate column name` エラーは無視する（複数回実行されても安全にするため）。

```sql
CREATE TABLE IF NOT EXISTS articles (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    feed_id      INTEGER NOT NULL,
    url          TEXT UNIQUE NOT NULL,
    title        TEXT NOT NULL,
    content      TEXT,
    published_at DATETIME,
    read         BOOLEAN DEFAULT 0,
    bookmarked   BOOLEAN DEFAULT 0,
    fetched_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    publisher    TEXT,
    thumbnail_url TEXT,
    summary      TEXT,
    category     TEXT,
    FOREIGN KEY(feed_id) REFERENCES feeds(id) ON DELETE CASCADE
);

-- 既存DB向け（duplicate column name エラーは無視する）
ALTER TABLE articles ADD COLUMN publisher TEXT;
ALTER TABLE articles ADD COLUMN thumbnail_url TEXT;
ALTER TABLE articles ADD COLUMN summary TEXT;
ALTER TABLE articles ADD COLUMN category TEXT;
```

## domain.Article の拡張

```go
type Article struct {
    ID           int64
    FeedID       int64
    URL          string
    Title        string
    Content      string
    PublishedAt  time.Time
    Read         bool
    Bookmarked   bool
    FetchedAt    time.Time
    FeedURL      string
    Publisher    string // 追加: フィードの発行元（Feed.Title）
    ThumbnailURL string // 追加: サムネイル画像URL
    Summary      string // 追加: AIによる要約（未設定なら空文字）
    Category     string // 追加: AIによるカテゴリ（未設定なら空文字）
}
```

## RSS reader の拡張（`internal/driver/rss/reader.go`）

`Fetch` は既に `feed.Title` を戻り値として返しているため、これを `Article.Publisher` に設定する
（呼び出し元の `usecase/fetch.go` で各 `Article` にセットする）。

サムネイル取得は `gofeed.Item` から以下の優先順位で探索するヘルパー関数 `extractThumbnail(item *gofeed.Item) string` を追加する。

1. `item.Extensions["media"]["thumbnail"]` の `url` 属性（`<media:thumbnail>`）
2. `item.Enclosures` のうち `Type` が `image/` で始まるものの `URL`
3. `item.Image.URL`（`<itunes:image>` 等）

いずれも見つからなければ空文字を返す。

## usecase / repository の変更

### `internal/usecase/fetch.go`

- `reader.Fetch()` の戻り値 `feedTitle` を各 `Article.Publisher` に設定する
- `extractThumbnail` の結果を `Article.ThumbnailURL` に設定する（reader 側で詰めて返す形でも可。
  実装時は `domain.Article` に持たせた状態で `Save` に渡す）

### `internal/adapter/driver/readerdb/article/article.go`（インターフェース）

```go
type Repository interface {
    // ...既存...
    UpdateEnrichment(ctx context.Context, id int64, summary, category string) error
    FindWithoutSummary(ctx context.Context, limit int) ([]domain.Article, error)
}
```

### `internal/driver/readerdb/article/article.go`（実装）

- `Save`: INSERT 文に `publisher`, `thumbnail_url` を追加
- `scanArticle` / `scanArticleWithFeed`: SELECT に `publisher, thumbnail_url, summary, category` を追加し、
  `sql.NullString` で受けて `domain.Article` にセット
- `UpdateEnrichment`: `UPDATE articles SET summary = ?, category = ? WHERE id = ?`
- `FindWithoutSummary`: `SELECT ... WHERE summary IS NULL OR summary = '' LIMIT ?`

## 要約・カテゴリ付与エージェント（`rss-agent enrich`）

既存の `internal/driver/anthropic/summarize.go`（その場で要約して表示するのみ）とは別に、
DB へ書き込む新エージェント `internal/driver/anthropic/enrich.go` を追加する。

### 処理フロー

```
rss-agent enrich [--limit N] [--force]
  └─ cmd/agent/main.go: enrichCmd
       └─ driveranthropic.NewEnrichAgent(articleRepo).Run(ctx, EnrichOptions{Limit, Force})
            ├─ Force=false: articleRepo.FindWithoutSummary(ctx, limit)
            ├─ Force=true : articleRepo.FetchLatest(ctx, limit, "")
            └─ 各記事について Claude に「要約 + カテゴリ」を JSON で出力させる
                 └─ articleRepo.UpdateEnrichment(ctx, id, summary, category)
```

- 1 記事ずつ、または `Limit` 件を1リクエストでまとめて処理するかは実装時に検討する
  （まとめる場合はレスポンスを `[{id, summary, category}, ...]` の JSON 配列として受け取り、
  `json.Unmarshal` 後に1件ずつ `UpdateEnrichment` する）
- `runAgentLoop`（`internal/driver/anthropic/loop.go`）を再利用せず、
  ツール呼び出し不要な単発の `Messages.New` で JSON 出力を得るシンプルな実装にする
  （要約対象の記事は事前に `FindWithoutSummary` で取得済みのため、ツールで取得し直す必要がない）

## API / 表示への反映

### `internal/adapter/handler/serve.go`

`Article` DTO（JSON 表現）に `publisher`, `thumbnail_url`, `summary`, `category` を追加する。

```json
{
  "id": 1,
  "feed_id": 1,
  "feed_url": "https://example.com/feed",
  "url": "https://example.com/articles/1",
  "title": "記事タイトル",
  "content": "本文",
  "published_at": "2026-06-14T00:00:00Z",
  "read": true,
  "bookmarked": false,
  "fetched_at": "2026-06-14T01:00:00Z",
  "publisher": "Example Tech Blog",
  "thumbnail_url": "https://example.com/images/thumb.jpg",
  "summary": "この記事は...",
  "category": "Tech"
}
```

### `internal/adapter/handler/list.go` / `search.go`（CLI 標準出力）

既存の `FeedURL` 表示を `Publisher`（未設定時は `FeedURL` にフォールバック）に変更する。

## 追加・変更ファイル

### 追加ファイル

| ファイル | 役割 |
|---------|------|
| `internal/driver/anthropic/enrich.go` | `EnrichAgent`：未要約記事に要約・カテゴリを付与しDBに保存 |
| `cmd/agent/main.go`（変更） | `enrich` サブコマンド追加 |

### 変更ファイル

| ファイル | 変更内容 |
|---------|---------|
| `internal/migration/migration.go` | `articles` テーブルに4カラム追加（`CREATE TABLE` + `ALTER TABLE`） |
| `internal/domain/article.go` | `Publisher` / `ThumbnailURL` / `Summary` / `Category` フィールド追加 |
| `internal/driver/rss/reader.go` | `extractThumbnail` 追加、`Publisher` 設定 |
| `internal/usecase/fetch.go` | `Publisher` / `ThumbnailURL` を `Article` にセット |
| `internal/adapter/driver/readerdb/article/article.go` | `UpdateEnrichment` / `FindWithoutSummary` をインターフェースに追加 |
| `internal/driver/readerdb/article/article.go` | `Save` / `scanArticle*` の変更、`UpdateEnrichment` / `FindWithoutSummary` 実装 |
| `internal/adapter/handler/serve.go` | DTO に4フィールド追加 |
| `internal/adapter/handler/list.go` / `search.go` | `Publisher` 表示への変更 |

## アーキテクチャ上の注意点

- `summarize.go`（既存の対話的要約）と `enrich.go`（DB永続化用バッチ要約）は責務が異なるため別ファイル・別コマンドとする
- `domain.Article` への JSON タグは付与せず、Web API は `serve.go` 側で DTO 変換する方針を継続する（フェーズ9と同様）
