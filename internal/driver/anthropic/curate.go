package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/samber/do/v2"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

type curateAgent struct {
	client messageCreator
	repo   articlerepo.Repository
}

func NewCurateAgent(i do.Injector) (adapteranthropic.CurateAgent, error) {
	client := newAnthropicClient()
	return &curateAgent{
		client: &client.Messages,
		repo:   do.MustInvoke[articlerepo.Repository](i),
	}, nil
}

// defaultCurateLimit は fetch_recent_articles のデフォルト取得件数。
// Opus モデルへの入力トークンを抑えるため 30 件に絞る。
const defaultCurateLimit = 30

// maxCurateLimit は fetch_recent_articles の上限。
const maxCurateLimit = 100

// maxCurateBookmarks は fetch_bookmarked_articles で取得するブックマーク記事数の上限。
// preference.go の maxBookmarkedArticles と同様、ブックマーク増加による入力トークン膨張を防ぐ。
const maxCurateBookmarks = 50

type enrichedArticleJSON struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Category    string `json:"category,omitempty"`
	Summary     string `json:"summary,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
}

func toEnrichedArticleJSONList(articles []domain.Article) []enrichedArticleJSON {
	result := make([]enrichedArticleJSON, len(articles))
	for i, a := range articles {
		pub := ""
		if !a.PublishedAt.IsZero() {
			pub = a.PublishedAt.Format(time.RFC3339)
		}
		result[i] = enrichedArticleJSON{
			ID:          a.ID,
			Title:       a.Title,
			URL:         a.URL,
			Category:    a.Category,
			Summary:     a.Summary,
			PublishedAt: pub,
		}
	}
	return result
}

func (a *curateAgent) Run(ctx context.Context, opts adapteranthropic.CurateOptions) (string, error) {
	defaultLimit := defaultCurateLimit
	if opts.Limit > 0 {
		defaultLimit = opts.Limit
	}

	tools := []anthropic.ToolUnionParam{
		{OfTool: &anthropic.ToolParam{
			Name:        "fetch_recent_articles",
			Description: anthropic.String("直近の記事を取得する。summary・category が付与されている場合は含む。推薦候補として使用する。"),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"limit": map[string]any{
						"type":        "integer",
						"description": fmt.Sprintf("取得件数の上限（省略時は%d件、最大%d件）", defaultLimit, maxCurateLimit),
					},
				},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "fetch_bookmarked_articles",
			Description: anthropic.String("ブックマーク済みの記事を取得する。ユーザーの趣向把握に使用する。"),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{},
			},
		}},
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeOpus4_8,
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{{
			Text: "あなたはRSSキュレーターです。" +
				"ブックマーク済み記事からユーザーの技術的興味・趣向を把握し、" +
				"直近の記事の中から今日読む価値がある記事を選定して日本語で推薦してください。" +
				"各記事に推薦理由（1〜2文）を添え、重要度が高い順にランク付きで出力してください。" +
				"Markdown形式で出力し、最大5件を目安にしてください。",
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(
				"ブックマーク記事と直近記事を取得して、今日読む価値がある記事を推薦してください。",
			)),
		},
		Tools: tools,
		Thinking: anthropic.ThinkingConfigParamUnion{
			OfAdaptive: &anthropic.ThinkingConfigAdaptiveParam{},
		},
	}

	fmt.Fprintln(os.Stderr, "[curate] 開始しています...")
	return runAgentLoopWithUsageLog(ctx, a.client, params, func(name, inputJSON string) (string, error) {
		switch name {
		case "fetch_recent_articles":
			limit, err := parseLimitInput(inputJSON, defaultLimit, maxCurateLimit)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(os.Stderr, "[curate] 直近記事を取得中（最大%d件）...\n", limit)
			articles, err := a.repo.FetchLatest(ctx, limit, "")
			if err != nil {
				return "", err
			}
			if len(articles) == 0 {
				fmt.Fprintln(os.Stderr, "[curate] 直近記事: 0件")
				return "記事がありません。bin/rss-feeder fetch で記事を取得してください。", nil
			}
			fmt.Fprintf(os.Stderr, "[curate] 直近記事: %d件取得\n", len(articles))
			b, err := json.Marshal(toEnrichedArticleJSONList(articles))
			if err != nil {
				return "", err
			}
			return string(b), nil

		case "fetch_bookmarked_articles":
			fmt.Fprintln(os.Stderr, "[curate] ブックマーク記事を取得中...")
			articles, err := a.repo.FindBookmarked(ctx)
			if err != nil {
				return "", err
			}
			if len(articles) == 0 {
				fmt.Fprintln(os.Stderr, "[curate] ブックマーク記事: 0件（趣向情報なしで推薦）")
				return "ブックマークされた記事がありません。趣向情報なしで推薦します。", nil
			}
			if len(articles) > maxCurateBookmarks {
				articles = articles[:maxCurateBookmarks]
			}
			fmt.Fprintf(os.Stderr, "[curate] ブックマーク記事: %d件取得\n", len(articles))
			b, err := json.Marshal(toEnrichedArticleJSONList(articles))
			if err != nil {
				return "", err
			}
			return string(b), nil

		default:
			return "", fmt.Errorf("未知のツール: %s", name)
		}
	})
}
