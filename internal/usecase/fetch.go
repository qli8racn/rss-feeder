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
}

func NewFetchUsecase(
	articleRepo articlerepo.Repository,
	feedRepo feedrepo.Repository,
	rssReader adapterrss.RSSReader,
) *FetchUsecase {
	return &FetchUsecase{
		articleRepo: articleRepo,
		feedRepo:    feedRepo,
		rssReader:   rssReader,
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

func (uc *FetchUsecase) fetchFeed(ctx context.Context, feedURL string) FeedFetchResult {
	feedTitle, articles, err := uc.rssReader.Fetch(ctx, feedURL)
	if err != nil {
		return FeedFetchResult{Err: fmt.Errorf("フェッチ失敗: %w", err)}
	}

	feedID, err := uc.feedRepo.Save(ctx, domain.Feed{FeedURL: feedURL, Title: feedTitle})
	if err != nil {
		return FeedFetchResult{Err: fmt.Errorf("フィード保存失敗: %w", err)}
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
			return FeedFetchResult{Saved: saved, Skipped: skipped, Err: fmt.Errorf("記事保存失敗: %w", err)}
		}
		saved = append(saved, article)
	}

	return FeedFetchResult{Saved: saved, Skipped: skipped}
}
