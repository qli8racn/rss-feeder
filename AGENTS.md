# RSS Feeder

Go 製 RSS リーダー CLI。

- 機能要求 → `docs/product-requirements.md`
- 機能設計・技術仕様・アーキテクチャ → `docs/design.md`

---

## Setup

```bash
go mod tidy
go build -o bin/rss-feeder ./cmd/rss-feeder
go build -o bin/rss-agent  ./cmd/agent
```

DB 初期化は初回起動時に自動実行される。

> **Note:** メモリが少ない環境では `rss-agent` のビルドで OOM が発生することがある。
> devcontainer.json に `--memory=6g --memory-swap=12g` を設定済み（OOM 対策）。
> それでも失敗する場合は並列数を制限する（詳細は `docs/design.md` 参照）：
> ```bash
> GOMAXPROCS=1 GOFLAGS="-gcflags=all=-l=0" go build -p 1 -o bin/rss-agent ./cmd/agent
> ```
>
> **必須:** `internal/driver/anthropic` を使用するビルド・テスト（`rss-agent` 関連）では、必ず上記の環境変数と `-p 1` オプションを指定すること。
> `go build ./...` のような全パッケージ一括ビルドは禁止。対象パッケージを明示すること。

---

## CLI Commands

### rss-feeder

```bash
bin/rss-feeder add-feed <url>             # RSS フィードを DB に登録
bin/rss-feeder list-feeds                 # 登録済みフィード一覧
bin/rss-feeder remove-feed <id>           # フィードを削除（記事も連動削除）
bin/rss-feeder fetch                      # 登録済みフィードを取得して DB に保存
bin/rss-feeder list [--all | --bookmarked]
bin/rss-feeder bookmark <id>
bin/rss-feeder reset [-y]
bin/rss-feeder search <keyword>
```

### rss-agent

```bash
bin/rss-agent summarize [--feed <url>] [--limit <n>]  # 最新記事を AI で要約
bin/rss-agent preference                               # ブックマークから趣向を分析
```

`ANTHROPIC_API_KEY` 環境変数が必要。

---

## Test

```bash
go test ./internal/domain/...
go test ./internal/usecase/...
go test ./internal/driver/...
```

---

## Troubleshooting

```bash
# DB 直接確認
sqlite3 reader.db "SELECT * FROM articles LIMIT 5"
sqlite3 reader.db "SELECT * FROM audit_log ORDER BY timestamp DESC LIMIT 10"
```
