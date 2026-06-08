package anthropic

import (
	"context"
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
)

type PreferenceAgent struct {
	client anthropic.Client
	reader articlerepo.Repository
}

func NewPreferenceAgent(r articlerepo.Repository) *PreferenceAgent {
	return &PreferenceAgent{
		client: anthropic.NewClient(),
		reader: r,
	}
}

func (a *PreferenceAgent) Run(ctx context.Context) (string, error) {
	tools := []anthropic.ToolUnionParam{
		{OfTool: &anthropic.ToolParam{
			Name:        "fetch_bookmarked",
			Description: anthropic.String("ブックマーク済みの記事を全件取得する。"),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{},
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

	return runAgentLoop(ctx, a.client, params, func(_, _ string) (string, error) {
		articles, err := a.reader.FindBookmarked(ctx)
		if err != nil {
			return "", err
		}
		if len(articles) == 0 {
			return "ブックマークされた記事がありません。bin/rss-feeder bookmark <id> で登録してください。", nil
		}
		b, err := json.Marshal(toArticleJSONList(articles))
		if err != nil {
			return "", err
		}
		return string(b), nil
	})
}
