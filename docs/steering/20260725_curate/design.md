# 設計：キュレーション機能

## 全体構成

```
bin/rss-agent curate [--limit <n>]
  └─ agent.NewCurateCommand
       └─ usecase.CurateUsecase.Execute
            └─ driver/anthropic.curateAgent.Run
                 ├─ tool: fetch_recent_articles(limit)
                 │    └─ articlerepo.Repository.FetchLatest
                 └─ tool: fetch_bookmarked_articles()
                      └─ articlerepo.Repository.FindBookmarked
```

## アーキテクチャ

既存の `preference` エージェントと同じパターンを踏襲する。

```
adapter/driver/anthropic/curate.go   … CurateAgent インターフェース・CurateOptions
driver/anthropic/curate.go           … curateAgent 実装（ツール使用エージェントループ）
usecase/curate.go                    … CurateUsecase
adapter/handler/agent/curate.go      … cobra CLI ハンドラ
cmd/agent/main.go                    … DI・コマンド登録
```

依存方向: `adapter/handler → usecase → adapter/driver(interface) ← driver`

## モデル選択

`claude-opus-4-8`（preference と同様）。
複数記事の比較・趣向との照合・推薦理由の生成には推論能力が必要なため、Haiku は不使用。
Thinking は adaptive で有効化する。

## ツール定義

### `fetch_recent_articles`

```json
{
  "name": "fetch_recent_articles",
  "description": "直近の記事を取得する。summary・category が付与されている場合は含む。",
  "input_schema": {
    "properties": {
      "limit": { "type": "integer", "description": "取得件数の上限（省略時はデフォルト値）" }
    }
  }
}
```

返却形式（JSON 配列）:

```json
[
  { "id": 1, "title": "...", "url": "...", "category": "Tech", "summary": "...", "published_at": "..." }
]
```

### `fetch_bookmarked_articles`

```json
{
  "name": "fetch_bookmarked_articles",
  "description": "ブックマーク済みの記事を取得する。趣向把握に使用する。"
}
```

返却形式は `fetch_recent_articles` と同形式（summary があれば含む）。

## JSON 型

`enrichedArticleJSON`（curate.go 内に定義）:

```go
type enrichedArticleJSON struct {
    ID          int64  `json:"id"`
    Title       string `json:"title"`
    URL         string `json:"url"`
    Category    string `json:"category,omitempty"`
    Summary     string `json:"summary,omitempty"`
    PublishedAt string `json:"published_at,omitempty"`
}
```

既存の `articleJSON`（loop.go）は title/URL のみで curate には情報量が不足するため、
curate 専用の struct を定義する。

## 出力形式

Markdown 形式でターミナル表示:

```
## 今日のおすすめ記事（N件）

### 1. [タイトル](URL)
**カテゴリ**: Tech
**推薦理由**: ...

### 2. ...
```

## コスト考慮

Opus モデルのため preference 同様コストがかかる。
`fetch_recent_articles` のデフォルト limit を 30 に絞り入力トークンを抑える。
実行後に `logUsage` で cost を stderr に出力する（既存パターン通り）。
