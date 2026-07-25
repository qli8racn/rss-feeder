package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/samber/do/v2"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

type discoverAgent struct {
	client     messageCreator
	articleRepo articlerepo.Repository
	feedRepo    feedrepo.Repository
}

func NewDiscoverAgent(i do.Injector) (adapteranthropic.DiscoverAgent, error) {
	client := newAnthropicClient()
	return &discoverAgent{
		client:      &client.Messages,
		articleRepo: do.MustInvoke[articlerepo.Repository](i),
		feedRepo:    do.MustInvoke[feedrepo.Repository](i),
	}, nil
}


type feedJSON struct {
	FeedURL string `json:"feed_url"`
	Title   string `json:"title"`
}

func toFeedJSONList(feeds []domain.Feed) []feedJSON {
	result := make([]feedJSON, len(feeds))
	for i, f := range feeds {
		result[i] = feedJSON{FeedURL: f.FeedURL, Title: f.Title}
	}
	return result
}

func (a *discoverAgent) Run(ctx context.Context) (string, error) {
	tools := []anthropic.ToolUnionParam{
		{OfTool: &anthropic.ToolParam{
			Name:        "fetch_bookmarked_articles",
			Description: anthropic.String("ブックマーク済みの記事を取得する。ユーザーの技術的興味・趣向の把握に使用する。"),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "fetch_registered_feeds",
			Description: anthropic.String("現在登録済みの RSS フィード一覧を取得する。重複推薦を避けるために使用する。"),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{},
			},
		}},
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeOpus4_8,
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{{
			Text: "あなたは RSS フィード推薦アシスタントです。" +
				"ブックマーク済み記事からユーザーの技術的興味・趣向を把握し、" +
				"登録済みフィードと重複しない新しい RSS フィードを 3〜5 件推薦してください。" +
				"各フィードに RSS フィード URL（または公式サイト URL）と推薦理由（1〜2文）を付与し、" +
				"Markdown 形式で日本語で出力してください。" +
				"実在する RSS フィードのみ推薦してください。",
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(
				"ブックマーク記事と登録済みフィードを取得して、新しく購読すべき RSS フィードを推薦してください。",
			)),
		},
		Tools: tools,
		Thinking: anthropic.ThinkingConfigParamUnion{
			OfAdaptive: &anthropic.ThinkingConfigAdaptiveParam{},
		},
	}

	fmt.Fprintln(os.Stderr, "[discover] 開始しています...")
	return runAgentLoopWithUsageLog(ctx, a.client, params, func(name, _ string) (string, error) {
		switch name {
		case "fetch_bookmarked_articles":
			fmt.Fprintln(os.Stderr, "[discover] ブックマーク記事を取得中...")
			articles, err := a.articleRepo.FindBookmarked(ctx)
			if err != nil {
				return "", err
			}
			if len(articles) == 0 {
				fmt.Fprintln(os.Stderr, "[discover] ブックマーク記事: 0件（趣向情報なしで推薦）")
				return "ブックマークされた記事がありません。趣向情報なしで推薦します。", nil
			}
			if len(articles) > maxBookmarks {
				articles = articles[:maxBookmarks]
			}
			fmt.Fprintf(os.Stderr, "[discover] ブックマーク記事: %d件取得\n", len(articles))
			b, err := json.Marshal(toEnrichedArticleJSONList(articles))
			if err != nil {
				return "", err
			}
			return string(b), nil

		case "fetch_registered_feeds":
			fmt.Fprintln(os.Stderr, "[discover] 登録済みフィードを取得中...")
			feeds, err := a.feedRepo.ListAll(ctx)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(os.Stderr, "[discover] 登録済みフィード: %d件取得\n", len(feeds))
			b, err := json.Marshal(toFeedJSONList(feeds))
			if err != nil {
				return "", err
			}
			return string(b), nil

		default:
			return "", fmt.Errorf("未知のツール: %s", name)
		}
	})
}
