package usecase

import (
	"context"
	"fmt"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
	adapterrss "github.com/qli8racn/rss-feeder/internal/adapter/driver/rss"
)

type FeedBackfillResult struct {
	FeedURL string
	Updated int64
	Err     error
}

type BackfillMetadataResult struct {
	Feeds []FeedBackfillResult
}

func (r BackfillMetadataResult) TotalUpdated() int64 {
	var n int64
	for _, f := range r.Feeds {
		n += f.Updated
	}
	return n
}

func (r BackfillMetadataResult) TotalErrors() int {
	n := 0
	for _, f := range r.Feeds {
		if f.Err != nil {
			n++
		}
	}
	return n
}

// BackfillMetadataUsecase は記事メタデータ拡充フェーズ以前に保存された既存記事に対し、
// 登録済みフィードを再取得して出版元・サムネイルを補完する（バックフィル）。
// RSSフィードは最新の一部記事しか配信しないため、フィードから既に外れた古い記事は対象外になる。
type BackfillMetadataUsecase struct {
	articleRepo articlerepo.Repository
	feedRepo    feedrepo.Repository
	rssReader   adapterrss.RSSReader
	userID      int64
}

func NewBackfillMetadataUsecase(
	articleRepo articlerepo.Repository,
	feedRepo feedrepo.Repository,
	rssReader adapterrss.RSSReader,
	userID int64,
) *BackfillMetadataUsecase {
	return &BackfillMetadataUsecase{
		articleRepo: articleRepo,
		feedRepo:    feedRepo,
		rssReader:   rssReader,
		userID:      userID,
	}
}

func (uc *BackfillMetadataUsecase) Execute(ctx context.Context) (BackfillMetadataResult, error) {
	feeds, err := uc.feedRepo.ListAll(ctx, uc.userID)
	if err != nil {
		return BackfillMetadataResult{}, fmt.Errorf("フィード一覧の取得に失敗: %w", err)
	}

	result := BackfillMetadataResult{Feeds: make([]FeedBackfillResult, 0, len(feeds))}
	for _, feed := range feeds {
		result.Feeds = append(result.Feeds, uc.backfillFeed(ctx, feed.FeedURL))
	}

	if n := result.TotalErrors(); n > 0 {
		return result, fmt.Errorf("%d 件のフィードで取得に失敗しました", n)
	}
	return result, nil
}

func (uc *BackfillMetadataUsecase) backfillFeed(ctx context.Context, feedURL string) FeedBackfillResult {
	_, articles, err := uc.rssReader.Fetch(ctx, feedURL)
	if err != nil {
		return FeedBackfillResult{FeedURL: feedURL, Err: fmt.Errorf("フェッチ失敗: %w", err)}
	}

	updates := make([]articlerepo.MetadataUpdate, len(articles))
	for i, a := range articles {
		updates[i] = articlerepo.MetadataUpdate{
			URL:          a.URL,
			Publisher:    a.Publisher,
			ThumbnailURL: a.ThumbnailURL,
		}
	}

	updated, err := uc.articleRepo.UpdateMetadataBatch(ctx, updates)
	if err != nil {
		return FeedBackfillResult{FeedURL: feedURL, Err: fmt.Errorf("更新失敗: %w", err)}
	}
	return FeedBackfillResult{FeedURL: feedURL, Updated: updated}
}
