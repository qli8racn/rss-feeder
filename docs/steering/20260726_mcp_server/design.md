# 設計：MCPサーバー化

## 全体構成

```
Claude Desktop
  └─ claude_desktop_config.json に登録された bin/mcp をサブプロセスとして起動
       └─ stdio（標準入出力）で JSON-RPC 通信
            └─ cmd/mcp/main.go
                 ├─ DI コンテナ（samber/do/v2）… cmd/web/main.go と同じ配線パターン
                 ├─ MCP SDK server … stdio transport でツール呼び出しを受信
                 └─ tool handler … internal/usecase の各 Usecase.Execute を呼び出す
                      └─ 既存 driver 層（readerdb・rss・htmlfetch・anthropic）
                           └─ rss-feeder-db/reader.db（CLI・Web UI と共有）
```

`cmd/mcp` は新しいビジネスロジックを持たず、MCPプロトコルのアダプタ層に徹する（ツール引数 → usecase 呼び出し → 結果をMCPレスポンス形式に変換、のみを担当する）。

## Go向けMCP SDKの選定

| 観点 | `github.com/modelcontextprotocol/go-sdk`（公式） | `github.com/mark3labs/mcp-go`（コミュニティ） |
|---|---|---|
| メンテナンス主体 | Anthropic + MCP運営が公式に管理 | コミュニティ有志が管理 |
| stdio transport | サポート（公式仕様への追従が早い） | サポート（実績があり利用例も多い） |
| tool定義のしやすさ | 構造体タグベースでスキーマ定義（JSON Schema自動生成の設計） | ビルダー関数（`mcp.NewTool` 等）でスキーマを組み立てる方式 |
| API安定性 | 比較的新しく破壊的変更のリスクがある | 先行して広く使われており実績ベースの安定感がある |

**採用方針**: 公式SDK（`github.com/modelcontextprotocol/go-sdk`）を第一候補とする。プロトコル仕様への追従・長期的なメンテナンス性を重視するため。ただし本比較は既知の情報の範囲によるものであり、両SDKとも開発が活発なため、**実装着手時に最新のドキュメント・リリース状況を確認したうえで最終決定する**（本ドキュメント確定時点ではAPI詳細の検証は行っていない）。

### 実装着手時の最終決定（2026-07-26）

実装時点で両SDKの最新リリースを確認した結果、以下の理由で公式SDK `github.com/modelcontextprotocol/go-sdk` の採用を確定した。

- 公式SDKは v1.0.0 以降 API が安定しており（本調査時点の最新は v1.6.1、MCP仕様 2025-11-25 まで対応）、`mcp.AddTool` による構造体タグベースの入出力スキーマ自動生成・`mcp.StdioTransport` によるstdio対応など、design.md記載の想定機能がそのまま利用できることを確認した。
- コミュニティSDK `github.com/mark3labs/mcp-go`（v0.57.0）も継続的にメンテナンスされているが、公式のプロトコル追従・長期サポートを優先し、当初方針どおり公式SDKを選定した。
- **Go バージョンへの影響**: `github.com/modelcontextprotocol/go-sdk` は v1.5.0 以降 `go >= 1.25.0` を要求する。本プロジェクトは従来 `go 1.24.3`（toolchain `go1.24.5`）で運用していたため、最新版（v1.6.1）を採用する場合はプロジェクト全体のGoバージョンを1.25系へ引き上げる必要がある。
  - 検討した代替案: 直近のGo1.24互換版である `v1.4.0` に固定する案もあったが、ユーザー判断により「Goを1.25に上げて最新の公式SDKを使う」方針を採用した。将来的なSDKアップデート（バグ修正・仕様追従）を継続して受けられることを優先している。
  - この決定に伴い `go.mod` の `go`/`toolchain` ディレクティブを `1.25.0`/`go1.25.12` に更新し、`.devcontainer/Dockerfile` のベースイメージを `mcr.microsoft.com/devcontainers/go:1-1.24-bullseye` から `mcr.microsoft.com/devcontainers/go:2-1.25-bookworm` に変更した（Go1.25系はMCR上でbullseyeベースの提供がないため、OSもbookwormに更新）。**既存のdevcontainerを使っている開発者は「Rebuild Container」が必要**（詳細は `AGENTS.md` 参照）。
  - CI（`.github/workflows/ci-go.yml`）は `actions/setup-go@v5` が `go-version-file: go.mod` を参照する設定のため、追加修正は不要と判断した（go.mod更新により自動的にGo1.25系がセットアップされる）。

## エントリポイントの配置

`cmd/mcp/main.go` を新規追加する。`cmd/web/main.go` の DI 配線パターンを踏襲し、`samber/do/v2` で既存の driver・usecase を組み立てる。

```go
i := do.New()
do.Provide(i, config.NewProvider)
do.Provide(i, driverlogger.NewLogger)
do.Provide(i, readerdb.NewClient)
do.Provide(i, dbrepoarticle.NewRepository)
do.Provide(i, dbrepofeed.NewRepository)
do.Provide(i, driverrss.NewReader)
do.Provide(i, driverhtmlfetch.NewFetcher)
// ...必要な usecase を組み立てたうえで、MCP tool ハンドラに渡す
```

`config.SetupAnthropicAPIKey()`（`cmd/agent/main.go` で使用しているもの）は、`ANTHROPIC_API_KEY` を要するツール（enrich等）を公開する場合にのみ呼び出す。

## 公開するツールの選定

既存機能を「読み取り系」「操作系（副作用あり）」「AI実行系（課金・長時間実行）」に分類し、フェーズ1で公開するかどうかを整理する。

| 既存機能 | 分類 | フェーズ1で公開 | 備考 |
|---|---|---|---|
| `list`（記事一覧） | 読み取り | ○ | ブックマーク・カテゴリでの絞り込みに対応 |
| `search`（記事検索） | 読み取り | ○ | キーワード検索 |
| `categories`（カテゴリ一覧） | 読み取り | ○ | |
| `list-feeds`（フィード一覧） | 読み取り | ○ | |
| `bookmark`（ブックマーク登録） | 操作系 | ○ | 副作用は単一記事への更新のみで影響範囲が小さい |
| `add-feed`（フィード登録） | 操作系 | ○ | フィードURL解決を伴うため実行時間はやや長い |
| `remove-feed`（フィード削除） | 操作系 | ○ | 記事も連動削除される破壊的操作のため、`confirm`必須引数（下記）で明示的同意を必須化する |
| `fetch`（フィード取得） | 操作系 | ○ | ネットワークI/Oのため数秒〜数十秒かかる可能性がある旨をツール説明文に明記する |
| `backfill-metadata` | 操作系 | × | 利用頻度が低いメンテナンス用途のため、フェーズ1では見送り |
| `enrich`（要約・カテゴライズ） | AI実行系 | ○ | `ANTHROPIC_API_KEY` 必須・課金発生・実行時間が長い（複数記事のバッチ処理）。`confirm`必須引数＋処理件数上限の必須化で暴走防止策を設ける |
| `summarize`（rss-agent） | AI実行系 | × | `enrich` と役割が重複するため、フェーズ1では見送り |
| `preference`（rss-agent） | AI実行系 | ○ | 課金は発生するが読み取り専用（DB更新なし）。`confirm`必須引数で明示的同意を必須化する |
| `discover` / `discover-feed`（rss-agent） | AI実行系 | × | 実行時間・課金コストが大きく、フェーズ1の主目的（Claude Desktopからの日常的な記事操作）から外れるため見送り |

**方針**: 読み取り系・軽量な操作系（`list`・`search`・`categories`・`list-feeds`・`bookmark`・`add-feed`・`fetch`）に加え、破壊的操作（`remove-feed`）とAI実行系（`enrich`・`preference`）もフェーズ1で公開する。ただし `remove-feed`・`enrich`・`preference` は「事前ヒアリング」を必須化するため、以下の設計を採用する。

### 確認フロー（`confirm`必須引数）

MCPクライアント（Claude Desktop）自体にも汎用のツール実行確認UIはあるが、それとは別に**サーバー側でも明示的な同意なしに実行できない**よう、対象ツールの入力スキーマに真偽値の必須引数 `confirm` を設ける。

- ツールの `description` に、以下を明記する
  - `remove-feed`: 「この操作はフィードと関連する記事を完全に削除する。実行前に必ずユーザーに削除対象（フィード名・記事件数）を提示し、明示的な同意を得てから `confirm: true` を渡すこと。ユーザーの同意なしに `true` を渡してはならない」
  - `enrich` / `preference`: 「この操作は ANTHROPIC_API_KEY による追加課金が発生する。実行前に必ずユーザーに『ANTHROPIC_API_KEY を使用した追加料金が発生しますが、実行してよいですか？』と確認し、明示的な同意を得てから `confirm: true` を渡すこと。ユーザーの同意なしに `true` を渡してはならない」
- ハンドラ側は `confirm != true` の場合、usecase を呼び出さずにエラー（同意が必要である旨のメッセージ）を返す
- `enrich` はさらに処理件数上限（`limit`）を必須引数にする（無制限実行を避ける）

この設計はモデルの振る舞いに依存する「お願いベース」の安全策であり、悪意のあるプロンプトインジェクション等に対する完全な防御にはならない点に留意する（フェーズ1の利用シーンはユーザー本人がDesktopを直接操作する想定のため、リスクは限定的と判断する）。

## ログ設計

stdio transport では標準出力（stdout）がプロトコル通信そのものに使われるため、ログを誤って stdout に出力するとMCP通信が破壊される。

- `cmd/mcp` では既存の `driverlogger.NewLogger`（`log/slog` ベース、`config.yml` の `log.output`/`log.format` で設定）をそのまま利用する
- ただし `log.output: stdout` が設定された場合に通信を破壊するリスクがあるため、`cmd/mcp/main.go` 起動時に `cfg.Log.Output == "stdout"` を検知したらエラーで起動を中断する（または強制的に `stderr` にフォールバックする）バリデーションを追加する
- デフォルト（`log.output: stderr`）のままであれば安全に動作する。運用上は `stderr` またはファイル出力を推奨する旨を `AGENTS.md`（または `cmd/mcp` 用のREADME）に記載する

## 既存DBとの関係・同時実行

`internal/driver/readerdb/client.go` の DSN は既に `_journal_mode=WAL&_busy_timeout=5000` を設定済みであり、複数プロセス（CLI・Web UI・MCPサーバー）からの同時アクセスに対して以下の耐性がある。

- WALモード: 読み取りが書き込みをブロックしない
- busy_timeout=5000ms: 書き込みロック競合時に即座にエラーとせず、5秒間リトライ待機する

`cmd/mcp` も同じ `readerdb.NewClient` をそのまま利用することで、追加の同時実行対策は不要と判断する。ただし、Claude Desktop からの操作とCLI/Web UIの操作が同時に発生する頻度は低いと見込まれるため、フェーズ1では上記の既存設定に委ね、問題が顕在化した場合に追加対策（リトライ・排他制御）を検討する。

## 動作確認で発覚した不具合の修正（2026-07-26）: CWD前提の相対パスによる起動時クラッシュ

Claude Desktop に `bin/mcp` を登録して実機確認したところ、`migration failed: unable to open database file` で即座にクラッシュした（"Server disconnected"）。

### 原因

以下2箇所が「プロセスのCWD == リポジトリルート」を前提にした相対パスになっており、Claude Desktop は `command` を exec する際にリポジトリルートへ `cd` してくれないため、任意のCWD（例: Claude Desktop自身の作業ディレクトリ）から起動されると解決に失敗する。

- `internal/driver/readerdb/client.go` の `dsn`（`rss-feeder-db/reader.db?...`）
- `internal/config/config.go` の `v.AddConfigPath("internal/config")`

`cmd/web`・`cmd/rss-feeder`・`cmd/agent` は Makefile・`go run` 経由でリポジトリルートから起動される運用のため、これまで顕在化していなかった。`cmd/mcp` だけが外部プロセス（Claude Desktop）から任意のCWDで起動される特殊なエントリポイントであるため、この問題が露呈した。

### 対応方針の検討

| 案 | 内容 | 採用可否 |
|---|---|---|
| A. 設定ファイル側でラッパースクリプトを噛ませる | `claude_desktop_config.json` の `command` に `cd <repo_root> && bin/mcp` 相当のシェルを指定する | 見送り。将来的な他の起動経路（systemdサービス化、docker exec等）でも同じ相対パス前提が再発するリスクを避けたいため、設定ファイル側の回避ではなくコード側で恒久的に直す方針とした |
| B. `internal/driver/readerdb/client.go`・`internal/config/config.go` 自体を絶対パス解決するよう書き換える | 全エントリポイント共通のロジックに手を入れる | 見送り。`cmd/web`・`cmd/rss-feeder`・`cmd/agent` は現状の相対パス運用で問題が起きておらず、無関係な3エントリポイントの挙動を変えるリスクを負うだけの必要性がない |
| C. `cmd/mcp/main.go` 側でDI配線前にリポジトリルートへ `os.Chdir` する（採用） | `os.Executable()` で実行ファイルの実パスを取得し、ビルドコマンド（`go build -o bin/mcp ./cmd/mcp`）が前提とする `<repo_root>/bin/mcp` という配置から2階層上をリポジトリルートとして算出し、起動直後・DIコンテナ配線前に `os.Chdir` する | 採用。`cmd/mcp` だけの特殊事情に閉じた修正であり、`internal/driver/readerdb`・`internal/config` の相対パスや他の3エントリポイントには一切手を入れずに済む。新しい必須フラグ（`--db-path` 等）の追加も不要で、`claude_desktop_config.json` の既存登録内容（絶対パスの `command`/`args`）もそのまま動作する |

### 実装

- `cmd/mcp/main.go` に `chdirToRepoRoot()` を追加し、`flag.Parse()` 直後・`config.SetupAnthropicAPIKey()` やDIコンテナ配線より前に呼び出す
- `filepath.EvalSymlinks` でシンボリックリンクを解決してから2階層上を算出する（`bin/mcp` がシンボリックリンク経由で起動された場合でも正しく解決するため）
- 制約: `bin/mcp` を `<repo_root>/bin/mcp` 以外の場所に配置すると相対パス解決が壊れる。既存のビルド手順（`go build -o bin/mcp ./cmd/mcp`）どおりに配置する運用を前提とする（`AGENTS.md` に注記済み）

## スコープ外（フェーズ2）: リモートMCPコネクタ対応

将来的にスマホの Claude アプリ等、リモートMCPコネクタ経由で外出先から利用したいという要望がある。この場合、以下の理由でフェーズ1と同じ設計を単純に転用できない。

- リモートMCPコネクタはリクエストがユーザーの端末ではなく Anthropic 側インフラのIPから届くため、IPアドレスによるallowlistは「ユーザー本人からのアクセスに限定する」という意図した制御にならない
- 認証手段としてデバイス証明書やOAuth等の導入が必要になる見込みだが、詳細はフェーズ2着手時に別途検討する
- transportもstdioではなくHTTP/SSE等のネットワーク経由になるため、サーバーの常時起動・公開範囲の設計が別途必要になる

本フェーズではこれらの検討・実装は行わない。
