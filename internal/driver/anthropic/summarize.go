package anthropic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/samber/do/v2"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
)

type summarizeAgent struct {
	client anthropic.Client
	reader articlerepo.Repository
}

func NewSummarizeAgent(i do.Injector) (adapteranthropic.SummarizeAgent, error) {
	return &summarizeAgent{
		client: anthropic.NewClient(),
		reader: do.MustInvoke[articlerepo.Repository](i),
	}, nil
}

func (a *summarizeAgent) Run(ctx context.Context, opts adapteranthropic.SummarizeOptions) (string, error) {
	if opts.Limit == 0 {
		opts.Limit = 10
	}

	tools := []anthropic.ToolUnionParam{
		{OfTool: &anthropic.ToolParam{
			Name:        "fetch_articles",
			Description: anthropic.String("SQLiteから記事を取得する。feed_urlを指定すると特定フィードの記事のみ取得。"),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"limit":    map[string]any{"type": "integer", "description": "取得件数"},
					"feed_url": map[string]any{"type": "string", "description": "絞り込むフィードのURL（省略可）"},
				},
			},
		}},
	}

	query := "最新記事を要約してください。各記事のタイトルと要点を簡潔にまとめてください。"
	if opts.FeedURL != "" {
		query = fmt.Sprintf("フィード %q の記事を要約してください。", opts.FeedURL)
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{{
			Text: "あなたはRSS記事の要約アシスタントです。fetch_articlesツールで記事を取得し、わかりやすく日本語で要約してください。",
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(query)),
		},
		Tools: tools,
	}

	defaultLimit := opts.Limit
	return runAgentLoop(ctx, a.client, params, func(_, inputJSON string) (string, error) {
		var input struct {
			FeedURL string `json:"feed_url"`
		}
		if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
			return "", fmt.Errorf("invalid tool input: %w", err)
		}
		limit, err := parseLimitInput(inputJSON, defaultLimit, 0)
		if err != nil {
			return "", err
		}
		articles, err := a.reader.FetchLatest(ctx, limit, input.FeedURL)
		if err != nil {
			return "", err
		}
		if len(articles) == 0 {
			return "記事が見つかりませんでした。先に bin/rss-feeder fetch を実行してください。", nil
		}
		b, err := json.Marshal(toArticleJSONList(articles))
		if err != nil {
			return "", err
		}
		return string(b), nil
	})
}
