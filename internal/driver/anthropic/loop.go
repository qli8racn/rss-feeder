package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

type articleJSON struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	FeedURL     string `json:"feed_url,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
}

func toArticleJSONList(articles []domain.Article) []articleJSON {
	result := make([]articleJSON, len(articles))
	for i, a := range articles {
		pub := ""
		if !a.PublishedAt.IsZero() {
			pub = a.PublishedAt.Format(time.RFC3339)
		}
		result[i] = articleJSON{Title: a.Title, URL: a.URL, FeedURL: a.FeedURL, PublishedAt: pub}
	}
	return result
}

type toolHandler func(name, inputJSON string) (string, error)

// parseLimitInput はツール呼び出しの inputJSON から "limit" フィールドを取り出す。
// 0以下、または未指定（inputJSON が空）の場合は defaultLimit を使う。
// maxLimit > 0 の場合、defaultLimit・指定値の双方をこの値に切り詰める（0以下なら上限なし）。
func parseLimitInput(inputJSON string, defaultLimit, maxLimit int) (int, error) {
	var input struct {
		Limit int `json:"limit"`
	}
	if inputJSON != "" {
		if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
			return 0, fmt.Errorf("invalid tool input: %w", err)
		}
	}

	limit := input.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if maxLimit > 0 && limit > maxLimit {
		limit = maxLimit
	}
	return limit, nil
}

func runAgentLoop(ctx context.Context, client anthropic.Client, params anthropic.MessageNewParams, handler toolHandler) (string, error) {
	messages := params.Messages
	for {
		params.Messages = messages
		resp, err := client.Messages.New(ctx, params)
		if err != nil {
			return "", err
		}
		messages = append(messages, resp.ToParam())

		if resp.StopReason != anthropic.StopReasonToolUse {
			for _, block := range resp.Content {
				if tb, ok := block.AsAny().(anthropic.TextBlock); ok {
					return tb.Text, nil
				}
			}
			break
		}

		var results []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			tu, ok := block.AsAny().(anthropic.ToolUseBlock)
			if !ok {
				continue
			}
			output, err := handler(tu.Name, tu.JSON.Input.Raw())
			if err != nil {
				results = append(results, anthropic.NewToolResultBlock(tu.ID, err.Error(), true))
				continue
			}
			results = append(results, anthropic.NewToolResultBlock(tu.ID, output, false))
		}
		messages = append(messages, anthropic.NewUserMessage(results...))
	}
	return "", nil
}
