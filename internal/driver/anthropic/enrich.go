package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/samber/do/v2"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

type enrichAgent struct {
	client anthropic.Client
	repo   articlerepo.Repository
}

func NewEnrichAgent(i do.Injector) (adapteranthropic.EnrichAgent, error) {
	return &enrichAgent{
		client: anthropic.NewClient(),
		repo:   do.MustInvoke[articlerepo.Repository](i),
	}, nil
}

type enrichResult struct {
	ID       int64  `json:"id"`
	Summary  string `json:"summary"`
	Category string `json:"category"`
}

// Run は要約・カテゴリが未設定の記事に対して Claude に要約・カテゴリ分類させ、結果を DB に保存する。
// Force が true の場合は要約済みの記事も含めた最新記事を対象に再処理する。
// 処理した記事数を返す。
func (a *enrichAgent) Run(ctx context.Context, opts adapteranthropic.EnrichOptions) (int, error) {
	if opts.Limit == 0 {
		opts.Limit = 10
	}

	var articles []domain.Article
	var err error
	if opts.Force {
		articles, err = a.repo.FetchLatest(ctx, opts.Limit, "")
	} else {
		articles, err = a.repo.FindWithoutSummary(ctx, opts.Limit)
	}
	if err != nil {
		return 0, err
	}
	if len(articles) == 0 {
		return 0, nil
	}

	requested := make(map[int64]bool, len(articles))
	for _, art := range articles {
		requested[art.ID] = true
	}

	results, err := a.summarizeAndCategorize(ctx, articles)
	if err != nil {
		return 0, err
	}

	n := 0
	for _, r := range results {
		if !requested[r.ID] {
			continue
		}
		if err := a.repo.UpdateEnrichment(ctx, r.ID, r.Summary, r.Category); err != nil {
			return n, fmt.Errorf("記事 %d の更新に失敗しました: %w", r.ID, err)
		}
		n++
	}

	return n, nil
}

func (a *enrichAgent) summarizeAndCategorize(ctx context.Context, articles []domain.Article) ([]enrichResult, error) {
	type articleInput struct {
		ID      int64  `json:"id"`
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	inputs := make([]articleInput, len(articles))
	for i, art := range articles {
		inputs[i] = articleInput{ID: art.ID, Title: art.Title, Content: art.Content}
	}
	b, err := json.Marshal(inputs)
	if err != nil {
		return nil, err
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeOpus4_8,
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

	resp, err := a.client.Messages.New(ctx, params)
	if err != nil {
		return nil, err
	}

	for _, block := range resp.Content {
		if tb, ok := block.AsAny().(anthropic.TextBlock); ok {
			var results []enrichResult
			if err := json.Unmarshal([]byte(extractJSON(tb.Text)), &results); err != nil {
				return nil, fmt.Errorf("レスポンスの解析に失敗しました: %w", err)
			}
			return results, nil
		}
	}

	return nil, fmt.Errorf("テキスト応答が見つかりませんでした")
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
