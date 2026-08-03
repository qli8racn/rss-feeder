package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/samber/do/v2"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
	adapterhtmlfetch "github.com/qli8racn/rss-feeder/internal/adapter/driver/htmlfetch"
	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

type enrichAgent struct {
	client  messageCreator
	repo    articlerepo.Repository
	fetcher adapterhtmlfetch.Fetcher
	logger  *slog.Logger
	userID  int64
}

// NewEnrichAgent は *domain.User をDIコンテナから取得し、userIDを保持する
// （NewPreferenceAgent と同じ理由。internal/driver/anthropic/preference.go 参照）。
// rss_enrich がユーザー横断で他ユーザーの記事を要約対象・課金対象にしてしまわないよう、
// FetchLatest・FindWithoutSummary・UpdateEnrichmentBatch の呼び出しに必ずuserIDを渡す。
func NewEnrichAgent(i do.Injector) (adapteranthropic.EnrichAgent, error) {
	client := newAnthropicClient()
	return &enrichAgent{
		client:  &client.Messages,
		repo:    do.MustInvoke[articlerepo.Repository](i),
		fetcher: do.MustInvoke[adapterhtmlfetch.Fetcher](i),
		logger:  do.MustInvoke[*slog.Logger](i),
		userID:  do.MustInvoke[*domain.User](i).ID,
	}, nil
}

type enrichResult struct {
	ID       int64  `json:"id"`
	Summary  string `json:"summary"`
	Category string `json:"category"`
}

// defaultEnrichBatchSize は1回のAPI呼び出しで処理する記事数のデフォルト値。
// 40件で output ≈ 3,400〜3,900トークンとなり、4096トークンの上限に対して
// 安全マージンがあることを実測した上で採用した値。`--batch-size` で上書きできる。
const defaultEnrichBatchSize = 40

// defaultEnrichConcurrency はバッチを並列実行する際の最大同時実行数のデフォルト値。
// `--concurrency` で上書きできる。
const defaultEnrichConcurrency = 4

// maxFetchConcurrency はバッチ内での URL フェッチの最大同時実行数。
// 同一ホストへの集中アクセスを避けるため上限を設ける。
const maxFetchConcurrency = 5

// Run は要約・カテゴリが未設定の記事に対して Claude に要約・カテゴリ分類させ、結果を DB に保存する。
// Force が true の場合は要約済みの記事も含めた最新記事を対象に再処理する。
// 記事数がバッチサイズ（opts.BatchSize、0以下なら defaultEnrichBatchSize）を超える場合は
// 複数バッチに分割し、並列度（opts.Concurrency、0以下なら defaultEnrichConcurrency）を
// 上限に並列実行する（1バッチが失敗しても他バッチの処理・DB保存は継続する。DB書き込みも
// バッチ単位で行うため、1バッチのDB書き込み失敗が他バッチの保存済み結果を巻き込むことはない）。
// 処理した記事数を返す。
func (a *enrichAgent) Run(ctx context.Context, opts adapteranthropic.EnrichOptions) (int, error) {
	start := time.Now()
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if opts.Limit == 0 {
		opts.Limit = 10
	}
	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = defaultEnrichBatchSize
	}
	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = defaultEnrichConcurrency
	}

	var articles []domain.Article
	var err error
	if opts.Force {
		articles, err = a.repo.FetchLatest(ctx, opts.Limit, opts.FeedURL, a.userID)
	} else {
		articles, err = a.repo.FindWithoutSummary(ctx, opts.Limit, a.userID)
	}
	if err != nil {
		return 0, err
	}
	if len(articles) == 0 {
		return 0, nil
	}
	log := a.logger.With("command", "enrich")
	log.Info("開始しています...", "count", len(articles))

	requested := make(map[int64]bool, len(articles))
	for _, art := range articles {
		requested[art.ID] = true
	}

	chunks := chunkArticles(articles, batchSize)
	outcomes := make([]chunkOutcome, len(chunks))

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	dispatched := 0
	for i, chunk := range chunks {
		// 同時実行数の上限に達している間はここでブロックする。ctxのチェックは
		// このブロッキング送信より前に行っても意味がない（送信がブロックしている間に
		// 他のチャンクがctxをキャンセルする可能性があるため）。空きが出てから
		// チェックすることで、実行中にキャンセルされた場合、まだディスパッチしていない
		// チャンク分のコストを無駄にしない。既にディスパッチ済みのチャンクは、
		// APIクライアント自身がctxを見て中断するのに任せる。
		sem <- struct{}{}
		if ctx.Err() != nil {
			<-sem
			break
		}
		dispatched++
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			enrichedChunk := a.fetchFullContent(ctx, log, chunk)
			results, usage, err := a.summarizeAndCategorizeWithSplitRetry(ctx, enrichedChunk)
			outcomes[i] = chunkOutcome{results: results, usage: usage, err: err}
		}()
	}
	wg.Wait()
	for i := dispatched; i < len(chunks); i++ {
		outcomes[i] = chunkOutcome{err: fmt.Errorf("チャンク%d: ディスパッチ前にcontextがキャンセルされました: %w", i, ctx.Err())}
	}

	_, totalUsage, batchErr := aggregateChunkOutcomes(outcomes)
	logUsage(a.logger, anthropic.ModelClaudeHaiku4_5, totalUsage, time.Since(start))

	// DBへの保存はチャンク単位で行う。1チャンクのDB書き込みが失敗しても、他チャンクで
	// 既にコミット済みの結果はそのまま残る（チャンク単位の部分成功をDB層でも維持する）。
	n := 0
	errs := []error{batchErr}
	for _, o := range outcomes {
		if o.err != nil {
			continue
		}
		updates := buildEnrichmentUpdates(o.results, requested)
		if len(updates) == 0 {
			continue
		}
		if err := a.repo.UpdateEnrichmentBatch(ctx, updates, a.userID); err != nil {
			errs = append(errs, err)
			continue
		}
		n += len(updates)
	}

	return n, errors.Join(errs...)
}

// buildEnrichmentUpdates は1チャンクの要約結果から、要求された記事IDのみを対象とした
// 更新リストを作る。モデルが同一IDを重複して返した場合は最初の1件だけを採用する。
func buildEnrichmentUpdates(results []enrichResult, requested map[int64]bool) []articlerepo.EnrichmentUpdate {
	updates := make([]articlerepo.EnrichmentUpdate, 0, len(results))
	seen := make(map[int64]bool, len(results))
	for _, r := range results {
		if !requested[r.ID] || seen[r.ID] {
			continue
		}
		seen[r.ID] = true
		updates = append(updates, articlerepo.EnrichmentUpdate{ID: r.ID, Summary: r.Summary, Category: r.Category})
	}
	return updates
}

// chunkArticles は articles を size 件ずつのバッチに分割する。
func chunkArticles(articles []domain.Article, size int) [][]domain.Article {
	if len(articles) == 0 {
		return nil
	}
	chunks := make([][]domain.Article, 0, (len(articles)+size-1)/size)
	for i := 0; i < len(articles); i += size {
		end := i + size
		if end > len(articles) {
			end = len(articles)
		}
		chunks = append(chunks, articles[i:end])
	}
	return chunks
}

type chunkOutcome struct {
	results []enrichResult
	usage   anthropic.Usage
	err     error
}

// aggregateChunkOutcomes は各バッチの結果を集約する。一部のバッチが失敗しても
// 成功したバッチの結果は保持し、失敗は errors.Join でまとめて返す（部分成功を許容する）。
// Usage は失敗したバッチ（レスポンス解析失敗など、APIリクエスト自体は成功したケース）の
// ものも含めて合算する。
func aggregateChunkOutcomes(outcomes []chunkOutcome) ([]enrichResult, anthropic.Usage, error) {
	var allResults []enrichResult
	var totalUsage anthropic.Usage
	var errs []error
	for _, o := range outcomes {
		totalUsage = addUsage(totalUsage, o.usage)
		if o.err != nil {
			errs = append(errs, o.err)
			continue
		}
		allResults = append(allResults, o.results...)
	}
	return allResults, totalUsage, errors.Join(errs...)
}

// maxContentRunes は要約対象として送信する記事本文の最大文字数。
// 入力トークンを抑えるため、これを超える本文は末尾を切り詰める。
const maxContentRunes = 2000

// errMaxTokensTruncated は、Claudeのレスポンスが MaxTokens に達して途中で切り詰められたために
// JSON解析に失敗したことを表すsentinelエラー。summarizeAndCategorizeWithSplitRetry が
// errors.Is で検知し、チャンクを分割して再試行するかどうかを判断する。
var errMaxTokensTruncated = errors.New("MaxTokensに達してレスポンスが途中で切り詰められました")

// minSplitRetrySize は分割リトライ時にこれ以下のサイズには分割しない下限。
// 1記事だけでもMaxTokensに達する場合は、本文（maxContentRunesで切り詰め後）自体が大きすぎる
// ことが原因であり、それ以上分割しても解決しないため、分割を諦めてエラーを返す。
const minSplitRetrySize = 1

// summarizeAndCategorizeWithSplitRetry は summarizeAndCategorize を呼び、MaxTokens切り詰めに
// よる失敗（errMaxTokensTruncated）を検知した場合はチャンクを半分に分割して再試行する
// （半分ずつでもMaxTokensに達する場合は、minSplitRetrySizeに達するまで再帰的に分割を続ける）。
// 分割した一方が成功し他方が失敗しても、成功した結果は保持する（チャンク内でも部分成功を許容する）。
func (a *enrichAgent) summarizeAndCategorizeWithSplitRetry(ctx context.Context, articles []domain.Article) ([]enrichResult, anthropic.Usage, error) {
	results, usage, err := a.summarizeAndCategorize(ctx, articles)
	if err == nil {
		return results, usage, nil
	}
	if !errors.Is(err, errMaxTokensTruncated) || len(articles) <= minSplitRetrySize {
		return nil, usage, err
	}

	mid := len(articles) / 2
	leftResults, leftUsage, leftErr := a.summarizeAndCategorizeWithSplitRetry(ctx, articles[:mid])
	rightResults, rightUsage, rightErr := a.summarizeAndCategorizeWithSplitRetry(ctx, articles[mid:])

	// 分割前の失敗した試行分のUsageも、課金は発生しているため合算する。
	totalUsage := addUsage(addUsage(usage, leftUsage), rightUsage)
	return append(leftResults, rightResults...), totalUsage, errors.Join(leftErr, rightErr)
}

// summarizeAndCategorize は1バッチ分の記事をまとめて Claude に要約・分類させる。
// 戻り値の Usage は、解析失敗時（resp.Usage は得られているが JSON 解析に失敗した場合）も
// 呼び出し元でコストを集計できるよう、エラーの有無に関わらず返す。
func (a *enrichAgent) summarizeAndCategorize(ctx context.Context, articles []domain.Article) ([]enrichResult, anthropic.Usage, error) {
	model := anthropic.ModelClaudeHaiku4_5

	type articleInput struct {
		ID      int64  `json:"id"`
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	inputs := make([]articleInput, len(articles))
	for i, art := range articles {
		inputs[i] = articleInput{ID: art.ID, Title: art.Title, Content: truncateRunes(art.Content, maxContentRunes)}
	}
	b, err := json.Marshal(inputs)
	if err != nil {
		return nil, anthropic.Usage{}, err
	}

	params := anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{{
			Text: "あなたはRSS記事の要約・分類アシスタントです。" +
				"各記事について、日本語で簡潔な要約（2〜3文）と、内容を表すカテゴリ（例: Tech, Business, Sports など）を付与してください。" +
				`出力は説明文を含まず、次の形式のJSON配列のみを返してください: [{"id": 1, "summary": "...", "category": "..."}]`,
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(string(b))),
		},
	}

	resp, err := a.client.New(ctx, params)
	if err != nil {
		return nil, anthropic.Usage{}, err
	}
	usage := resp.Usage

	for _, block := range resp.Content {
		if tb, ok := block.AsAny().(anthropic.TextBlock); ok {
			var results []enrichResult
			if err := json.Unmarshal([]byte(extractJSON(tb.Text)), &results); err != nil {
				if resp.StopReason == anthropic.StopReasonMaxTokens {
					return nil, usage, fmt.Errorf(
						"%w（バッチ%d件、MaxTokens=%d）: %w",
						errMaxTokensTruncated, len(articles), params.MaxTokens, err)
				}
				return nil, usage, fmt.Errorf("レスポンスの解析に失敗しました: %w", err)
			}
			return results, usage, nil
		}
	}

	return nil, usage, fmt.Errorf("テキスト応答が見つかりませんでした")
}

// truncateRunes は s をルーン数 max までに切り詰める。max 以下の場合はそのまま返す。
func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// extractJSON はテキストから最初の JSON 配列部分のみを取り出す（前後の説明文を除去する）。
func extractJSON(text string) string {
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 || end < start {
		return text
	}
	return text[start : end+1]
}

// fetchFullContent はバッチ内の各記事 URL から HTML を取得してテキストを抽出し、
// Content フィールドを上書きした記事スライスを返す。
// フェッチ失敗・テキスト抽出結果が空の場合は元の Content を保持する（フォールバック）。
// URL フェッチは maxFetchConcurrency を上限に並列実行する。
func (a *enrichAgent) fetchFullContent(ctx context.Context, log *slog.Logger, articles []domain.Article) []domain.Article {
	result := make([]domain.Article, len(articles))
	copy(result, articles)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var successCount int
	sem := make(chan struct{}, maxFetchConcurrency)

	for i := range articles {
		if articles[i].URL == "" {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			html, err := a.fetcher.Fetch(ctx, articles[idx].URL)
			if err != nil {
				return
			}
			text := extractTextFromHTML(html)
			if text == "" {
				return
			}
			result[idx].Content = text
			mu.Lock()
			successCount++
			mu.Unlock()
		}(i)
	}
	wg.Wait()
	log.Info("フルテキスト取得", "success", successCount, "total", len(articles))
	return result
}

// extractTextFromHTML は HTML から読み取り可能なテキストを抽出する。
// ノイズ要素（script/style/nav/header/footer/aside）を除去したあと、
// article → main → [role=main] → body の順で最初に見つかった要素のテキストを返す。
func extractTextFromHTML(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}
	doc.Find("script, style, nav, header, footer, aside").Remove()

	for _, sel := range []string{"article", "main", "[role=main]", ".content", ".post-content", ".entry-content", "body"} {
		text := strings.Join(strings.Fields(doc.Find(sel).Text()), " ")
		if text != "" {
			return text
		}
	}
	return ""
}
