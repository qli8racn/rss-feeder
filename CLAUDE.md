# RSS Feeder

Go 製 RSS リーダー CLI。Claude Code のエージェント・Hooks・SQLite 連携を学習するプロジェクト。

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
> devcontainer.json に `--memory=3g --memory-swap=6g` を設定済み（OOM 対策）。
> それでも失敗する場合は並列数を制限する（詳細は `docs/design.md` 参照）：
> ```bash
> go build -p 1 -o bin/rss-agent ./cmd/agent
> ```

---

## CLI Commands

### rss-feeder

```bash
bin/rss-feeder fetch
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

## Hooks

Hook スクリプトは `.claude/hooks/` に配置する。
設定は `.claude/settings.json` の `hooks` セクションで行う（詳細は `docs/design.md` 参照）。

---

## Troubleshooting

```bash
# DB 直接確認
sqlite3 reader.db "SELECT * FROM articles LIMIT 5"
sqlite3 reader.db "SELECT * FROM audit_log ORDER BY timestamp DESC LIMIT 10"

# Hook が動かない場合: Ctrl+O で verbose mode を有効化
```
