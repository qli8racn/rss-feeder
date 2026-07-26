# 要件：MCPサーバー化

## 背景・目的

現在 rss-feeder の機能（フィード管理・記事取得・AI要約など）は `bin/rss-feeder`（CLI）・`bin/web`（HTTP API）・`bin/rss-agent`（AIエージェント）の3系統から利用できるが、Claude Desktop から直接呼び出す手段がない。

ユーザーは Claude Desktop の会話の中から rss-feeder の機能を実行したいと考えている。MCP（Model Context Protocol）サーバーとして機能を公開することで、Claude Desktop がツールとして直接呼び出せるようにする。

## 要件

- rss-feeder の機能の一部を MCPツールとして公開する新規バイナリ（`cmd/mcp`）を追加する
- transport は **stdio** のみサポートする（Claude Desktop がサブプロセスとして起動し、標準入出力でJSON-RPC通信する方式）
- Claude Desktop の `claude_desktop_config.json` にバイナリパスを登録することで利用できる
- 既存の `internal/usecase` 層のユースケースを再利用し、MCPツール専用のビジネスロジックは持たない（`cmd/web/main.go` の DI 配線パターンを踏襲する）
- 公開するツールは既存 CLI/API 機能の中から選定する（対象・除外の判断は design.md に記載）
- 既存の `rss-feeder-db/reader.db` を Web UI・CLI と共有し、同一DBファイルに対して読み書きする
- ログは既存の構造化ログ基盤（`log/slog`）と整合させ、stdio transport の通信を破壊しない出力先にする

## 制約・前提

- Go バージョン: 1.24.3（既存プロジェクトと同一）
- DI コンテナは既存同様 `samber/do/v2` を使用する
- Go向けMCP SDKは未選定。選定比較は design.md に記載する
- `enrich` や `rss-agent` 系（summarize・preference・discover等）は `ANTHROPIC_API_KEY` を要し、実行時間・課金の観点で注意が必要なため、公開の要否は design.md で個別に検討する
- `remove-feed`（破壊的操作）・`enrich`・`preference`（課金発生）はフェーズ1で公開するが、実行前にユーザーへの事前ヒアリング（同意確認）を必須とする（`confirm`必須引数。設計は design.md 参照）

## スコープ外

- **リモートMCPコネクタ対応（フェーズ2）**: スマホの Claude アプリ等、外出先からリモート経由で利用する方式は今回のスコープ外とする。この方式ではリクエストが Anthropic 側インフラのIPから届くため、単純なIP allowlistでは意図したアクセス制御にならない。デバイス証明書・OAuth等の認証手段の検討が必要であり、将来の課題として design.md に記録する
- stdio transport 以外の transport（HTTP/SSE等）のサポート
- MCPツール経由でのフィード自動登録・自動巡回等、既存にない新機能の追加
- IPアドレスによるアクセス制御（stdio はネットワークを経由しないため、フェーズ1では不要と判断する）

## 完了条件

- `cmd/mcp` ディレクトリに MCPサーバーのエントリポイントが実装され、`go build -o bin/mcp ./cmd/mcp` が通る
- Claude Desktop の `claude_desktop_config.json` にバイナリを登録し、stdio 経由でツール一覧が取得できる
- 選定したツール（design.md で確定）が MCPツールとして呼び出し可能で、既存DBに対する読み書き結果が CLI/Web UI から見て整合する
- ログ出力が stdout（プロトコル通信用）を汚染しないことを確認できる
