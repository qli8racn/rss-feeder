package usecase

import (
	"context"
	"fmt"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
	adapterrss "github.com/qli8racn/rss-feeder/internal/adapter/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

type FetchResult struct {
	Saved   int
	Skipped int
	Errors  int
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
	var result FetchResult
	total := len(feedURLs)

	for idx, feedURL := range feedURLs {
		fmt.Printf("[%d/%d] %s\n", idx+1, total, feedURL)

		feedTitle, articles, err := uc.rssReader.Fetch(ctx, feedURL)
		if err != nil {
			fmt.Printf("  エラー: %v\n", err)
			result.Errors++
			continue
		}

		feedID, err := uc.feedRepo.Save(ctx, domain.Feed{FeedURL: feedURL, Title: feedTitle})
		if err != nil {
			fmt.Printf("  エラー: フィード保存に失敗: %v\n", err)
			result.Errors++
			continue
		}

		feedSaved, feedSkipped := 0, 0
		for _, article := range articles {
			article.FeedID = feedID
			if err := uc.articleRepo.Save(ctx, article); err != nil {
				feedSkipped++
				continue
			}
			fmt.Printf("  + %s  %s\n", article.Title, article.URL)
			feedSaved++
		}
		if feedSkipped > 0 {
			fmt.Printf("  スキップ: %d 件\n", feedSkipped)
		}
		result.Saved += feedSaved
		result.Skipped += feedSkipped
	}

	fmt.Printf("\n完了: 新規 %d 件 / スキップ %d 件 / エラー %d 件\n", result.Saved, result.Skipped, result.Errors)
	return result, nil
}
