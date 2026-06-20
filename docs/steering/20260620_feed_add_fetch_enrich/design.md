# 設計：フィード追加時の自動取得・要約

## 処理フロー

```
add-feed <url>（cmd/rss-feeder）
  └─ handler/cli/add_feed.go
       ├─ usecase/resolve_feed_url.go ResolveFeedURLUsecase.Execute（既存・変更なし）
       ├─ usecase/add_feed.go AddFeedUsecase.Execute（既存・変更なし）
       ├─ usecase/fetch.go FetchUsecase.Execute(ctx, []string{resolvedURL})（新規呼び出し。usecase自体は既存）
       │    失敗時 → 標準エラー出力に警告、処理は続行（フィード登録の成否には影響しない）
       └─ 新規保存記事（FetchResult.TotalSaved()）が1件以上あれば:
            usecase/trigger_enrich.go TriggerEnrichUsecase.Execute(ctx, resolvedURL, 保存件数)
              （内部で enrichTimeout=30秒の context.WithTimeout を設定。stepThreeTimeout と同じ考え方）
              └─ feedenrich.Agent.Enrich(ctx, feedURL, limit)（サブプロセス実装。下記参照）
                   └─ exec.CommandContext(ctx, rssAgentPath, "enrich", "--feed", feedURL, "--force", "--limit", strconv.Itoa(limit))
                        └─ bin/rss-agent enrich --feed <url> --force --limit <n>（別プロセス。cmd/agent はAnthropic SDKに直接依存）
            失敗時（タイムアウト含む）→ 標準エラー出力に警告、処理は続行

POST /api/feeds（cmd/web）
  └─ handler/web/feed.go AddFeedHandler
       ├─ ResolveFeedURLUsecase.Execute（既存・変更なし）
       ├─ AddFeedUsecase.Execute（既存・変更なし）
       └─ usecase/fetch.go FetchUsecase.Execute(ctx, []string{resolvedURL})（新規呼び出し）
            失敗時 → log.Printf で記録、処理は続行（レスポンスは既存通り 201 + Feed DTO）
            ※ enrich は cmd/web では実行しない（アーキテクチャ上の制約。requirements.md 参照）
```

`ResolveFeedURLUsecase.Execute` 呼び出し時に設定された `ctx`（Web側は `addFeedTimeout`=80秒の
`context.WithTimeout`、CLI側はタイムアウト無し）をそのまま fetch・enrich にも引き継ぐ。

## 新規ファイル

| ファイル | 役割 |
|---------|------|
| `internal/adapter/driver/feedenrich/feedenrich.go` | `Agent` interface（`Enrich(ctx, feedURL string, limit int) error`）、`ErrAgentUnavailable`（`feeddiscovery` と同じ命名パターン） |
| `internal/driver/feedenrich/subprocess.go` | `bin/rss-agent enrich --feed <url> --force --limit <n>` をサブプロセス実行する実装。`cmd/rss-feeder` の単発実行のみが対象のため、同時実行数制限（セマフォ）は持たない |
| `internal/driver/feedenrich/subprocess_test.go` | 統合テスト（`feeddiscovery/subprocess_test.go` と同パターン。正常系・異常終了・バイナリ未検出） |
| `internal/usecase/trigger_enrich.go` | `TriggerEnrichUsecase`（`feedenrich.Agent` への薄いラッパー） |

## 変更ファイル

| ファイル | 変更内容 |
|---------|---------|
| `internal/adapter/driver/anthropic/enrich.go` | `EnrichOptions` に `FeedURL string` フィールドを追加（既存の `Limit`・`Force` はそのまま） |
| `internal/driver/anthropic/enrich.go` | `Run` の `Force` 分岐で `a.repo.FetchLatest(ctx, opts.Limit, opts.FeedURL)` を呼ぶ（従来はハードコードされた `""` だった部分を `opts.FeedURL` に変更。`FeedURL` 未指定時は `""` のままなので既存の `enrich --force` の動作は変わらない） |
| `internal/adapter/handler/agent/enrich.go` | `NewEnrichCommand` に `--feed` フラグを追加し、`EnrichOptions.FeedURL` に渡す（省略時は `""` で従来通り全フィード対象） |
| `internal/adapter/handler/cli/add_feed.go` | `NewAddFeedCommand` に `fetchUC *usecase.FetchUsecase` ・ `triggerEnrichUC *usecase.TriggerEnrichUsecase` を追加し、登録成功後に fetch→（保存件数>0なら）enrichトリガーを呼ぶ |
| `internal/adapter/handler/web/feed.go` | `AddFeedHandler` に `fetchUC *usecase.FetchUsecase` を追加し、登録成功後に fetch を呼ぶ（enrichは呼ばない） |
| `cmd/rss-feeder/main.go` | `feeddiscovery.Agent`・`feedenrich.Agent` をどちらもサブプロセス実装で DI 登録（`--rss-agent-path` フラグを追加）。Anthropic SDK 関連の import は撤去（`docs/steering/20260619_feed_url_discovery/` のアーキテクチャ変更箇所を参照）。`fetchUC`・`triggerEnrichUC` を `NewAddFeedCommand` に渡す |
| `cmd/web/main.go` | `AddFeedHandler` の呼び出しに既存の `fetchUC`（同ファイルに既に生成済み）を追加で渡す |

## インターフェース定義（差分）

```go
// internal/adapter/driver/feedenrich/feedenrich.go
package feedenrich

import "errors"

var ErrAgentUnavailable = errors.New("rss-agent is not available")

type Agent interface {
    // Enrich は指定フィードの最新記事を対象に要約・カテゴライズをトリガーする。
    // バイナリが存在しない・実行できない場合は ErrAgentUnavailable を返す。
    Enrich(ctx context.Context, feedURL string, limit int) error
}
```

```go
// internal/driver/feedenrich/subprocess.go
package feedenrich

type subprocessAgent struct {
    binPath string
}

func NewSubprocessAgent(binPath string) adapterfeedenrich.Agent {
    return &subprocessAgent{binPath: binPath}
}

func (a *subprocessAgent) Enrich(ctx context.Context, feedURL string, limit int) error {
    cmd := exec.CommandContext(ctx, a.binPath, "enrich", "--feed", feedURL, "--force", "--limit", strconv.Itoa(limit))
    var stderr bytes.Buffer
    cmd.Stderr = &stderr
    if err := cmd.Run(); err != nil {
        if errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
            return adapterfeedenrich.ErrAgentUnavailable
        }
        return fmt.Errorf("rss-agent enrich に失敗しました: %s: %w", strings.TrimSpace(stderr.String()), err)
    }
    return nil
}
```

```go
// internal/adapter/driver/anthropic/enrich.go
package anthropic

type EnrichOptions struct {
    Limit   int
    Force   bool
    FeedURL string // 指定時はそのフィードのみを対象にする（空文字列なら全フィード対象、既存動作）
}
```

```go
// internal/driver/anthropic/enrich.go の Run 内、Force分岐
if opts.Force {
    articles, err = a.repo.FetchLatest(ctx, opts.Limit, opts.FeedURL)
} else {
    articles, err = a.repo.FindWithoutSummary(ctx, opts.Limit)
}
```

```go
// internal/usecase/trigger_enrich.go
package usecase

// enrichTimeout はサブプロセス呼び出し（Claude API呼び出しを含む）が無期限にハングしないための上限。
// internal/usecase/resolve_feed_url.go の stepThreeTimeout（discover-feed 用）と同じ 30秒を採用する。
const enrichTimeout = 30 * time.Second

type TriggerEnrichUsecase struct {
    agent adapterfeedenrich.Agent
}

func NewTriggerEnrichUsecase(agent adapterfeedenrich.Agent) *TriggerEnrichUsecase {
    return &TriggerEnrichUsecase{agent: agent}
}

func (uc *TriggerEnrichUsecase) Execute(ctx context.Context, feedURL string, limit int) error {
    ctx, cancel := context.WithTimeout(ctx, enrichTimeout)
    defer cancel()
    return uc.agent.Enrich(ctx, feedURL, limit)
}
```

## ハンドラ実装イメージ

```go
// internal/adapter/handler/cli/add_feed.go
func NewAddFeedCommand(
    addFeedUC *usecase.AddFeedUsecase,
    resolveFeedURLUC *usecase.ResolveFeedURLUsecase,
    fetchUC *usecase.FetchUsecase,
    triggerEnrichUC *usecase.TriggerEnrichUsecase,
) *cobra.Command {
    return &cobra.Command{
        Use:   "add-feed <url>",
        Short: "RSS フィード URL を DB に登録する",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            resolvedURL, err := resolveFeedURLUC.Execute(cmd.Context(), args[0])
            if err != nil {
                return err
            }
            if _, err := addFeedUC.Execute(cmd.Context(), resolvedURL); err != nil {
                return err
            }
            fmt.Printf("登録しました: %s\n", resolvedURL)

            result, err := fetchUC.Execute(cmd.Context(), []string{resolvedURL})
            if err != nil {
                fmt.Fprintf(os.Stderr, "警告: 記事の取得に失敗しました: %v\n", err)
                return nil
            }
            if saved := result.TotalSaved(); saved > 0 {
                if err := triggerEnrichUC.Execute(cmd.Context(), resolvedURL, saved); err != nil {
                    fmt.Fprintf(os.Stderr, "警告: 要約・カテゴライズに失敗しました: %v\n", err)
                }
            }
            return nil
        },
    }
}
```

```go
// internal/adapter/handler/web/feed.go
func AddFeedHandler(
    addFeedUC *usecase.AddFeedUsecase,
    resolveFeedURLUC *usecase.ResolveFeedURLUsecase,
    fetchUC *usecase.FetchUsecase,
) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // ...（既存のデコード・resolve・addFeedUC呼び出しはそのまま）

        if _, err := fetchUC.Execute(ctx, []string{resolvedURL}); err != nil {
            log.Printf("[add_feed] 記事の取得に失敗しました %s: %v", resolvedURL, err)
        }

        writeJSON(w, http.StatusCreated, toFeedDTO(*feed))
    }
}
```

`cli/add_feed.go` は `fmt.Fprintf(os.Stderr, ...)`、`web/feed.go` は `log.Printf` と、それぞれの
ハンドラ層で既存使われている方式（CLIは標準エラー出力、Webサーバーはログ）に合わせる。

## `cmd/rss-feeder/main.go` の DI 構成

`--rss-agent-path`（デフォルト `bin/rss-agent`）フラグを追加し、`feeddiscovery.Agent`・`feedenrich.Agent` の
両方をこのパスから構築する（`cmd/web/main.go` の既存の構成と同じパターン）。

```go
feedDiscoveryAgent := driverfeeddiscovery.NewSubprocessAgent(*rssAgentPath, 1)
feedEnrichAgent := driverfeedenrich.NewSubprocessAgent(*rssAgentPath)
resolveFeedURLUC := usecase.NewResolveFeedURLUsecase(
    do.MustInvoke[adapterrss.RSSReader](i), do.MustInvoke[htmlfetch.Fetcher](i), feedDiscoveryAgent,
)
triggerEnrichUC := usecase.NewTriggerEnrichUsecase(feedEnrichAgent)
```

## テスト戦略

- `internal/driver/feedenrich/subprocess_test.go`: `feeddiscovery/subprocess_test.go` と同じパターン
  （シェルスクリプトのフィクスチャを使った正常系・異常終了・バイナリ未検出のテスト）。同時実行数制限は無いため
  その観点のテストは不要。
- `internal/driver/anthropic/enrich_test.go` に「`Force=true` かつ `FeedURL` 指定時、`FetchLatest` がその `feedURL`
  で呼ばれること」のテストケースを追加する。
- ハンドラ層（`internal/adapter/handler/cli/`・`internal/adapter/handler/web/`）は既存方針通りテスト対象外。
- `FetchUsecase`・`TriggerEnrichUsecase` 自体のロジックは薄いラッパーのため、`TriggerEnrichUsecase` は
  モックエージェントを使った簡単なユニットテストのみで十分。`enrichTimeout` 経過時に `ctx.Err()`（`DeadlineExceeded`）が
  `agent.Enrich` に伝播することを検証するテストケースを `internal/usecase/trigger_enrich_test.go` に含める
  （モックエージェントが `ctx.Done()` を見て即座にエラーを返す形でシミュレートする）。
