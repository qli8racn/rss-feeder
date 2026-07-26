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

## 並行開発（git worktree）

複数の機能を並行して進める場合は、`docs/steering/` のディレクトリ（1機能）ごとに
ブランチ・worktree・Claude Code セッションを1対1で対応させる。

- **1 steering ドキュメント = 1 ブランチ = 1 worktree = 1 Claude Code セッション**
- worktree は `/workspaces/rss-feeder-<slug>/` に作成する。`<slug>` はブランチ名の `/` を `-` に
  置換したもの（例: `fix/cron/poll-feeds` → `rss-feeder-fix-cron-poll-feeds`）。

新規ブランチで着手する場合:

```bash
git worktree add /workspaces/rss-feeder-<slug> -b <branch-name>
```

既存ブランチを引き継ぐ場合は `-b <branch-name>` を省略する。

各 worktree は独立した作業ツリーを持つため、メインの worktree とは別に以下の再セットアップが必要:

- `cd web/frontend && npm ci`（`node_modules` は worktree ごとに独立）
- `cp internal/config/config.yml`（Git管理外のため、メインworktreeから都度コピーするか作り直す）
- DB ファイル（`rss-feeder-db/reader.db`）は初回起動時に自動生成される。worktree間で共有されず
  独立するのは意図した挙動（機能間でデータが混ざらない）。

作業完了・PRマージ後の後片付け:

```bash
git worktree remove /workspaces/rss-feeder-<slug>
./scripts/cleanup-merged-branches.sh --delete
```

`git branch -d`（`cleanup-merged-branches.sh` が内部で使用）は、対象ブランチが他のworktreeで
checkoutされたままだと失敗する。**必ず `git worktree remove` を先に実行してから**
cleanup スクリプトを実行すること。

---

## Setup

> **必須:** 本プロジェクトは Go 1.25 系を要求する（`github.com/modelcontextprotocol/go-sdk` が
> `go >= 1.25.0` を要求するため）。devcontainer を使っている場合、`.devcontainer/Dockerfile` の
> ベースイメージを Go 1.25 系（`mcr.microsoft.com/devcontainers/go:2-1.25-bookworm`）に更新済みだが、
> **既存コンテナには反映されない**。VS Code の「Dev Containers: Rebuild Container」を実行して
> イメージを作り直すこと。`go env GOTOOLCHAIN` が `local` の環境では go.mod が要求する
> `go1.25.x` toolchain を自動取得しないため、Rebuild せずに `go build`/`go test` を実行すると
> `requires go >= 1.25.0` エラーになる（`GOTOOLCHAIN=auto` を一時的に指定すればRebuildせずに
> toolchainを自動ダウンロードして動作確認できる場合がある）。

```bash
go mod tidy
go build -o bin/rss-feeder ./cmd/rss-feeder
go build -o bin/web        ./cmd/web
go build -o bin/rss-agent  ./cmd/agent
go build -o bin/mcp        ./cmd/mcp
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

> **Note:** メモリが少ない環境では `rss-agent`・`mcp`（ともに `internal/driver/anthropic` に依存）の
> ビルドで OOM が発生することがある。devcontainer.json に `--memory=6g --memory-swap=12g` を設定済み
> （OOM 対策）。それでも失敗する場合は並列数を制限する（詳細は `docs/design.md` 参照）：
> ```bash
> GOMAXPROCS=1 GOFLAGS="-gcflags=all=-l=0" go build -p 1 -o bin/rss-agent ./cmd/agent
> GOMAXPROCS=1 GOFLAGS="-gcflags=all=-l=0" go build -p 1 -o bin/mcp       ./cmd/mcp
> ```
>
> **必須:** `internal/driver/anthropic` を使用するビルド・テスト（`rss-agent`・`mcp` 関連）では、必ず上記の環境変数と `-p 1` オプションを指定すること。
> `go build ./...` のような全パッケージ一括ビルドは禁止。対象パッケージを明示すること。
>
> 同様に `go test ./...` も `internal/driver/anthropic` を含む場合 OOM するため、
> `go test $(go list ./... | grep -v internal/driver/anthropic)` のように除外して実行すること。

### DB ドライバの切り替え（SQLite / Supabase）

`internal/config/config.yml` の `db.driver` で使用するDBを選択する（`rss-feeder`・`web`・
`rss-agent`・`mcp` の4エントリポイント共通）。

```yaml
db:
  driver: supabase   # 省略時・"sqlite" の場合は従来通りローカルの rss-feeder-db/reader.db（SQLite）を使用する
  supabase:
    dsn: "postgres://postgres.xxxxxxxx:xxxxxxxx@aws-0-xxxxx.pooler.supabase.com:5432/postgres?sslmode=require"
```

- `db.driver` が未設定 または `"sqlite"` の場合は既存挙動（SQLiteへの読み書き）のまま。
  `"sqlite"`・`"supabase"`・空文字以外の値（タイプミス等）を指定した場合は起動時にエラーになる。
- `db.driver: supabase` を指定する場合は、あらかじめ [supabase.com](https://supabase.com) で
  プロジェクトを作成し、プロジェクト設定画面に表示される接続文字列（Connection string）を
  そのまま `db.supabase.dsn` に貼り付ける。
- Supabaseダッシュボードの接続文字列には **Direct connection**・**Session pooler**（ポート`5432`）・
  **Transaction pooler**（ポート`6543`）の3種類がある。**Session pooler（`5432`）の利用を推奨**する。
  Transaction pooler（`6543`）はpgxのprepared statementキャッシュと衝突する可能性があるため、
  やむを得ず使う場合はDSNに `default_query_exec_mode=simple_protocol` を付与すること
  （`pgx/v5/stdlib` がサポートするクエリ実行モードの1つ）。
- スキーマ（`feeds`・`articles`・`audit_log`）は各エントリポイントの起動時に自動作成される
  （SQLite同様、Supabase側も初回起動時の簡易マイグレーションで初期化される）。
- 手動確認時の注意点: PostgresのTIMESTAMPTZはUTCに正規化して返すのに対し、SQLiteのDATETIMEは
  保存時のロケーションをそのまま引きずるため、日時の表示フォーマットに差が出ることがある。
- Self-hosted Supabase（Docker版）は現時点で未対応（`docs/steering/20260726_supabase_db_driver/`
  の「今後のタスク」参照）。

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

### MCP Server（cmd/mcp）

Claude Desktop 等の MCP クライアントからローカル起動する MCP サーバー。transport は **stdio のみ**。
`internal/usecase` を再利用するアダプタ層で、新しいビジネスロジックは持たない
（設計の詳細は `docs/steering/20260726_mcp_server/design.md` を参照）。

```bash
GOMAXPROCS=1 GOFLAGS="-gcflags=all=-l=0" go build -p 1 -o bin/mcp ./cmd/mcp
bin/mcp [--rss-agent-path bin/rss-agent]
```

公開するツール（他のMCPサーバーとの判別のため `rss_` 接頭辞を付与）: `rss_list`・`rss_search`・
`rss_categories`・`rss_list_feeds`・`rss_bookmark`・`rss_mark_read`・`rss_add_feed`・`rss_fetch`・
`rss_remove_feed`（破壊的操作）・`rss_enrich`・`rss_preference`（ANTHROPIC_API_KEY必須・課金発生）。
`rss_remove_feed`・`rss_enrich`・`rss_preference` は真偽値の必須引数 `confirm` を持ち、`confirm:true` を渡す前に
MCPクライアント（LLM）がユーザーに明示的な同意を得ることを前提とした設計になっている
（ツールの description に明記）。`rss_enrich` はさらに処理件数上限の必須引数 `limit` を持ち、
サーバー側でも `limit`（100）・`batch_size`（40。`internal/driver/anthropic`の
`defaultEnrichBatchSize`と同値で、これを超えるとMaxTokens超過による分割リトライを誘発しやすい）・
`concurrency`（5）を上限にクランプする。
`rss_list`・`rss_search` は `limit`・`page` でページネーションし、応答の `total` で
絞り込み条件に一致する総件数を返す（デフォルト50件・上限200件。CLI・Web UIと異なり閲覧による
既読化は行わない読み取り専用ツールで、既読にしたい場合は `rss_mark_read` を使う）。

> **重要:** stdio transport では標準出力(stdout)がJSON-RPC通信そのものに使われるため、
> `internal/config/config.yml` の `log.output` に `stdout` を指定していると通信が壊れる。
> `cmd/mcp` は起動時にこれを検知してエラー終了する。`stderr`（デフォルト）またはログファイルパスを指定すること。

> **Note:** `cmd/mcp` は他の3つのエントリポイント（`cmd/web`・`cmd/rss-feeder`・`cmd/agent`）と異なり、
> Claude Desktop 等の外部プロセスから任意のCWDで起動される（サブプロセス起動時にリポジトリの作業
> ディレクトリへ `cd` してくれない）。`internal/driver/readerdb`・`internal/config` は
> 「CWD == リポジトリルート」を前提とした相対パスで実装されているため、`cmd/mcp/main.go` は起動直後・
> DIコンテナ配線より前に、実行ファイルの実パス（`os.Executable()` をシンボリックリンク解決したもの）から
> `<repo_root>/bin/mcp` という配置（上記ビルドコマンド参照）を前提にリポジトリルートを算出し、
> `os.Chdir` する。そのため `bin/mcp` を配置場所ごと移動すると相対パス解決が壊れるので、
> ビルドコマンドどおり `<repo_root>/bin/mcp` に配置すること。

Claude Desktop への登録は `claude_desktop_config.json`（macOSでは
`~/Library/Application Support/Claude/claude_desktop_config.json`）に以下のように追記する。

```json
{
  "mcpServers": {
    "rss-feeder": {
      "command": "/absolute/path/to/rss-feeder/bin/mcp",
      "args": ["--rss-agent-path", "/absolute/path/to/rss-feeder/bin/rss-agent"]
    }
  }
}
```

`command`/`args` は絶対パスで指定する（Claude Desktop はリポジトリの作業ディレクトリを引き継がないため）。
登録後 Claude Desktop を再起動すると、ツール一覧に `rss-feeder` サーバーの各ツールが表示される。

> **DevContainer上でビルドした場合の注意:** Claude Desktop はホストOS上でネイティブに動くアプリのため、
> 上記の `command` にDevContainer内でビルドした `bin/mcp` のパスをそのまま指定すると、ホストとコンテナの
> OS/アーキテクチャが異なる場合（例: ホストがmacOSで、DevContainerがPodman machine等のLinux VM経由の
> 場合）に `cannot execute binary file` で起動できない。この場合はホストにClaude Desktopを起動したまま
> `docker exec` でコンテナ内のバイナリにstdioをブリッジする。
>
> ```json
> {
>   "mcpServers": {
>     "rss-feeder": {
>       "command": "/absolute/path/to/docker",
>       "args": ["exec", "-i", "<devcontainerのコンテナ名>", "/workspaces/rss-feeder/bin/mcp",
>                "--rss-agent-path", "bin/rss-agent"]
>     }
>   }
> }
> ```
>
> `-t`（擬似TTY割り当て）は付けない（stdio JSON-RPCのフレーミングを壊す可能性があるため）。
> コンテナ名は `docker ps` で確認する（DevContainerのフルリビルド後は変わりうるので都度確認する）。
> この方式は DevContainer（コンテナ）が起動している間しか動かないため、使う前に VS Code 側で
> ワークスペースを開いてコンテナが起動していることを確認すること。

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
sqlite3 rss-feeder-db/reader.db "SELECT * FROM articles LIMIT 5"
sqlite3 rss-feeder-db/reader.db "SELECT * FROM audit_log ORDER BY timestamp DESC LIMIT 10"
```
