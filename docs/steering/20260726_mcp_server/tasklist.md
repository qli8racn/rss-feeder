# タスクリスト：MCPサーバー化

## docs/steering

- [x] `docs/steering/20260726_mcp_server/requirements.md` 作成
- [x] `docs/steering/20260726_mcp_server/design.md` 作成

## SDK選定

- [x] `github.com/modelcontextprotocol/go-sdk` と `github.com/mark3labs/mcp-go` の最新状況（メンテナンス状況・stdio対応・APIの安定性）を実装着手時に再確認する
- [x] 採用SDKを確定し、`go.mod` に依存追加する（`github.com/modelcontextprotocol/go-sdk` v1.6.1。プロジェクトの Go を 1.25 系へ更新した上で最新版を採用。経緯は design.md 参照）

## エントリポイント（cmd/mcp）

- [x] `cmd/mcp/main.go` を新規作成し、DI コンテナ（`samber/do/v2`）で既存 driver・usecase を配線する
- [x] `config.SetupAnthropicAPIKey()` の呼び出し要否を、公開ツールの確定内容に応じて判断する（enrich/preference が Anthropic API を使うため呼び出す）
- [x] `go build -o bin/mcp ./cmd/mcp` が通ることを確認する

## 公開ツールの実装

- [x] `list`（記事一覧）を MCPツールとして実装する
- [x] `search`（記事検索）を MCPツールとして実装する
- [x] `categories`（カテゴリ一覧）を MCPツールとして実装する
- [x] `list-feeds`（フィード一覧）を MCPツールとして実装する
- [x] `bookmark`（ブックマーク登録）を MCPツールとして実装する
- [x] `add-feed`（フィード登録）を MCPツールとして実装する
- [x] `fetch`（フィード取得）を MCPツールとして実装する
- [x] `remove-feed` を MCPツールとして実装し、破壊的操作である旨をdescriptionに明記した上で `confirm`必須引数（同意なしのtrueを禁止する旨をdescriptionに明記）を実装する
- [x] `backfill-metadata` は見送り（フェーズ1では公開しない）
- [x] `enrich` を MCPツールとして実装し、課金発生の明記・`confirm`必須引数・処理件数上限（`limit`）の必須化を行う
- [x] `preference` を MCPツールとして実装し、課金発生の明記・`confirm`必須引数を実装する

## ログ設計

- [x] `cmd/mcp/main.go` 起動時に `cfg.Log.Output == "stdout"` を検知した場合のバリデーション（起動中断 or stderrへのフォールバック）を実装する
- [x] stdio 通信が汚染されないことを手動確認する（`bin/mcp` に initialize/tools・listのJSON-RPCを標準入力で流し、標準出力にJSON-RPC応答のみが出力され標準エラー出力が空であることを確認済み）

## 動作確認

- [ ] `claude_desktop_config.json` に `bin/mcp` を登録し、Claude Desktop からツール一覧が取得できることを確認する（実機のClaude Desktopでの確認はユーザー側で実施。開発環境ではstdio JSON-RPCを直接流してツール一覧取得・スキーマを確認済み）
- [ ] 各公開ツールを Claude Desktop から呼び出し、既存DB（`rss-feeder-db/reader.db`）への読み書き結果が CLI/Web UI と整合することを確認する
- [ ] CLI/Web UIとMCPサーバーを同時に操作し、WALモード・busy_timeoutによる同時実行時の挙動に問題がないことを確認する
- [x] Claude Desktop 実機登録で発覚した「任意のCWDから起動されるとCWD前提の相対パス（DB・config.yml）解決に失敗しクラッシュする」不具合を `cmd/mcp/main.go` の起動時 `os.Chdir` で修正する（詳細は design.md 参照）

## ドキュメント整備

- [x] `AGENTS.md` に `cmd/mcp` のビルド手順・`bin/mcp` の起動方法を追記する
- [x] Claude Desktop への登録手順（`claude_desktop_config.json` の設定例）をドキュメント化する
