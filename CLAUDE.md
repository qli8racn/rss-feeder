# RSS Feeder

Go 製 RSS リーダー CLI。Claude Code のエージェント・Hooks・SQLite 連携を学習するプロジェクト。

- 機能要求 → `docs/product-requirements.md`
- 機能設計・技術仕様・アーキテクチャ → `docs/design.md`

---

## Setup

```bash
go mod tidy
go build -o rss-feeder ./cmd/rss-feeder
```

DB 初期化は初回起動時に自動実行される。

---

## CLI Commands

```bash
rss-feeder fetch
rss-feeder list [--all | --bookmarked]
rss-feeder bookmark <id>
rss-feeder reset [-y]
```

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
