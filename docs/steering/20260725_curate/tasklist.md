# タスクリスト：キュレーション機能

## 削除

- [x] `.claude/agents/rss-orchestrator.md` を削除（逐次実行スクリプトに過ぎず、エージェントとして価値が薄い）

## docs/steering

- [x] `docs/steering/20260725_curate/requirements.md` 作成
- [x] `docs/steering/20260725_curate/design.md` 作成
- [x] `docs/steering/20260725_curate/tasklist.md` 作成

## 実装

- [x] `internal/adapter/driver/anthropic/curate.go` — `CurateAgent` インターフェース・`CurateOptions` 定義
- [x] `internal/driver/anthropic/curate.go` — `curateAgent` 実装（ツール使用エージェントループ）
- [x] `internal/usecase/curate.go` — `CurateUsecase`
- [x] `internal/adapter/handler/agent/curate.go` — cobra CLI ハンドラ（`--limit` フラグ）
- [x] `cmd/agent/main.go` — `NewCurateAgent` の DI 登録・`NewCurateCommand` の追加

## テスト・確認

- [x] `go build -p 1 -o bin/rss-agent ./cmd/agent` でビルドが通ること
- [ ] `bin/rss-agent curate` を実行し、推薦記事リストが出力されること
- [ ] `bin/rss-agent curate --limit 10` で件数を変えられること

## レビュー指摘対応

- [x] `internal/usecase/curate_test.go` 追加（正常系・オプション伝達・エラー系の3ケース）
- [x] `fetch_bookmarked_articles` にブックマーク上限（`maxCurateBookmarks = 50`）を追加
