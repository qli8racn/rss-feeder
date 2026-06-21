# RSS Feeder

Go 製 RSS リーダー CLI。

- 機能要求 → `docs/product-requirements.md`
- 機能設計・技術仕様・アーキテクチャ → `docs/design.md`

---

## 開発フロー（docs/steering 必須化）

新しい機能の追加・既存機能の改修を行う場合は、**必ず** `docs/steering/YYYYMMDD_変更内容を表す英語スラッグ/` を作成し、
以下の3ファイルを揃えること（単純な bug fix・chore・typo修正など、設計判断を伴わない変更は対象外）。

- `requirements.md` — 概要・背景・目的・受け入れ条件・スコープ外
- `design.md` — 構成図・実装方針・複数の選択肢を検討した場合はその比較と採用理由
- `tasklist.md` — 実施項目のチェックボックス一覧（未実施項目は `[ ]` のまま残す）

複数のコミットに渡る一連の変更は、同じフェーズディレクトリに追記して1つにまとめる
（コミット単位で都度新規ディレクトリを作る必要はない）。既存の `docs/steering/` 以下の各ディレクトリを
フォーマットの参考にすること。

---

## Setup

```bash
go mod tidy
go build -o bin/rss-feeder ./cmd/rss-feeder
go build -o bin/web        ./cmd/web
go build -o bin/rss-agent  ./cmd/agent
```

DB 初期化は初回起動時に自動実行される。

`rss-agent` 用の `ANTHROPIC_API_KEY` は `internal/config/config.yml` で管理する。
`config.yml` が無い場合は `internal/config/config.example.yml` をコピーして作成し、
`anthropic_api_key` に値を設定する（`config.yml` は Git 管理対象外）。

> `ANTHROPIC_API_KEY` は claude.ai（コンシューマー向けチャット）のログイン情報ではなく、
> Claude Console（https://console.anthropic.com）の「API Keys」から発行する API キーを使用する。

```bash
cp internal/config/config.example.yml internal/config/config.yml
```

> **Note:** メモリが少ない環境では `rss-agent` のビルドで OOM が発生することがある。
> devcontainer.json に `--memory=6g --memory-swap=12g` を設定済み（OOM 対策）。
> それでも失敗する場合は並列数を制限する（詳細は `docs/design.md` 参照）：
> ```bash
> GOMAXPROCS=1 GOFLAGS="-gcflags=all=-l=0" go build -p 1 -o bin/rss-agent ./cmd/agent
> ```
>
> **必須:** `internal/driver/anthropic` を使用するビルド・テスト（`rss-agent` 関連）では、必ず上記の環境変数と `-p 1` オプションを指定すること。
> `go build ./...` のような全パッケージ一括ビルドは禁止。対象パッケージを明示すること。
>
> 同様に `go test ./...` も `internal/driver/anthropic` を含む場合 OOM するため、
> `go test $(go list ./... | grep -v internal/driver/anthropic)` のように除外して実行すること。

---

## CLI Commands

### rss-feeder

```bash
bin/rss-feeder add-feed <url>             # RSS フィードを DB に登録
bin/rss-feeder list-feeds                 # 登録済みフィード一覧
bin/rss-feeder remove-feed <id>           # フィードを削除（記事も連動削除）
bin/rss-feeder fetch                      # 登録済みフィードを取得して DB に保存
bin/rss-feeder list [--all | --bookmarked] [--category <name>]
bin/rss-feeder bookmark <id>
bin/rss-feeder reset [-y]
bin/rss-feeder search <keyword> [--bookmarked] [--category <name>]
bin/rss-feeder categories                 # 記事に付与済みのカテゴリ一覧
bin/rss-feeder backfill-metadata          # 既存記事の出版元・サムネイルを補完（再取得で判明した分のみ）
```

### rss-agent

```bash
bin/rss-agent summarize [--feed <url>] [--limit <n>]  # 最新記事を AI で要約
bin/rss-agent preference                               # ブックマークから趣向を分析
bin/rss-agent enrich [--limit <n>] [--force] [--batch-size <n>] [--concurrency <n>]  # 記事に要約・カテゴリを付与してDBに保存
```

`ANTHROPIC_API_KEY` が必要（`config.yml` の `anthropic_api_key` または環境変数で設定）。

### Web UI（cmd/web）

```bash
cd web/frontend && npm ci && npm run build  # web/static/ にビルド成果物を出力
bin/web [--port 8080] [--static-dir web/static]
```

`GET /api/articles`・`GET /api/articles/search`・`POST /api/articles/{id}/bookmark`・`GET /api/categories`・`POST /api/articles/fetch` を提供する（詳細は `docs/design.md`）。
フロントエンドの動作確認方針（型チェック・ユニットテスト・ビルド確認のみ、ブラウザ目視確認はしない）は `CLAUDE.md` を参照。

API 仕様は `docs/openapi.yaml`（OpenAPI 3.0）で管理する。仕様変更時は以下を再生成すること（詳細は `docs/design.md`）。

```bash
go generate ./internal/adapter/handler/web/openapi/...                          # Go 側の型（internal/adapter/handler/web/openapi/types.gen.go）
cd web/frontend && npm run generate:api                                         # フロントエンド側の型（src/api/schema.gen.ts）
```

---

## Test

```bash
go test ./internal/domain/...
go test ./internal/usecase/...
go test $(go list ./internal/driver/... | grep -v internal/driver/anthropic)
```

---

## Troubleshooting

```bash
# DB 直接確認
sqlite3 reader.db "SELECT * FROM articles LIMIT 5"
sqlite3 reader.db "SELECT * FROM audit_log ORDER BY timestamp DESC LIMIT 10"
```
