package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

// fakeFetcher は adapterhtmlfetch.Fetcher を実装するテスト用フェイク。
type fakeFetcher struct {
	fetch func(ctx context.Context, url string) (string, error)
}

func (f *fakeFetcher) Fetch(ctx context.Context, url string) (string, error) {
	return f.fetch(ctx, url)
}

func TestTruncateRunes_ShorterThanMax(t *testing.T) {
	got := truncateRunes("hello", 10)
	if got != "hello" {
		t.Errorf("truncateRunes: got %q, want %q", got, "hello")
	}
}

func TestTruncateRunes_EqualToMax(t *testing.T) {
	got := truncateRunes("hello", 5)
	if got != "hello" {
		t.Errorf("truncateRunes: got %q, want %q", got, "hello")
	}
}

func TestTruncateRunes_LongerThanMax(t *testing.T) {
	got := truncateRunes("hello world", 5)
	if got != "hello" {
		t.Errorf("truncateRunes: got %q, want %q", got, "hello")
	}
}

func TestTruncateRunes_MultiByteRunes(t *testing.T) {
	// マルチバイト文字（日本語）を含む文字列でも、バイト単位ではなくルーン単位で
	// 切り詰められ、文字が破壊されないことを確認する。
	got := truncateRunes("こんにちは世界", 5)
	if got != "こんにちは" {
		t.Errorf("truncateRunes: got %q, want %q", got, "こんにちは")
	}
}

func TestTruncateRunes_Empty(t *testing.T) {
	got := truncateRunes("", 5)
	if got != "" {
		t.Errorf("truncateRunes: got %q, want empty", got)
	}
}

func articlesWithIDs(ids ...int64) []domain.Article {
	articles := make([]domain.Article, len(ids))
	for i, id := range ids {
		articles[i] = domain.Article{ID: id}
	}
	return articles
}

// rangeIDs は start から end までの連番ID（両端含む）を返す。
func rangeIDs(start, end int64) []int64 {
	ids := make([]int64, 0, end-start+1)
	for id := start; id <= end; id++ {
		ids = append(ids, id)
	}
	return ids
}

func enrichOptions(limit int) adapteranthropic.EnrichOptions {
	return adapteranthropic.EnrichOptions{Limit: limit}
}

func TestChunkArticles_ExactMultiple(t *testing.T) {
	chunks := chunkArticles(articlesWithIDs(1, 2, 3, 4), 2)
	if len(chunks) != 2 {
		t.Fatalf("chunkArticles: got %d chunks, want 2", len(chunks))
	}
	if len(chunks[0]) != 2 || len(chunks[1]) != 2 {
		t.Errorf("chunkArticles: got chunk sizes %d, %d, want 2, 2", len(chunks[0]), len(chunks[1]))
	}
}

func TestChunkArticles_WithRemainder(t *testing.T) {
	chunks := chunkArticles(articlesWithIDs(1, 2, 3, 4, 5), 2)
	if len(chunks) != 3 {
		t.Fatalf("chunkArticles: got %d chunks, want 3", len(chunks))
	}
	if len(chunks[2]) != 1 {
		t.Errorf("chunkArticles: last chunk size got %d, want 1", len(chunks[2]))
	}
}

func TestChunkArticles_Empty(t *testing.T) {
	chunks := chunkArticles(nil, 2)
	if len(chunks) != 0 {
		t.Errorf("chunkArticles: got %d chunks, want 0", len(chunks))
	}
}

func TestAggregateChunkOutcomes_AllSucceed(t *testing.T) {
	outcomes := []chunkOutcome{
		{results: []enrichResult{{ID: 1, Summary: "a", Category: "Tech"}}, usage: anthropic.Usage{InputTokens: 10, OutputTokens: 5}},
		{results: []enrichResult{{ID: 2, Summary: "b", Category: "Business"}}, usage: anthropic.Usage{InputTokens: 20, OutputTokens: 8}},
	}

	results, usage, err := aggregateChunkOutcomes(outcomes)
	if err != nil {
		t.Fatalf("aggregateChunkOutcomes: unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("aggregateChunkOutcomes: got %d results, want 2", len(results))
	}
	if usage.InputTokens != 30 || usage.OutputTokens != 13 {
		t.Errorf("aggregateChunkOutcomes: got usage %+v, want input=30 output=13", usage)
	}
}

func TestAggregateChunkOutcomes_PartialFailure(t *testing.T) {
	// 一部のバッチが失敗しても、成功したバッチの結果は保持されることを確認する
	// （部分成功を許容する設計）。
	failureErr := errors.New("レスポンスの解析に失敗しました")
	outcomes := []chunkOutcome{
		{results: []enrichResult{{ID: 1, Summary: "a", Category: "Tech"}}, usage: anthropic.Usage{InputTokens: 10, OutputTokens: 5}},
		{err: failureErr, usage: anthropic.Usage{InputTokens: 15, OutputTokens: 2}},
	}

	results, usage, err := aggregateChunkOutcomes(outcomes)
	if len(results) != 1 || results[0].ID != 1 {
		t.Fatalf("aggregateChunkOutcomes: got results %+v, want only the successful chunk's result", results)
	}
	if err == nil || !errors.Is(err, failureErr) {
		t.Errorf("aggregateChunkOutcomes: got err %v, want it to wrap %v", err, failureErr)
	}
	// 失敗したバッチも resp.Usage は取得済みのため、コスト集計には含める。
	if usage.InputTokens != 25 || usage.OutputTokens != 7 {
		t.Errorf("aggregateChunkOutcomes: got usage %+v, want input=25 output=7 (failed batch usage included)", usage)
	}
}

func TestAggregateChunkOutcomes_AllFail(t *testing.T) {
	outcomes := []chunkOutcome{
		{err: errors.New("error 1")},
		{err: errors.New("error 2")},
	}

	results, _, err := aggregateChunkOutcomes(outcomes)
	if len(results) != 0 {
		t.Errorf("aggregateChunkOutcomes: got %d results, want 0", len(results))
	}
	if err == nil {
		t.Error("aggregateChunkOutcomes: want non-nil error when all chunks fail")
	}
}

// --- Run() のテスト用フェイク ---

// fakeRepo は articlerepo.Repository を埋め込み、テストで使うメソッドだけを上書きする。
// 上書きしていないメソッドが呼ばれた場合は nil 埋め込みの呼び出しとなり、即座に panic して
// テストが失敗するため、想定外の呼び出しはすぐに検出できる。
type fakeRepo struct {
	articlerepo.Repository
	findWithoutSummary func(ctx context.Context, limit int) ([]domain.Article, error)
	fetchLatest        func(ctx context.Context, limit int, feedURL string) ([]domain.Article, error)
	updateBatch        func(ctx context.Context, updates []articlerepo.EnrichmentUpdate) error
}

func (f *fakeRepo) FindWithoutSummary(ctx context.Context, limit int) ([]domain.Article, error) {
	return f.findWithoutSummary(ctx, limit)
}

func (f *fakeRepo) FetchLatest(ctx context.Context, limit int, feedURL string) ([]domain.Article, error) {
	return f.fetchLatest(ctx, limit, feedURL)
}

func (f *fakeRepo) UpdateEnrichmentBatch(ctx context.Context, updates []articlerepo.EnrichmentUpdate) error {
	return f.updateBatch(ctx, updates)
}

// fakeMessageCreator は messageCreator を実装し、実APIを呼ばずにレスポンスをfakeできる。
type fakeMessageCreator struct {
	new func(ctx context.Context, body anthropic.MessageNewParams, opts ...option.RequestOption) (*anthropic.Message, error)
}

func (f *fakeMessageCreator) New(ctx context.Context, body anthropic.MessageNewParams, opts ...option.RequestOption) (*anthropic.Message, error) {
	return f.new(ctx, body, opts...)
}

// fakeMessage はAPIレスポンスのJSONを組み立てて anthropic.Message にデコードしたものを返す。
// ContentBlockUnion.AsAny() はSDK内部でJSONデコード時に保持される生バイト列を参照するため、
// 構造体リテラルを直接組み立てるのではなく、実際にJSONを経由してデコードする必要がある。
func fakeMessage(t *testing.T, text string, usage anthropic.Usage, stopReason anthropic.StopReason) *anthropic.Message {
	t.Helper()
	payload := map[string]any{
		"id":   "msg_test",
		"type": "message",
		"role": "assistant",
		"content": []map[string]any{
			{"type": "text", "text": text},
		},
		"model":       "claude-haiku-4-5",
		"stop_reason": stopReason,
		"usage": map[string]any{
			"input_tokens":  usage.InputTokens,
			"output_tokens": usage.OutputTokens,
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("fakeMessage: marshal: %v", err)
	}
	var msg anthropic.Message
	if err := json.Unmarshal(b, &msg); err != nil {
		t.Fatalf("fakeMessage: unmarshal: %v", err)
	}
	return &msg
}

// requestedIDs はリクエストの本文（articleInput のJSON配列）から id だけを取り出す。
func requestedIDs(body anthropic.MessageNewParams) []int64 {
	var text string
	for _, m := range body.Messages {
		for _, c := range m.Content {
			if c.OfText != nil {
				text = c.OfText.Text
			}
		}
	}
	var inputs []struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal([]byte(text), &inputs)
	ids := make([]int64, len(inputs))
	for i, in := range inputs {
		ids[i] = in.ID
	}
	return ids
}

// echoSuccess はリクエストされた記事IDをそのまま要約結果として返すfakeハンドラを生成する。
func echoSuccess(t *testing.T, usage anthropic.Usage) func(context.Context, anthropic.MessageNewParams, ...option.RequestOption) (*anthropic.Message, error) {
	return func(_ context.Context, body anthropic.MessageNewParams, _ ...option.RequestOption) (*anthropic.Message, error) {
		ids := requestedIDs(body)
		results := make([]enrichResult, len(ids))
		for i, id := range ids {
			results[i] = enrichResult{ID: id, Summary: "要約", Category: "Tech"}
		}
		b, err := json.Marshal(results)
		if err != nil {
			return nil, err
		}
		return fakeMessage(t, string(b), usage, anthropic.StopReasonEndTurn), nil
	}
}

func TestEnrichAgent_Run_MultipleChunksSucceed(t *testing.T) {
	// defaultEnrichBatchSize(40) を超える件数を渡し、複数バッチに分割されても全件処理されることを確認する。
	articles := articlesWithIDs(rangeIDs(1, 90)...)
	var gotUpdates []articlerepo.EnrichmentUpdate
	repo := &fakeRepo{
		findWithoutSummary: func(_ context.Context, _ int) ([]domain.Article, error) { return articles, nil },
		updateBatch: func(_ context.Context, updates []articlerepo.EnrichmentUpdate) error {
			// DB書き込みはチャンクごとに呼ばれる（メインゴルーチンで直列実行されるため
			// 並行アクセスの心配はない）。
			gotUpdates = append(gotUpdates, updates...)
			return nil
		},
	}
	agent := &enrichAgent{
		client: &fakeMessageCreator{new: echoSuccess(t, anthropic.Usage{InputTokens: 100, OutputTokens: 50})},
		repo:   repo,
	}

	n, err := agent.Run(context.Background(), enrichOptions(90))
	if err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}
	if n != 90 {
		t.Errorf("Run: got n=%d, want 90", n)
	}
	if len(gotUpdates) != 90 {
		t.Errorf("UpdateEnrichmentBatch: got %d updates, want 90", len(gotUpdates))
	}
}

func TestEnrichAgent_Run_CustomBatchSizeOverridesDefault(t *testing.T) {
	// --batch-size 相当の opts.BatchSize を指定した場合、defaultEnrichBatchSize(40) ではなく
	// 指定値でチャンク分割されることを、API呼び出し回数（チャンク数）で確認する。
	articles := articlesWithIDs(rangeIDs(1, 10)...)
	repo := &fakeRepo{
		findWithoutSummary: func(_ context.Context, _ int) ([]domain.Article, error) { return articles, nil },
		updateBatch:        func(_ context.Context, _ []articlerepo.EnrichmentUpdate) error { return nil },
	}
	var calls int32
	agent := &enrichAgent{
		client: &fakeMessageCreator{new: func(c context.Context, body anthropic.MessageNewParams, opts ...option.RequestOption) (*anthropic.Message, error) {
			atomic.AddInt32(&calls, 1)
			return echoSuccess(t, anthropic.Usage{})(c, body, opts...)
		}},
		repo: repo,
	}

	opts := enrichOptions(10)
	opts.BatchSize = 3
	n, err := agent.Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}
	if n != 10 {
		t.Errorf("Run: got n=%d, want 10", n)
	}
	// 10件を3件ずつに分割すると4チャンク（3,3,3,1）になるはず。
	if got := atomic.LoadInt32(&calls); got != 4 {
		t.Errorf("Run: got %d API calls, want 4 (10 articles / batch-size 3)", got)
	}
}

func TestEnrichAgent_Run_CustomConcurrencyOverridesDefault(t *testing.T) {
	// --concurrency 相当の opts.Concurrency を指定した場合、defaultEnrichConcurrency(4) ではなく
	// 指定値が同時実行数の上限として使われることを確認する
	// （TestEnrichAgent_Run_StopsDispatchingAfterCancel と同じ手法：ctxキャンセルでブロックさせ、
	// 上限を超えたチャンクがディスパッチされないことを呼び出し回数で検証する）。
	articles := articlesWithIDs(rangeIDs(1, 200)...) // defaultEnrichBatchSize=40で5チャンク
	repo := &fakeRepo{
		findWithoutSummary: func(_ context.Context, _ int) ([]domain.Article, error) { return articles, nil },
		updateBatch:        func(_ context.Context, _ []articlerepo.EnrichmentUpdate) error { return nil },
	}
	ctx, cancel := context.WithCancel(context.Background())
	var calls int32
	agent := &enrichAgent{
		client: &fakeMessageCreator{new: func(c context.Context, body anthropic.MessageNewParams, opts ...option.RequestOption) (*anthropic.Message, error) {
			atomic.AddInt32(&calls, 1)
			cancel()
			return echoSuccess(t, anthropic.Usage{})(c, body, opts...)
		}},
		repo: repo,
	}

	opts := enrichOptions(200)
	opts.Concurrency = 2
	if _, err := agent.Run(ctx, opts); err == nil {
		t.Fatal("Run: want non-nil error when ctx is canceled mid-dispatch")
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("Run: got %d API calls, want exactly 2 (custom concurrency limit)", got)
	}
}

func TestSummarizeAndCategorizeWithSplitRetry_RecoversFromMaxTokensBySplitting(t *testing.T) {
	// 4件まとめてリクエストするとMaxTokens切り詰めで失敗するが、2件以下に分割すれば
	// 成功するfakeクライアントで、分割リトライにより最終的に全件処理されることを確認する。
	agent := &enrichAgent{
		client: &fakeMessageCreator{new: func(ctx context.Context, body anthropic.MessageNewParams, opts ...option.RequestOption) (*anthropic.Message, error) {
			ids := requestedIDs(body)
			if len(ids) > 2 {
				return fakeMessage(t, "invalid json", anthropic.Usage{InputTokens: 10, OutputTokens: 2}, anthropic.StopReasonMaxTokens), nil
			}
			return echoSuccess(t, anthropic.Usage{InputTokens: 10, OutputTokens: 5})(ctx, body, opts...)
		}},
	}

	results, _, err := agent.summarizeAndCategorizeWithSplitRetry(context.Background(), articlesWithIDs(1, 2, 3, 4))
	if err != nil {
		t.Fatalf("summarizeAndCategorizeWithSplitRetry: unexpected error: %v", err)
	}
	if len(results) != 4 {
		t.Errorf("summarizeAndCategorizeWithSplitRetry: got %d results, want 4", len(results))
	}
}

func TestSummarizeAndCategorizeWithSplitRetry_GivesUpAtMinSplitSize(t *testing.T) {
	// 1件にまで分割してもMaxTokens切り詰めが解消しない場合、それ以上は分割せずエラーを返す
	// （本文が大きすぎることが原因であり、分割では解決しないと判断するケース）。
	var calls int32
	agent := &enrichAgent{
		client: &fakeMessageCreator{new: func(ctx context.Context, body anthropic.MessageNewParams, opts ...option.RequestOption) (*anthropic.Message, error) {
			atomic.AddInt32(&calls, 1)
			return fakeMessage(t, "invalid json", anthropic.Usage{InputTokens: 10, OutputTokens: 2}, anthropic.StopReasonMaxTokens), nil
		}},
	}

	_, _, err := agent.summarizeAndCategorizeWithSplitRetry(context.Background(), articlesWithIDs(1, 2))
	if err == nil {
		t.Fatal("summarizeAndCategorizeWithSplitRetry: want non-nil error when even a single article hits MaxTokens")
	}
	if !errors.Is(err, errMaxTokensTruncated) {
		t.Errorf("summarizeAndCategorizeWithSplitRetry: got err=%v, want errMaxTokensTruncated", err)
	}
	// 2件 → 1件+1件の3回呼ばれ、それぞれが1件まで分割された時点で諦めて終わるはず
	// （1件はminSplitRetrySizeのため、さらに半分(0件)には分割しない）。
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Errorf("Run: got %d API calls, want 3 (1 for the original 2-article batch, 1 each for the two 1-article retries)", got)
	}
}

func TestEnrichAgent_Run_PartialChunkFailureStillSavesSuccessfulChunks(t *testing.T) {
	// 90件 = 3チャンク（40,40,10）。2番目のチャンクだけJSON解析に失敗させ、
	// 1番目・3番目のチャンクの結果は保存され、エラーも返ることを確認する（部分成功）。
	articles := articlesWithIDs(rangeIDs(1, 90)...)
	var gotUpdates []articlerepo.EnrichmentUpdate
	repo := &fakeRepo{
		findWithoutSummary: func(_ context.Context, _ int) ([]domain.Article, error) { return articles, nil },
		updateBatch: func(_ context.Context, updates []articlerepo.EnrichmentUpdate) error {
			gotUpdates = append(gotUpdates, updates...)
			return nil
		},
	}
	// チャンクは並列に実行されるため呼び出し順は保証されない。リクエスト内容
	// （2番目のチャンクに含まれるID=41）で判定することで、どの順に実行されても
	// 「2番目のチャンクだけ失敗する」を確定的に再現できる。
	// StopReasonはMaxTokens以外（EndTurn）にすることで、分割リトライ
	// （summarizeAndCategorizeWithSplitRetry）が発動せず、チャンク全体が
	// そのまま失敗するケースをテストする（MaxTokens起因の分割リトライ自体は
	// TestSummarizeAndCategorizeWithSplitRetry_* で別途テストする）。
	agent := &enrichAgent{
		client: &fakeMessageCreator{new: func(ctx context.Context, body anthropic.MessageNewParams, opts ...option.RequestOption) (*anthropic.Message, error) {
			ids := requestedIDs(body)
			for _, id := range ids {
				if id == 41 {
					return fakeMessage(t, "invalid json", anthropic.Usage{InputTokens: 10, OutputTokens: 2}, anthropic.StopReasonEndTurn), nil
				}
			}
			return echoSuccess(t, anthropic.Usage{InputTokens: 10, OutputTokens: 5})(ctx, body, opts...)
		}},
		repo: repo,
	}

	n, err := agent.Run(context.Background(), enrichOptions(90))
	if err == nil {
		t.Fatal("Run: want non-nil error when one chunk fails")
	}
	if n != 50 {
		t.Errorf("Run: got n=%d, want 50 (40+10 from the two successful chunks)", n)
	}
	if len(gotUpdates) != 50 {
		t.Errorf("UpdateEnrichmentBatch: got %d updates, want 50", len(gotUpdates))
	}
}

func TestEnrichAgent_Run_DBWriteFailureJoinsBatchErr(t *testing.T) {
	// 唯一のチャンク（40件）でAPI呼び出しは成功するが、DB書き込みが失敗するケース。
	// dbErrがbatchErrと合成されてerrorとして返り、n=0（そのチャンクは未コミット）になる
	// ことを確認する。
	articles := articlesWithIDs(rangeIDs(1, 40)...)
	repo := &fakeRepo{
		findWithoutSummary: func(_ context.Context, _ int) ([]domain.Article, error) { return articles, nil },
		updateBatch: func(_ context.Context, _ []articlerepo.EnrichmentUpdate) error {
			return errors.New("disk full")
		},
	}
	agent := &enrichAgent{
		client: &fakeMessageCreator{new: echoSuccess(t, anthropic.Usage{InputTokens: 10, OutputTokens: 5})},
		repo:   repo,
	}

	n, err := agent.Run(context.Background(), enrichOptions(40))
	if err == nil {
		t.Fatal("Run: want non-nil error when DB write fails")
	}
	if n != 0 {
		t.Errorf("Run: got n=%d, want 0 (transaction should not have committed)", n)
	}
}

func TestEnrichAgent_Run_OneChunkDBWriteFailsOthersStillSaved(t *testing.T) {
	// 90件 = 3チャンク（40,40,10）。全チャンクのAPI呼び出しは成功するが、ID=41を含む
	// チャンクのDB書き込みだけ失敗させる。DB書き込みはチャンク単位のトランザクションのため、
	// 他チャンクの保存済み結果が巻き込まれてロールバックされないことを確認する
	// （DB層でもチャンク単位の部分成功を維持する）。
	articles := articlesWithIDs(rangeIDs(1, 90)...)
	var gotUpdates []articlerepo.EnrichmentUpdate
	repo := &fakeRepo{
		findWithoutSummary: func(_ context.Context, _ int) ([]domain.Article, error) { return articles, nil },
		updateBatch: func(_ context.Context, updates []articlerepo.EnrichmentUpdate) error {
			for _, u := range updates {
				if u.ID == 41 {
					return errors.New("disk full")
				}
			}
			gotUpdates = append(gotUpdates, updates...)
			return nil
		},
	}
	agent := &enrichAgent{
		client: &fakeMessageCreator{new: echoSuccess(t, anthropic.Usage{InputTokens: 10, OutputTokens: 5})},
		repo:   repo,
	}

	n, err := agent.Run(context.Background(), enrichOptions(90))
	if err == nil {
		t.Fatal("Run: want non-nil error when one chunk's DB write fails")
	}
	if n != 50 {
		t.Errorf("Run: got n=%d, want 50 (40+10 from the two chunks whose DB write succeeded)", n)
	}
	if len(gotUpdates) != 50 {
		t.Errorf("UpdateEnrichmentBatch: got %d persisted updates, want 50", len(gotUpdates))
	}
}

func TestEnrichAgent_Run_StopsDispatchingAfterContextCanceled(t *testing.T) {
	// 200件 = 5チャンク、並列度4。最初に実行された4チャンクのいずれかがctxをキャンセルする
	// ため、5番目のチャンクはディスパッチ前にスキップされ、APIが呼ばれないことを確認する
	// （実行中キャンセルへの対応）。
	articles := articlesWithIDs(rangeIDs(1, 200)...)
	repo := &fakeRepo{
		findWithoutSummary: func(_ context.Context, _ int) ([]domain.Article, error) { return articles, nil },
		updateBatch:        func(_ context.Context, _ []articlerepo.EnrichmentUpdate) error { return nil },
	}
	ctx, cancel := context.WithCancel(context.Background())
	var calls int32
	agent := &enrichAgent{
		client: &fakeMessageCreator{new: func(c context.Context, body anthropic.MessageNewParams, opts ...option.RequestOption) (*anthropic.Message, error) {
			atomic.AddInt32(&calls, 1)
			cancel()
			return echoSuccess(t, anthropic.Usage{})(c, body, opts...)
		}},
		repo: repo,
	}

	_, err := agent.Run(ctx, enrichOptions(200))
	if err == nil {
		t.Fatal("Run: want non-nil error when ctx is canceled mid-dispatch")
	}
	if got := atomic.LoadInt32(&calls); got != defaultEnrichConcurrency {
		t.Errorf("Run: got %d API calls, want exactly %d (concurrency limit; 5th chunk should never be dispatched)", got, defaultEnrichConcurrency)
	}
}

func TestBuildEnrichmentUpdates_FiltersUnrequestedIDs(t *testing.T) {
	requested := map[int64]bool{1: true, 2: true}
	results := []enrichResult{
		{ID: 1, Summary: "a", Category: "Tech"},
		{ID: 99, Summary: "hallucinated", Category: "Tech"},
	}
	got := buildEnrichmentUpdates(results, requested)
	if len(got) != 1 || got[0].ID != 1 {
		t.Errorf("buildEnrichmentUpdates: got %+v, want only ID=1", got)
	}
}

func TestBuildEnrichmentUpdates_DedupesRepeatedIDs(t *testing.T) {
	requested := map[int64]bool{1: true}
	results := []enrichResult{
		{ID: 1, Summary: "first", Category: "Tech"},
		{ID: 1, Summary: "duplicate", Category: "Business"},
	}
	got := buildEnrichmentUpdates(results, requested)
	if len(got) != 1 {
		t.Fatalf("buildEnrichmentUpdates: got %d updates, want 1 (duplicate ID should be collapsed)", len(got))
	}
	if got[0].Summary != "first" {
		t.Errorf("buildEnrichmentUpdates: got summary %q, want first occurrence to win", got[0].Summary)
	}
}

func TestEnrichAgent_Run_ForceWithFeedURLScopesFetchLatest(t *testing.T) {
	// Force=true かつ FeedURL 指定時、FetchLatest がその feedURL で呼ばれることを確認する
	// （フィード追加時の自動enrichが、対象フィードの記事のみに絞り込めることの検証）。
	articles := articlesWithIDs(1, 2)
	var gotFeedURL string
	repo := &fakeRepo{
		fetchLatest: func(_ context.Context, _ int, feedURL string) ([]domain.Article, error) {
			gotFeedURL = feedURL
			return articles, nil
		},
		updateBatch: func(_ context.Context, _ []articlerepo.EnrichmentUpdate) error { return nil },
	}
	agent := &enrichAgent{
		client: &fakeMessageCreator{new: echoSuccess(t, anthropic.Usage{InputTokens: 10, OutputTokens: 5})},
		repo:   repo,
	}

	_, err := agent.Run(context.Background(), adapteranthropic.EnrichOptions{Limit: 2, Force: true, FeedURL: "https://example.com/feed"})
	if err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}
	if gotFeedURL != "https://example.com/feed" {
		t.Errorf("FetchLatest: got feedURL %q, want %q", gotFeedURL, "https://example.com/feed")
	}
}

func TestEnrichAgent_Run_AlreadyCanceledContext(t *testing.T) {
	// 既にキャンセル済みのctxでRunを呼んだ場合、API呼び出しを一切行わずに即座にエラーを返す
	// ことを確認する（キャンセル済みなのに全チャンクを無駄にディスパッチしない）。
	called := false
	repo := &fakeRepo{
		findWithoutSummary: func(_ context.Context, _ int) ([]domain.Article, error) {
			t.Fatal("FindWithoutSummary should not be called when ctx is already canceled")
			return nil, nil
		},
	}
	agent := &enrichAgent{
		client: &fakeMessageCreator{new: func(context.Context, anthropic.MessageNewParams, ...option.RequestOption) (*anthropic.Message, error) {
			called = true
			return nil, errors.New("should not be called")
		}},
		repo: repo,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	n, err := agent.Run(ctx, enrichOptions(10))
	if err == nil {
		t.Fatal("Run: want non-nil error for already-canceled context")
	}
	if n != 0 {
		t.Errorf("Run: got n=%d, want 0", n)
	}
	if called {
		t.Error("Run: API should not have been called for an already-canceled context")
	}
}

// --- extractTextFromHTML のテスト ---

func TestExtractTextFromHTML_ArticleTag(t *testing.T) {
	html := `<html><body><nav>ナビ</nav><article>本文テキスト</article></body></html>`
	got := extractTextFromHTML(html)
	if got != "本文テキスト" {
		t.Errorf("extractTextFromHTML: got %q, want %q", got, "本文テキスト")
	}
}

func TestExtractTextFromHTML_FallsBackToBody(t *testing.T) {
	// article/main などが存在しない場合は body 全体にフォールバックする。
	html := `<html><body><p>ボディテキスト</p></body></html>`
	got := extractTextFromHTML(html)
	if got != "ボディテキスト" {
		t.Errorf("extractTextFromHTML: got %q, want %q", got, "ボディテキスト")
	}
}

func TestExtractTextFromHTML_RemovesNoiseElements(t *testing.T) {
	// script・style はテキスト抽出前に削除される。
	html := `<html><body><script>var x=1;</script><style>.foo{}</style><article>本文</article></body></html>`
	got := extractTextFromHTML(html)
	if got != "本文" {
		t.Errorf("extractTextFromHTML: got %q, want %q", got, "本文")
	}
}

func TestExtractTextFromHTML_NormalizesWhitespace(t *testing.T) {
	html := `<html><body><article>  空白　　 の　テスト  </article></body></html>`
	got := extractTextFromHTML(html)
	// strings.Fields で分割・Join するため先頭末尾の空白と連続空白が除去される。
	if got == "" {
		t.Error("extractTextFromHTML: got empty, want non-empty")
	}
}

func TestExtractTextFromHTML_EmptyHTML(t *testing.T) {
	got := extractTextFromHTML("<html><body></body></html>")
	if got != "" {
		t.Errorf("extractTextFromHTML: got %q, want empty", got)
	}
}

// --- fetchFullContent のテスト ---

func TestFetchFullContent_UpdatesContentOnSuccess(t *testing.T) {
	articles := []domain.Article{
		{ID: 1, URL: "https://example.com/1", Content: "original"},
	}
	agent := &enrichAgent{
		fetcher: &fakeFetcher{
			fetch: func(_ context.Context, _ string) (string, error) {
				return `<html><body><article>フルテキスト</article></body></html>`, nil
			},
		},
	}
	result := agent.fetchFullContent(context.Background(), articles)
	if result[0].Content != "フルテキスト" {
		t.Errorf("fetchFullContent: Content got %q, want %q", result[0].Content, "フルテキスト")
	}
}

func TestFetchFullContent_SkipsEmptyURL(t *testing.T) {
	// URL が空の記事はフェッチをスキップし、元の Content を保持する。
	articles := []domain.Article{
		{ID: 1, URL: "", Content: "original"},
	}
	called := false
	agent := &enrichAgent{
		fetcher: &fakeFetcher{
			fetch: func(_ context.Context, _ string) (string, error) {
				called = true
				return "", nil
			},
		},
	}
	result := agent.fetchFullContent(context.Background(), articles)
	if called {
		t.Error("fetchFullContent: fetcher should not be called for empty URL")
	}
	if result[0].Content != "original" {
		t.Errorf("fetchFullContent: Content got %q, want %q", result[0].Content, "original")
	}
}

func TestFetchFullContent_FallbackOnFetchError(t *testing.T) {
	// フェッチ失敗時は元の Content を保持する（フォールバック）。
	articles := []domain.Article{
		{ID: 1, URL: "https://example.com/1", Content: "original"},
	}
	agent := &enrichAgent{
		fetcher: &fakeFetcher{
			fetch: func(_ context.Context, _ string) (string, error) {
				return "", errors.New("connection refused")
			},
		},
	}
	result := agent.fetchFullContent(context.Background(), articles)
	if result[0].Content != "original" {
		t.Errorf("fetchFullContent: Content got %q, want %q (should fallback on error)", result[0].Content, "original")
	}
}

func TestFetchFullContent_FallbackOnEmptyText(t *testing.T) {
	// HTML 取得は成功したがテキスト抽出結果が空の場合も元の Content を保持する。
	articles := []domain.Article{
		{ID: 1, URL: "https://example.com/1", Content: "original"},
	}
	agent := &enrichAgent{
		fetcher: &fakeFetcher{
			fetch: func(_ context.Context, _ string) (string, error) {
				return `<html><body></body></html>`, nil
			},
		},
	}
	result := agent.fetchFullContent(context.Background(), articles)
	if result[0].Content != "original" {
		t.Errorf("fetchFullContent: Content got %q, want %q (should fallback when text is empty)", result[0].Content, "original")
	}
}

func TestFetchFullContent_PreservesOtherFields(t *testing.T) {
	// Content 以外のフィールドが書き換えられないことを確認する。
	articles := []domain.Article{
		{ID: 42, URL: "https://example.com/1", Title: "タイトル", Content: "original"},
	}
	agent := &enrichAgent{
		fetcher: &fakeFetcher{
			fetch: func(_ context.Context, _ string) (string, error) {
				return `<html><body><article>新しい本文</article></body></html>`, nil
			},
		},
	}
	result := agent.fetchFullContent(context.Background(), articles)
	if result[0].ID != 42 || result[0].Title != "タイトル" {
		t.Errorf("fetchFullContent: non-Content fields modified: got %+v", result[0])
	}
}
