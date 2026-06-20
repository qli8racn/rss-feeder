# 設計：フィードURL自動探索

> **2026-06-20 改訂**: ステップ3（Claude フォールバック）の実装方式を、`cmd/web`・`cmd/rss-feeder` どちらも
> サブプロセス経由に統一した（当初は `cmd/rss-feeder` のみインプロセス実装だったが、Anthropic SDK への依存を
> `cmd/agent` に一本化する方針に変更したため）。以下は改訂後の内容。

## 処理フロー

ステップ3（Claude フォールバック）は `cmd/web`・`cmd/rss-feeder` のどちらも同じ実装（サブプロセス経由）を使う。
`ResolveFeedURLUsecase` は依存する `feeddiscovery.Agent` を抽象として受け取るため、バイナリ間で usecase 自体の
コードは完全に共通。

```
add-feed <url> / POST /api/feeds  body: { "feed_url": "<url>" }
  └─ handler/cli/add_feed.go・handler/web/feed.go
       ├─ usecase/resolve_feed_url.go（新規）ResolveFeedURLUsecase.Execute(ctx, url)
       │    ├─ [1] adapterrss.RSSReader.Fetch(ctx, url)
       │    │      成功 → そのまま url を返す
       │    ├─ [2] htmlfetch.Fetcher.Fetch(ctx, url) → HTML取得
       │    │      └─ findFeedLink(html, url)（<link rel="alternate"> 抽出・相対URL解決、純粋関数）
       │    │           候補あり → RSSReader.Fetch で再検証 → 成功すれば採用
       │    └─ [3] feeddiscovery.Agent.Discover(ctx, url)（cmd/web・cmd/rss-feeder共通のサブプロセス実装。下記参照）
       │           解決できなければ ErrFeedNotFound
       └─ usecase/add_feed.go AddFeedUsecase.Execute(ctx, resolvedURL)（既存・変更なし）
```

### ステップ3の実装方式

```
cmd/web ・ cmd/rss-feeder（共通。サブプロセス経由。Anthropic SDKに直接依存しない）
  feeddiscovery.Agent = subprocessAgent（internal/driver/feeddiscovery/subprocess.go）
    └─ セマフォで同時実行数を制限 → exec.CommandContext(ctx, rssAgentPath, "discover-feed", url)
         └─ bin/rss-agent discover-feed <url>（別プロセス。cmd/agent はAnthropic SDKに直接依存）
              ├─ htmlfetch.Fetcher.Fetch(ctx, url) → HTML取得（<head>抽出・truncate）
              ├─ adapteranthropic.FeedDiscoveryAgent.Discover(ctx, url, head)
              │    └─ Claude (claude-haiku-4-5) にURL推測を依頼
              └─ 候補URLを adapterrss.RSSReader.Fetch で再検証
                   成功 → stdoutにURLを出力・終了コード0
                   失敗 → stderrにメッセージ・終了コード1
```

`internal/usecase/discover_feed.go`（`DiscoverFeedUsecase`）・`internal/adapter/driver/anthropic/discover_feed.go`・
`internal/driver/anthropic/discover_feed.go` は **`cmd/agent` 専用**（Anthropic SDK に直接依存する箇所をこのバイナリに
閉じ込めるため）。`cmd/web`・`cmd/rss-feeder` からはこれらのパッケージを import しない。

## 新規ファイル

### 共通（`cmd/web` ・ `cmd/rss-feeder` ・ `cmd/agent` から利用）

| ファイル | 役割 |
|---------|------|
| `internal/adapter/driver/htmlfetch/htmlfetch.go` | `Fetcher` interface（`Fetch(ctx, url) (html string, err error)`） |
| `internal/driver/htmlfetch/htmlfetch.go` | `net/http` での HTML 取得実装。タイムアウト15秒。`Content-Type` レスポンスヘッダが `text/html` で**始まらない**場合はエラーとする（`charset=utf-8` 等のパラメータが付くのが実運用上ほぼ必須のため、完全一致ではなく `strings.HasPrefix` で判定する） |

### `cmd/agent` 専用（Claude を直接呼ぶ。Anthropic SDK に直接依存）

| ファイル | 役割 |
|---------|------|
| `internal/adapter/driver/anthropic/discover_feed.go` | `FeedDiscoveryAgent` interface（`Discover(ctx, url, html string) (feedURL string, err error)`） |
| `internal/driver/anthropic/discover_feed.go` | Claude (`claude-haiku-4-5`) 実装。`<head>` 抽出・truncate、レスポンスから URL 抽出 |
| `internal/usecase/discover_feed.go` | `DiscoverFeedUsecase`（HTML取得 → Claude問い合わせ → `RSSReader.Fetch` で再検証、の薄いオーケストレーション）。`cmd/agent` のみから import される |
| `internal/usecase/discover_feed_test.go` | モックベースのユニットテスト |
| `internal/adapter/handler/agent/discover_feed.go` | `discover-feed <url>` cobra コマンド。成功時 stdout にURLのみ出力、失敗時 stderr + exit 1 |

### `cmd/web` ・ `cmd/rss-feeder` 共通（サブプロセス経由。Anthropic SDK に依存しない）

| ファイル | 役割 |
|---------|------|
| `internal/adapter/driver/feeddiscovery/feeddiscovery.go` | `Agent` interface（`Discover(ctx, url) (feedURL string, err error)`）、`ErrAgentUnavailable` |
| `internal/driver/feeddiscovery/subprocess.go` | `bin/rss-agent discover-feed <url>` をサブプロセス実行する実装。同時実行数を制限するセマフォを内包する（「同時実行数の制御」節参照）。バイナリ未検出時は `ErrAgentUnavailable` を返す。`cmd/web`・`cmd/rss-feeder` の両方から利用する |
| `internal/driver/feeddiscovery/subprocess_test.go` | 統合テスト |

### `cmd/web` ・ `cmd/rss-feeder` 共通（探索オーケストレーション）

| ファイル | 役割 |
|---------|------|
| `internal/usecase/resolve_feed_url.go` | `ResolveFeedURLUsecase`（探索フロー1→2→3のオーケストレーション、`findFeedLink` 純粋関数を含む）。`feeddiscovery.Agent` を抽象として受け取るため、`cmd/web`・`cmd/rss-feeder` どちらの実装が注入されても変更不要 |
| `internal/usecase/resolve_feed_url_test.go` | モックベースのユニットテスト（1で成功・2で成功・3で成功・全滅） |

## 変更ファイル

| ファイル | 変更内容 |
|---------|---------|
| `internal/adapter/handler/cli/add_feed.go` | `ResolveFeedURLUsecase.Execute` を呼んで解決した URL を `AddFeedUsecase.Execute` に渡す |
| `internal/adapter/handler/web/feed.go` | 同上。`ErrFeedNotFound` を 400 に、タイムアウトを 504 にマッピング |
| `cmd/web/main.go` | `htmlfetch.Fetcher` ・ `feeddiscovery.Agent`（サブプロセス実装、同時実行数の上限値を渡す）・ `ResolveFeedURLUsecase` を DI 登録。`--rss-agent-path`（デフォルト `bin/rss-agent`）・`--feed-discovery-concurrency`（デフォルト2）フラグを追加。`AddFeedHandler` 呼び出し箇所で `context.WithTimeout` を設定 |
| `cmd/rss-feeder/main.go` | `htmlfetch.Fetcher` ・ `feeddiscovery.Agent`（サブプロセス実装。`cmd/web` と同じ `NewSubprocessAgent` を同時実行数1で利用）・ `ResolveFeedURLUsecase` を DI 登録。`--rss-agent-path`（デフォルト `bin/rss-agent`）フラグを追加。Anthropic SDK 関連の import（`adapteranthropic`・`driveranthropic`・`config.SetupAnthropicAPIKey()`・専用アダプタ型）は撤去 |
| `cmd/agent/main.go` | `adapteranthropic.FeedDiscoveryAgent` ・ `DiscoverFeedUsecase` を DI 登録、`discover-feed` コマンドを追加 |
| `docs/design.md` | コマンド一覧・パッケージ構成・依存ライブラリ表を更新 |
| `go.mod` | `golang.org/x/net` を `// indirect` から直接依存へ変更（`golang.org/x/net/html` を `<link>` 抽出に使用。バージョンは既存の `v0.50.0` のまま、新規モジュール追加ではない） |

## インターフェース定義

```go
// internal/adapter/driver/htmlfetch/htmlfetch.go
package htmlfetch

type Fetcher interface {
    Fetch(ctx context.Context, url string) (html string, err error)
}

// internal/adapter/driver/feeddiscovery/feeddiscovery.go
package feeddiscovery

import "errors"

var ErrAgentUnavailable = errors.New("rss-agent is not available")

type Agent interface {
    // Discover はフィードURLを推測する。cmd/web・cmd/rss-feeder とも同じサブプロセス実装を使う。
    // バイナリが存在しない・実行できない場合は ErrAgentUnavailable を返す。
    Discover(ctx context.Context, url string) (feedURL string, err error)
}

// internal/adapter/driver/anthropic/discover_feed.go
package anthropic

type FeedDiscoveryAgent interface {
    Discover(ctx context.Context, url string, html string) (feedURL string, err error)
}
```

```go
// internal/driver/feeddiscovery/subprocess.go（cmd/web ・ cmd/rss-feeder 共通）
package feeddiscovery

type subprocessAgent struct {
    binPath string
    sem     chan struct{} // 同時実行数の上限を表すセマフォ。capacity = 上限値
}

func NewSubprocessAgent(binPath string, maxConcurrency int) feeddiscovery.Agent {
    return &subprocessAgent{binPath: binPath, sem: make(chan struct{}, maxConcurrency)}
}

func (a *subprocessAgent) Discover(ctx context.Context, url string) (string, error) {
    select {
    case a.sem <- struct{}{}:
        defer func() { <-a.sem }()
    case <-ctx.Done():
        // 上限に達したまま空きが出る前にcontextがタイムアウト/キャンセルされた
        return "", ctx.Err()
    }

    cmd := exec.CommandContext(ctx, a.binPath, "discover-feed", url)
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    if err := cmd.Run(); err != nil {
        if errors.Is(err, exec.ErrNotFound) || os.IsNotExist(err) {
            return "", ErrAgentUnavailable
        }
        return "", fmt.Errorf("rss-agent discover-feed failed: %s: %w", strings.TrimSpace(stderr.String()), err)
    }
    return strings.TrimSpace(stdout.String()), nil
}
```

```go
// internal/usecase/resolve_feed_url.go
package usecase

var ErrFeedNotFound = errors.New("feed url could not be resolved")

type ResolveFeedURLUsecase struct {
    rssReader   adapterrss.RSSReader
    htmlFetcher htmlfetch.Fetcher
    agent       feeddiscovery.Agent // nil 不可
}

func (uc *ResolveFeedURLUsecase) Execute(ctx context.Context, inputURL string) (string, error) {
    if _, err := url.ParseRequestURI(inputURL); err != nil {
        return "", fmt.Errorf("URLの形式が正しくありません: %w", err)
    }

    if _, _, err := uc.rssReader.Fetch(ctx, inputURL); err == nil {
        return inputURL, nil
    }

    if html, err := uc.htmlFetcher.Fetch(ctx, inputURL); err == nil {
        if candidate, ok := findFeedLink(html, inputURL); ok {
            if _, _, err := uc.rssReader.Fetch(ctx, candidate); err == nil {
                return candidate, nil
            }
        }
    }

    // ステップ3用に独立したタイムアウトを設定する（「タイムアウト方針」節参照）。
    // 親 ctx（handler が設定した全体タイムアウト）の残り予算を超えない範囲でネストする。
    stepCtx, cancel := context.WithTimeout(ctx, stepThreeTimeout)
    defer cancel()
    if candidate, err := uc.agent.Discover(stepCtx, inputURL); err == nil && candidate != "" {
        return candidate, nil
    }
    // agent.Discover が ErrAgentUnavailable / タイムアウト / その他のエラーを返した場合も
    // すべて「ステップ3失敗」として扱い、全滅（ErrFeedNotFound）に集約する。

    return "", ErrFeedNotFound
}

// findFeedLink は HTML から <link rel="alternate" type="application/rss+xml|application/atom+xml">
// を抽出し、baseURL を基準に相対URLを解決する。golang.org/x/net/html を使用。外部I/Oを持たない純粋関数。
func findFeedLink(html, baseURL string) (string, bool)
```

> 実装時の注意: 上記の `Execute` 内ではステップ1・2・3の失敗を握り潰さず、サーバーログ用に
> `log.Printf` 等で原因（どのステップで何が失敗したか）を残すこと。クライアントへの応答は
> `ErrFeedNotFound` の単一メッセージのままでよいが、運用時に「なぜ見つからなかったか」を
> 追跡できるようにする。

## 同時実行数の制御

- `cmd/web`: `subprocessAgent` がセマフォ（capacity = `--feed-discovery-concurrency`、デフォルト 2）でステップ3の同時実行数を制限する
- 上限に達している間に来たリクエストは、空きが出るまで `Discover` 内でブロックする（待機時間は後述のステップ3用タイムアウトの範囲内）
- ステップ1・2（HTTP取得のみ）は制限しない
- `cmd/rss-feeder`: 同じ `subprocessAgent` を `maxConcurrency=1` で構築する（CLIはユーザーが手動で起動する単発実行のため、
  上限値をフラグ化する必要はない）

## タイムアウト方針

タイムアウトは2階層で構成する（ネスト構造）。

| 階層 | 設定場所 | 値 | 説明 |
|------|---------|-----|------|
| 全体 | `internal/adapter/handler/web/feed.go`（`AddFeedHandler`） | 80秒 | `context.WithTimeout(r.Context(), 80*time.Second)` を `ResolveFeedURLUsecase.Execute` 呼び出し全体に適用する |
| ステップ3個別 | `internal/usecase/resolve_feed_url.go`（`ResolveFeedURLUsecase.Execute` 内部） | 30秒 | 全体タイムアウトの残り予算内で、`uc.agent.Discover` 呼び出しにさらに `context.WithTimeout` を設定する（親が先に切れればそちらが優先される） |

ステップ1（`RSSReader.Fetch` 既存の `fetchTimeout` = 30秒）・ステップ2（`htmlfetch.Fetcher` = 15秒）は
各ドライバが内部で持つ既存・新規のクライアントタイムアウトであり、`ResolveFeedURLUsecase` が個別に設定するものではない。
直列実行のため最悪ケースは概ね 30+15+30 ≒ 75秒で、全体タイムアウト80秒はこれに収まるよう設定した安全マージンである。

超過時は `504` 相当（`writeJSONError(w, http.StatusGatewayTimeout, ...)`、`openapi.yaml` に `504` レスポンスを追加）を返す。
`cmd/rss-feeder`（CLI）側は全体タイムアウトを設定せず、各ステップの既存クライアントタイムアウトのみに従う
（CLIの単発実行であり、サーバーのリクエストハンドリングのような上限設定の必要性が薄いため）。

## Claude プロンプト方針（`internal/driver/anthropic/discover_feed.go`）

- モデル: `claude-haiku-4-5`（`enrich`/`summarize` と同じ、単純抽出タスクのため）
- 入力: URLと、HTMLから抽出した `<head>` 部分（`<head>` が見つからない場合は先頭 `maxHTMLRunes`（20,000 rune）で truncate した全体）
- システムプロンプト: 「HTMLからRSS/AtomフィードへのリンクURLを1つ特定し、URLのみを出力する。見つからない場合は `NONE` と出力する」のような明確な出力フォーマット指定
- レスポンスが `NONE` または有効なURL形式でない場合はエラーとして扱う

## OpenAPI への追加（`docs/openapi.yaml`）

`POST /api/feeds` のレスポンスに以下を追加する。

```yaml
responses:
  "400":
    $ref: '#/components/responses/BadRequest'   # 既存。フィード未解決もここに含める
  "504":
    description: フィードURLの探索がタイムアウトした
    content:
      application/json:
        schema:
          $ref: '#/components/schemas/Error'
```

型再生成は既存と同様 `go generate ./internal/adapter/handler/web/openapi/...` / `npm run generate:api` を実行する
（フロントエンドの型・`api.ts` の手書きエラーハンドリングには変更不要 — `res.ok` チェックの汎用エラーパスでカバーされる）。

## テスト戦略

- `internal/usecase/resolve_feed_url_test.go`: `rssReader` / `htmlFetcher` / `agent` をモック化し、
  「1で成功」「1失敗→2で成功」「1,2失敗→3で成功」「全滅」「agentがErrAgentUnavailableを返した場合も全滅扱い」を検証
- `findFeedLink` 純粋関数: 標準的な `<link>` タグ・相対URL・rel/type順序が異なる場合・該当タグなしのテストケースを
  `internal/usecase/resolve_feed_url_test.go` 内に含める（外部I/Oがないため domain 同様の素のユニットテストで足りる）
- `internal/usecase/discover_feed_test.go`: `FeedDiscoveryAgent` ・ `RSSReader` をモック化し、Claude応答の検証ロジックをテスト
- `internal/driver/htmlfetch/htmlfetch_test.go`: `httptest.NewServer` を使った統合テスト（既存の `driver/rss/reader_test.go` と同パターン）。
  `Content-Type: text/html; charset=utf-8` のようにパラメータ付きヘッダでも成功することを必ず検証する
- `internal/driver/feeddiscovery/subprocess_test.go`: 実際に小さなダミーバイナリ（テスト用シェルスクリプト or `go build` した一時バイナリ）を使い、
  正常系・異常終了・バイナリ未検出（`ErrAgentUnavailable`）・同時実行数の上限を超えた場合に待機すること、を検証
- `internal/adapter/handler/web/feed.go` ・ `internal/adapter/handler/cli/add_feed.go` は既存方針通りテスト対象外（adapter/handler 層）
