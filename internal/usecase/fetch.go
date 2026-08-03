package usecase

import (
	"context"
	"errors"
	"fmt"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
	adapterrss "github.com/qli8racn/rss-feeder/internal/adapter/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

type FeedFetchResult struct {
	FeedURL string
	Saved   []domain.Article
	Skipped int
	Err     error
}

type FetchResult struct {
	Feeds []FeedFetchResult
}

func (r FetchResult) TotalSaved() int {
	n := 0
	for _, f := range r.Feeds {
		n += len(f.Saved)
	}
	return n
}

func (r FetchResult) TotalSkipped() int {
	n := 0
	for _, f := range r.Feeds {
		n += f.Skipped
	}
	return n
}

func (r FetchResult) TotalErrors() int {
	n := 0
	for _, f := range r.Feeds {
		if f.Err != nil {
			n++
		}
	}
	return n
}

type FetchUsecase struct {
	articleRepo articlerepo.Repository
	feedRepo    feedrepo.Repository
	rssReader   adapterrss.RSSReader
	userID      int64
}

func NewFetchUsecase(
	articleRepo articlerepo.Repository,
	feedRepo feedrepo.Repository,
	rssReader adapterrss.RSSReader,
	userID int64,
) *FetchUsecase {
	return &FetchUsecase{
		articleRepo: articleRepo,
		feedRepo:    feedRepo,
		rssReader:   rssReader,
		userID:      userID,
	}
}

func (uc *FetchUsecase) Execute(ctx context.Context, feedURLs []string) (FetchResult, error) {
	result := FetchResult{Feeds: make([]FeedFetchResult, 0, len(feedURLs))}

	for _, feedURL := range feedURLs {
		result.Feeds = append(result.Feeds, uc.fetchFeed(ctx, feedURL))
	}

	if n := result.TotalErrors(); n > 0 {
		return result, fmt.Errorf("%d 件のフィードで取得に失敗しました", n)
	}
	return result, nil
}

// ExecuteAll は登録済みの全フィードを取得して DB に保存する。
// CLI（fetch コマンド）と Web API（POST /api/articles/fetch）はともに「登録済み全フィードの取得」を
// 行うため、フィード一覧の取得から URL 抽出までをここに集約する。
func (uc *FetchUsecase) ExecuteAll(ctx context.Context) (FetchResult, error) {
	feeds, err := uc.feedRepo.ListAll(ctx, uc.userID)
	if err != nil {
		return FetchResult{}, fmt.Errorf("フィード一覧の取得に失敗: %w", err)
	}

	urls := make([]string, len(feeds))
	for i, f := range feeds {
		urls[i] = f.FeedURL
	}

	return uc.Execute(ctx, urls)
}

func (uc *FetchUsecase) fetchFeed(ctx context.Context, feedURL string) FeedFetchResult {
	feedTitle, articles, err := uc.rssReader.Fetch(ctx, feedURL)
	if err != nil {
		return FeedFetchResult{FeedURL: feedURL, Err: fmt.Errorf("フェッチ失敗: %w", err)}
	}

	feedID, err := uc.feedRepo.Save(ctx, domain.Feed{FeedURL: feedURL, Title: feedTitle}, uc.userID)
	if err != nil {
		return FeedFetchResult{FeedURL: feedURL, Err: fmt.Errorf("フィード保存失敗: %w", err)}
	}

	var saved []domain.Article
	skipped := 0

	for _, article := range articles {
		article.FeedID = feedID
		if err := uc.articleRepo.Save(ctx, article); err != nil {
			if errors.Is(err, articlerepo.ErrDuplicate) {
				skipped++
				continue
			}
			return FeedFetchResult{FeedURL: feedURL, Saved: saved, Skipped: skipped, Err: fmt.Errorf("記事保存失敗: %w", err)}
		}
		saved = append(saved, article)
	}

	return FeedFetchResult{FeedURL: feedURL, Saved: saved, Skipped: skipped}
}
