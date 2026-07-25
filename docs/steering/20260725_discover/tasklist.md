# タスクリスト：フィード推薦（discover）

## docs/steering

- [x] `docs/steering/20260725_discover/requirements.md` 作成
- [x] `docs/steering/20260725_discover/design.md` 作成
- [x] `docs/steering/20260725_discover/tasklist.md` 作成

## 実装

- [x] `internal/adapter/driver/anthropic/discover.go` — `DiscoverAgent` インターフェース定義
- [x] `internal/driver/anthropic/discover.go` — `discoverAgent` 実装（ツール使用エージェントループ）
- [x] `internal/usecase/discover.go` — `DiscoverUsecase`
- [x] `internal/usecase/discover_test.go` — `DiscoverUsecase` のテスト
- [x] `internal/adapter/handler/agent/discover.go` — cobra CLI ハンドラ
- [x] `cmd/agent/main.go` — feed repo DI 登録・`NewDiscoverAgent` 登録・`NewDiscoverCommand` 追加

## テスト・確認

- [x] `go build -p 1 -o bin/rss-agent ./cmd/agent` でビルドが通ること
- [x] `bin/rss-agent discover` を実行し、推薦フィードリストが出力されること
