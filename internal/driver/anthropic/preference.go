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

type preferenceAgent struct {
	client messageCreator
	reader articlerepo.Repository
}

func NewPreferenceAgent(i do.Injector) (adapteranthropic.PreferenceAgent, error) {
	client := newAnthropicClient()
	return &preferenceAgent{
		client: &client.Messages,
		reader: do.MustInvoke[articlerepo.Repository](i),
	}, nil
}

// maxBookmarkedArticles は preferenceAgent が一度に分析対象とするブックマーク記事数の上限。
// 上限を設けず全件送信すると、ブックマークの増加に比例して入力トークンが膨張するため制限する。
const maxBookmarkedArticles = 50

func (a *preferenceAgent) Run(ctx context.Context) (string, error) {
	tools := []anthropic.ToolUnionParam{
		{OfTool: &anthropic.ToolParam{
			Name:        "fetch_bookmarked",
			Description: anthropic.String("ブックマーク済みの記事を取得する。"),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"limit": map[string]any{"type": "integer", "description": fmt.Sprintf("取得件数の上限（省略時は%d件、最大%d件）", maxBookmarkedArticles, maxBookmarkedArticles)},
				},
			},
		}},
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeOpus4_8,
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{{
			Text: "あなたはユーザーの技術的な興味を分析するアシスタントです。" +
				"ブックマークされた記事のタイトルとURLから、ユーザーがどんな技術・トピックに関心を持っているかを洞察してください。",
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(
				"ブックマークした記事からユーザーの技術的な興味・趣向を分析してください。" +
					"どんな技術分野・トピックに関心があるか、傾向をまとめてください。",
			)),
		},
		Tools: tools,
		Thinking: anthropic.ThinkingConfigParamUnion{
			OfAdaptive: &anthropic.ThinkingConfigAdaptiveParam{},
		},
	}

	return runAgentLoopWithUsageLog(ctx, a.client, params, func(_, inputJSON string) (string, error) {
		limit, err := parseLimitInput(inputJSON, maxBookmarkedArticles, maxBookmarkedArticles)
		if err != nil {
			return "", err
		}

		articles, err := a.reader.FindBookmarked(ctx)
		if err != nil {
			return "", err
		}
		if len(articles) == 0 {
			return "ブックマークされた記事がありません。bin/rss-feeder bookmark <id> で登録してください。", nil
		}
		if len(articles) > limit {
			articles = articles[:limit]
		}
		b, err := json.Marshal(toArticleJSONList(articles))
		if err != nil {
			return "", err
		}
		return string(b), nil
	})
}
