# 設計：フィード推薦（discover）

## 全体構成

```
bin/rss-agent discover
  └─ agent.NewDiscoverCommand
       └─ usecase.DiscoverUsecase.Execute
            └─ driver/anthropic.discoverAgent.Run
                 ├─ tool: fetch_bookmarked_articles()
                 │    └─ articlerepo.Repository.FindBookmarked
                 └─ tool: fetch_registered_feeds()
                      └─ feedrepo.Repository.ListAll
```

## アーキテクチャ

curate・preference と同パターン。

```
adapter/driver/anthropic/discover.go   … DiscoverAgent インターフェース
driver/anthropic/discover.go           … discoverAgent 実装（ツール使用エージェントループ）
usecase/discover.go                    … DiscoverUsecase
adapter/handler/agent/discover.go      … cobra CLI ハンドラ
cmd/agent/main.go                      … DI 登録・コマンド追加
```

## モデル・Thinking

`claude-opus-4-8` + Thinking adaptive（curate・preference と同様）。
知識ベースから RSS フィード URL を生成する推論が必要なため。

## ツール定義

### `fetch_bookmarked_articles`

curate.go と同じ実装（ブックマーク記事一覧、最大 50 件）。

### `fetch_registered_feeds`

```json
{
  "name": "fetch_registered_feeds",
  "description": "現在登録済みの RSS フィード一覧を取得する。重複推薦の回避に使用する。"
}
```

返却形式（JSON 配列）:

```json
[{ "feed_url": "https://example.com/feed", "title": "Example Blog" }]
```

## JSON 型

`feedJSON`（discover.go 内に定義）:

```go
type feedJSON struct {
    FeedURL string `json:"feed_url"`
    Title   string `json:"title"`
}
```

## DI 登録の変更（cmd/agent/main.go）

feed repo が agent バイナリの DI コンテナに未登録のため追加する:

```go
import dbrepoFeed "github.com/qli8racn/rss-feeder/internal/driver/readerdb/feed"

do.Provide(i, dbrepoFeed.NewRepository)
do.Provide(i, driveranthropic.NewDiscoverAgent)
```

## ログ出力（stderr）

```
[discover] 開始しています...
[discover] ブックマーク記事: N件取得
[discover] 登録済みフィード: N件取得
[rss-agent] model=claude-opus-4-8 input=... output=... cost=$... elapsed=...
```
