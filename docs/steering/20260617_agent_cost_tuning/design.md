# 設計：エージェント系コマンドの再確認とリファクタリング

## 1. 現状の振り返り（フェーズ開始時点）

| ファイル | モデル | Thinking | 件数上限 | 備考 |
|---|---|---|---|---|
| `enrich.go` | `claude-haiku-4-5` | なし | `opts.Limit`（既定 10）、本文 2000 文字に切り詰め | 単発リクエスト（ツール未使用） |
| `summarize.go` | `claude-haiku-4-5` | なし | `opts.Limit`（既定 10） | `fetch_articles` ツール経由 |
| `preference.go` | `claude-opus-4-8` | `adaptive` | `fetch_bookmarked` の `limit`（既定・上限 50） | 趣向分析という性質上、モデル・Thinking は意図的に維持 |

`summarize`/`enrich` は単純な要約・分類タスクのため Haiku 4.5 に変更済み。`preference` は
ブックマークの傾向を洞察する分析タスクであり、要約系より難易度が高いと判断し Opus 4.8 +
adaptive thinking を維持する。本フェーズではこの判断が妥当か再確認しつつ、可視化（費用・実行時間）
を主目的とする。

## 2. ANTHROPIC_API_KEY の取得元の明記

`README.md` の前提条件セクションと `AGENTS.md` のセットアップ手順に以下を追記する。

```markdown
> `ANTHROPIC_API_KEY` は claude.ai（コンシューマー向けチャット）のログイン情報ではなく、
> Claude Console（https://console.anthropic.com）の「API Keys」から発行する API キーを使用する。
```

`internal/config/config.example.yml` のコメントにも同様の注記を追加する。

## 3. 費用・実行時間の計測

### 3.1 料金テーブル

`internal/driver/anthropic/usage.go`（新規）に、使用するモデルの USD/1M トークン単価を保持する。

```go
package anthropic

// modelPricing は 1M トークンあたりの USD 単価（入力・出力）。
// 価格改定時はここを更新する。
var modelPricing = map[string]struct{ Input, Output float64 }{
    "claude-haiku-4-5": {Input: 1.00, Output: 5.00},
    "claude-sonnet-4-6": {Input: 3.00, Output: 15.00},
    "claude-opus-4-8":   {Input: 5.00, Output: 25.00},
}

// estimateCostUSD は Usage から概算費用（USD）を計算する。
// cache_read は通常入力の約 0.1 倍、cache_creation は約 1.25 倍として概算する。
func estimateCostUSD(model string, u anthropic.Usage) float64 {
    price, ok := modelPricing[model]
    if !ok {
        return 0
    }
    input := float64(u.InputTokens) + float64(u.CacheCreationInputTokens)*1.25 + float64(u.CacheReadInputTokens)*0.1
    output := float64(u.OutputTokens)
    return (input*price.Input + output*price.Output) / 1_000_000
}
```

### 3.2 実行時間計測と出力フォーマット

各エージェントの `Run()` の冒頭で `start := time.Now()` を取り、`defer` で1行のサマリーを
**標準エラー出力**に出す（標準出力は要約結果などコマンドの主たる出力に使うため汚さない）。

```go
defer func() {
    fmt.Fprintf(os.Stderr, "[rss-agent] model=%s input=%d output=%d cost=$%.4f elapsed=%s\n",
        model, usage.InputTokens, usage.OutputTokens, estimateCostUSD(model, usage), time.Since(start).Round(time.Millisecond))
}()
```

- `enrich.go`：単発リクエストのため `resp.Usage` をそのまま使う。
- `summarize.go`/`preference.go`：`runAgentLoop`（`loop.go`）がツール呼び出しで複数回
  `Messages.New` を呼ぶため、ループ内で `Usage` を加算して `runAgentLoop` の戻り値に含める
  必要がある。`runAgentLoop` の戻り値を `(string, anthropic.Usage, error)` に拡張する。

### 3.3 `runAgentLoop` の変更点（概要）

```go
func runAgentLoop(ctx context.Context, client anthropic.Client, params anthropic.MessageNewParams, handler toolHandler) (string, anthropic.Usage, error) {
    var total anthropic.Usage
    messages := params.Messages
    for {
        params.Messages = messages
        resp, err := client.Messages.New(ctx, params)
        if err != nil {
            return "", total, err
        }
        total.InputTokens += resp.Usage.InputTokens
        total.OutputTokens += resp.Usage.OutputTokens
        total.CacheCreationInputTokens += resp.Usage.CacheCreationInputTokens
        total.CacheReadInputTokens += resp.Usage.CacheReadInputTokens
        // ...既存のツール呼び出し処理...
    }
}
```

呼び出し元（`summarize.go`/`preference.go`）はこの加算済み `Usage` を使ってコスト・実行時間を
出力する。

## 4. 表示の既定動作

フラグでの切り替えは行わず、常に標準エラー出力へ1行出すデフォルト動作とする
（CI・cron 実行時のログにも残るようにするため）。標準出力（要約結果など）には影響しない。

## 5. enrich の並列バッチ処理（追加スコープ）

### 5.1 バッチ分割

`enrichBatchSize = 40` を新規定数として追加する（40件で output ≈ 3,400〜3,900トークンと
実測しており、4096トークンの上限に対して安全マージンがある値として採用）。
`Run()` で取得した記事一覧をこの単位でチャンクに分割する。

### 5.2 並列実行

`enrichConcurrency = 4` を新規定数として追加する。チャンクごとに `summarizeAndCategorize`
を呼ぶゴルーチンを、`sync.WaitGroup` とバッファ付きチャネル（セマフォ）で同時実行数を
制限しながら起動する。

> **設計の試行錯誤**：当初は `golang.org/x/sync/errgroup` の `errgroup.Group.SetLimit` を
> 採用したが、各 `g.Go` のクロージャは常に `nil` を返す実装になっており（実エラーは
> `outcomes[i]` 経由で個別に集約するため）、`errgroup` 本来のエラー伝播・キャンセル機能を
> 一切使っていなかった。そのためコードレビューで「`golang.org/x/sync` を直接依存にする
> 理由がない」と指摘され、手動の `sync.WaitGroup`+セマフォに戻した。

同時実行数の上限に達してチャネルへの送信がブロックしている間に、他チャンクが
`ctx` をキャンセルする可能性がある。そのため `ctx.Err()` のチェックは
**ブロッキング送信の後**に行う（送信前にチェックしても、チェック直後に送信がブロックして
いる間にキャンセルされるケースを取り逃してしまうため）。これにより、実行中に `ctx` が
キャンセルされた場合、まだディスパッチしていないチャンク分のAPIコストを節約できる。

```go
const (
    enrichBatchSize   = 40
    enrichConcurrency = 4
)

func chunkArticles(articles []domain.Article, size int) [][]domain.Article { ... }

func (a *enrichAgent) Run(ctx context.Context, opts adapteranthropic.EnrichOptions) (int, error) {
    start := time.Now()
    if err := ctx.Err(); err != nil {
        return 0, err // 既にキャンセル済みのctxなら、APIを1件も呼ばずに即座に返す
    }
    // ...記事取得は現状のまま（FindWithoutSummary / FetchLatest）...

    chunks := chunkArticles(articles, enrichBatchSize)
    type chunkOutcome struct {
        results []enrichResult
        usage   anthropic.Usage
        err     error
    }
    outcomes := make([]chunkOutcome, len(chunks))

    var wg sync.WaitGroup
    sem := make(chan struct{}, enrichConcurrency)
    dispatched := 0
    for i, chunk := range chunks {
        sem <- struct{}{} // 上限に達していればここでブロックする
        if ctx.Err() != nil {
            <-sem
            break // 未ディスパッチのチャンクはここで諦める
        }
        dispatched++
        wg.Add(1)
        go func() {
            defer wg.Done()
            defer func() { <-sem }()
            results, usage, err := a.summarizeAndCategorize(ctx, chunk)
            outcomes[i] = chunkOutcome{results: results, usage: usage, err: err}
        }()
    }
    wg.Wait()
    for i := dispatched; i < len(chunks); i++ {
        outcomes[i] = chunkOutcome{err: fmt.Errorf("チャンク%d: ディスパッチ前にcontextがキャンセルされました: %w", i, ctx.Err())}
    }

    // 5.3（Usage集計）・5.4（DB書き込み）・5.5（エラー集約）は本文参照
}
```

Go 1.22以降はforループ変数がイテレーションごとに新しいインスタンスになるため、
クロージャ内で `i`/`chunk` を直接参照してもキャプチャ事故は起きない
（`go.mod` は `go 1.24.3` を指定）。明示的な引数渡しは不要。

1チャンクの失敗（API エラー・JSON解析エラー）が他チャンクに伝播しないよう、ゴルーチンは
常に `nil` を返さず（手動セマフォ方式のため `g.Go` のような戻り値もない）、各チャンクの
結果（成功時は `[]enrichResult`、失敗時は `error`）を `outcomes[i]` に個別に格納する。

### 5.3 Usage 集計とログ出力

`summarizeAndCategorize` から内部の `defer logUsage(...)` を削除し、
`(results []enrichResult, usage anthropic.Usage, err error)` を返すように変更する。
`Run()` 側で全チャンクの `usage` を `addUsage` で合算し、Run 全体の `time.Since(start)` と
あわせて最後に1回だけ `logUsage` を呼ぶ（チャンクごとに複数行出力しない）。

### 5.4 DB 書き込み：チャンク単位のトランザクション

`mattn/go-sqlite3` を素の DSN（`reader.db`）で開いており、WAL モードや `busy_timeout` の
設定がない（`internal/driver/readerdb/client.go`）。並列ゴルーチンから直接DB更新を呼ぶと
`database is locked`（SQLITE_BUSY）のリスクがあるため、DB 書き込みは全チャンクのAPI呼び出し
完了後、メインゴルーチンでのみ直列に行う（API 呼び出しのみを並列化し、DB アクセスは
並列化しない）。

1記事ごとに `ExecContext` していた既存の `UpdateEnrichment` ループは、チャンク数が増える
ほどジャーナルへのfsync回数が増えるため、複数件をまとめて1トランザクションで更新する
`Repository.UpdateEnrichmentBatch(ctx, []EnrichmentUpdate)` を新設した。

> **設計の試行錯誤**：当初は全チャンクの結果をまとめて `UpdateEnrichmentBatch` を**1回だけ**
> 呼ぶ実装にした。しかしこれは「1チャンクが失敗しても他チャンクの処理・DB保存は継続する」
> という当初の受け入れ条件に反することがコードレビューで判明した：1トランザクションに
> まとめると、1件でも `ExecContext` が失敗すれば**他チャンクの分も含めて全件ロールバック**
> されてしまう。API呼び出しレベルでは丁寧に保たれていた部分成功が、永続化の最後の一歩で
> 失われる構造だった。
>
> そのため `UpdateEnrichmentBatch` は**チャンクごとに1回**呼ぶように変更した。
> トランザクションの単位はチャンク（最大 `enrichBatchSize` 件）に揃え、1チャンクのDB書き込み
> 失敗が他チャンクの保存済み結果を巻き込まないようにした。チャンク内の1件でも失敗すれば
> そのチャンクの分だけロールバックされる（部分コミットはしない）。

DB更新失敗時のエラーには、対象記事のIDを含める（`UpdateEnrichmentBatch` 内で
`記事 %d の更新に失敗しました` と付与する）。これは1トランザクション化によって失われていた
「どの記事の更新が失敗したか」という診断情報を復元するための変更で、呼び出し元
（`enrich.go`）はこのエラーをそのまま伝播するだけでよい。

### 5.5 エラー集約

各チャンクの `err`（API呼び出し由来）と各チャンクのDB書き込み失敗（`UpdateEnrichmentBatch`
由来）を、すべて `errors.Join` で1つのエラーにまとめて返す。成功したチャンクの結果は
（他チャンクがAPI呼び出し・DB書き込みのいずれで失敗していても）保存する（部分成功を
チャンク単位でAPI呼び出し・DB保存の両方の段階を通じて維持する）。

### 5.6 切り詰め検知

`summarizeAndCategorize` のJSON解析失敗時、`resp.StopReason == anthropic.StopReasonMaxTokens`
を確認し、真であれば「MaxTokensに達して切り詰められたため解析に失敗した」ことが分かる
専用のエラーメッセージを返す（`enrichBatchSize` の見直しが必要、という診断が即座につく
ようにする）。それ以外の解析失敗は従来通り汎用メッセージのままとする。

### 5.7 テスト容易性のためのインターフェース化

`Messages.New` のみを抽象化した `messageCreator` インターフェースを `usage.go`
（`enrich.go`/`summarize.go`/`preference.go` が共有するファイル）に定義し、3エージェントの
`client` フィールドを具象の `anthropic.Client` からこのインターフェースに変更する
（各 `NewXxxAgent` では `client := anthropic.NewClient(); ...client: &client.Messages` の形に
なる）。`runAgentLoop`/`runAgentLoopWithUsageLog`（`loop.go`/`usage.go`）の `client` 引数の
型もこのインターフェースに変更し、3エージェント全てが同じ抽象化に揃う。

> **設計の試行錯誤**：当初は `messageCreator` を `enrich.go` だけに定義していたが、
> コードレビューで「`summarize`/`preference` は具象型のままで、テスト容易性の改善が
> 3エージェント中1つだけに留まっている」と指摘され、共有ファイルに移動して統一した。

`enrichAgent.Run()` はAPIを実際に呼ばずにテストできる（テストでは `messageCreator`/
`articlerepo.Repository` のfakeを使い、複数チャンク成功・部分失敗・DB書き込み失敗・
1チャンクのDB書き込み失敗が他チャンクを巻き込まないこと・ctx既キャンセル・実行中キャンセル
の6パターン以上を検証する）。`summarize`/`preference` 自体のテスト追加は本フェーズの
スコープ外（フォローアップ参照）。

### 5.8 CLIの一貫性：SilenceUsage

`cmd.SilenceUsage = true` を `enrich` サブコマンドだけに設定すると、同じ問題
（実行時エラーとcobraの自動usage出力が混在して紛らわしい）が `summarize`/`preference` にも
当てはまるのに片方しか直っていない状態になる。`cmd/agent/main.go` の**ルートコマンド**に
`root.SilenceUsage = true` を設定することで、全サブコマンドに一括で適用する
（cobraは実行されたサブコマンド自身か、`Execute()` を呼んだコマンド＝ルートのいずれかで
`SilenceUsage` が真であればusage出力を抑制する）。

### 5.9 重複IDの排除

モデルが同一記事IDを含む結果を複数回返した場合、`requested[r.ID]` のフィルタだけでは
重複を除けず、同一記事への冗長な `UPDATE` 文や処理件数 `n` の過大表示につながる。
チャンクごとの結果を `UpdateEnrichmentBatch` に渡す前に `buildEnrichmentUpdates` で
ID単位の重複除去（最初の1件を採用）を行う。

### 5.10 使われなくなった単発更新の削除

チャンク単位の `UpdateEnrichmentBatch` に置き換えたことで、1記事ずつ更新する旧
`Repository.UpdateEnrichment` は本番コードから一切呼ばれなくなった。インターフェース・
実装・専用テスト・usecase側の5つのモック実装から完全に削除する（インターフェースに
残したまま放置すると、実体のない「存続義務」だけが残り続けるため）。
