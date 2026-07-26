package mcp

import (
	"context"
	"log/slog"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// AddFeedInput は add-feed ツールの入力。
type AddFeedInput struct {
	URL string `json:"url" jsonschema:"登録するRSS/AtomフィードのURL、またはフィードを探索する対象サイトのURL"`
}

// AddFeedOutput は add-feed ツールの出力。
type AddFeedOutput struct {
	ID      int64  `json:"id"`
	FeedURL string `json:"feed_url"`
	Title   string `json:"title,omitempty"`
	Fetched int    `json:"fetched"`
}

// AddFeedTool はフィードURLを解決・登録し、登録直後に1回記事取得する add-feed ツールの
// ハンドラを構築する。記事取得の失敗は登録自体の成功結果を変えない best-effort とする
// （internal/adapter/handler/web の AddFeedHandler と同じ方針）。
func AddFeedTool(
	addFeedUC *usecase.AddFeedUsecase,
	resolveFeedURLUC *usecase.ResolveFeedURLUsecase,
	fetchUC *usecase.FetchUsecase,
	logger *slog.Logger,
) mcpsdk.ToolHandlerFor[AddFeedInput, AddFeedOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, input AddFeedInput) (*mcpsdk.CallToolResult, AddFeedOutput, error) {
		resolvedURL, err := resolveFeedURLUC.Execute(ctx, input.URL)
		if err != nil {
			return nil, AddFeedOutput{}, err
		}

		feed, err := addFeedUC.Execute(ctx, resolvedURL)
		if err != nil {
			return nil, AddFeedOutput{}, err
		}

		// FetchResult は一部のフィードでエラーが起きても他フィードの保存結果を保持する値型のため、
		// エラー時も部分的に保存できた件数を fetched に反映する（0件で握り潰さない）。
		result, fetchErr := fetchUC.Execute(ctx, []string{resolvedURL})
		if fetchErr != nil {
			logger.Warn("add-feed: 登録直後の記事取得に失敗しました", "feed_url", resolvedURL, "error", fetchErr)
		}

		return nil, AddFeedOutput{ID: feed.ID, FeedURL: feed.FeedURL, Title: feed.Title, Fetched: result.TotalSaved()}, nil
	}
}
