package anthropic

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/samber/do/v2"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
)

// maxHTMLRunes は <head> が見つからない場合に送信するHTML全体の最大文字数。
// 入力トークンを抑えるため、これを超える部分は末尾を切り詰める。
const maxHTMLRunes = 20000

type feedDiscoveryAgent struct {
	client messageCreator
	logger *slog.Logger
}

func NewFeedDiscoveryAgent(i do.Injector) (adapteranthropic.FeedDiscoveryAgent, error) {
	client := newAnthropicClient()
	return &feedDiscoveryAgent{
		client: &client.Messages,
		logger: do.MustInvoke[*slog.Logger](i),
	}, nil
}

func (a *feedDiscoveryAgent) Discover(ctx context.Context, pageURL string, html string) (string, error) {
	start := time.Now()
	model := anthropic.ModelClaudeHaiku4_5

	input := extractHead(html)
	if input == "" {
		input = truncateRunes(html, maxHTMLRunes)
	}

	params := anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: 256,
		System: []anthropic.TextBlockParam{{
			Text: "あなたはHTMLからRSS/AtomフィードへのリンクURLを1つ特定するアシスタントです。" +
				"<link>タグや本文中のヒントからフィードURLを推測してください。" +
				"出力は説明文を含まず、見つかったURLのみを返してください。見つからない場合は NONE のみを返してください。",
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(fmt.Sprintf("ページURL: %s\n\nHTML:\n%s", pageURL, input))),
		},
	}

	resp, err := a.client.New(ctx, params)
	if err != nil {
		return "", err
	}
	logUsage(a.logger, model, resp.Usage, time.Since(start))

	for _, block := range resp.Content {
		if tb, ok := block.AsAny().(anthropic.TextBlock); ok {
			candidate := strings.TrimSpace(tb.Text)
			if candidate == "" || candidate == "NONE" {
				return "", fmt.Errorf("フィードURLが見つかりませんでした %s", pageURL)
			}
			if _, err := url.ParseRequestURI(candidate); err != nil {
				return "", fmt.Errorf("Claudeの応答が有効なURL形式ではありません %q: %w", candidate, err)
			}
			return candidate, nil
		}
	}

	return "", fmt.Errorf("テキスト応答が見つかりませんでした")
}

// extractHead は html から <head>...</head> 部分を抽出する（大文字小文字を区別しない）。
// 見つからない場合は空文字列を返す。
func extractHead(html string) string {
	lower := strings.ToLower(html)
	start := strings.Index(lower, "<head")
	if start == -1 {
		return ""
	}
	end := strings.Index(lower[start:], "</head>")
	if end == -1 {
		return ""
	}
	return html[start : start+end+len("</head>")]
}
