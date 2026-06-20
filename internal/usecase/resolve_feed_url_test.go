package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

type mockDiscoveryAgent struct {
	discoverFn func(ctx context.Context, url string) (string, error)
}

func (m *mockDiscoveryAgent) Discover(ctx context.Context, url string) (string, error) {
	return m.discoverFn(ctx, url)
}

func TestResolveFeedURLUsecase_Step1Success(t *testing.T) {
	uc := NewResolveFeedURLUsecase(
		&mockRSSReader{},
		&mockHTMLFetcher{fetchFn: func(_ context.Context, _ string) (string, error) {
			t.Fatal("htmlFetcher should not be called when step1 succeeds")
			return "", nil
		}},
		&mockDiscoveryAgent{discoverFn: func(_ context.Context, _ string) (string, error) {
			t.Fatal("agent should not be called when step1 succeeds")
			return "", nil
		}},
	)

	got, err := uc.Execute(context.Background(), "https://example.com/feed.xml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://example.com/feed.xml" {
		t.Errorf("got %q, want %q", got, "https://example.com/feed.xml")
	}
}

func TestResolveFeedURLUsecase_Step2Success(t *testing.T) {
	const htmlWithLink = `<html><head><link rel="alternate" type="application/rss+xml" href="/feed.xml"></head></html>`

	calls := 0
	rssReader := &mockRSSReader{fetchFn: func(_ context.Context, feedURL string) (string, []domain.Article, error) {
		calls++
		if calls == 1 {
			// 1回目: 入力URL自体のパース（失敗させてステップ2に進ませる）
			return "", nil, errors.New("not a feed")
		}
		// 2回目: 標準探索で見つかった候補の再検証
		if feedURL != "https://example.com/feed.xml" {
			t.Errorf("feedURL: got %q, want %q", feedURL, "https://example.com/feed.xml")
		}
		return "Example Feed", nil, nil
	}}

	uc := NewResolveFeedURLUsecase(
		rssReader,
		&mockHTMLFetcher{fetchFn: func(_ context.Context, _ string) (string, error) {
			return htmlWithLink, nil
		}},
		&mockDiscoveryAgent{discoverFn: func(_ context.Context, _ string) (string, error) {
			t.Fatal("agent should not be called when step2 succeeds")
			return "", nil
		}},
	)

	got, err := uc.Execute(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://example.com/feed.xml" {
		t.Errorf("got %q, want %q", got, "https://example.com/feed.xml")
	}
}

func TestResolveFeedURLUsecase_Step3Success(t *testing.T) {
	uc := NewResolveFeedURLUsecase(
		&mockRSSReader{err: errors.New("not a feed")},
		&mockHTMLFetcher{fetchFn: func(_ context.Context, _ string) (string, error) {
			return "<html></html>", nil
		}},
		&mockDiscoveryAgent{discoverFn: func(_ context.Context, url string) (string, error) {
			if url != "https://example.com" {
				t.Errorf("url: got %q, want %q", url, "https://example.com")
			}
			return "https://example.com/feed.xml", nil
		}},
	)

	got, err := uc.Execute(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://example.com/feed.xml" {
		t.Errorf("got %q, want %q", got, "https://example.com/feed.xml")
	}
}

func TestResolveFeedURLUsecase_AllFail(t *testing.T) {
	uc := NewResolveFeedURLUsecase(
		&mockRSSReader{err: errors.New("not a feed")},
		&mockHTMLFetcher{fetchFn: func(_ context.Context, _ string) (string, error) {
			return "<html></html>", nil
		}},
		&mockDiscoveryAgent{discoverFn: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("agent unavailable")
		}},
	)

	if _, err := uc.Execute(context.Background(), "https://example.com"); !errors.Is(err, ErrFeedNotFound) {
		t.Errorf("expected ErrFeedNotFound, got %v", err)
	}
}

func TestResolveFeedURLUsecase_InvalidURL(t *testing.T) {
	uc := NewResolveFeedURLUsecase(
		&mockRSSReader{},
		&mockHTMLFetcher{fetchFn: func(_ context.Context, _ string) (string, error) {
			t.Fatal("htmlFetcher should not be called for invalid URL")
			return "", nil
		}},
		&mockDiscoveryAgent{discoverFn: func(_ context.Context, _ string) (string, error) {
			t.Fatal("agent should not be called for invalid URL")
			return "", nil
		}},
	)

	if _, err := uc.Execute(context.Background(), "not-a-url"); err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

func TestFindFeedLink_StandardTag(t *testing.T) {
	htmlStr := `<html><head><link rel="alternate" type="application/rss+xml" href="https://example.com/feed.xml"></head></html>`

	got, ok := findFeedLink(htmlStr, "https://example.com")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "https://example.com/feed.xml" {
		t.Errorf("got %q, want %q", got, "https://example.com/feed.xml")
	}
}

func TestFindFeedLink_RelativeURL(t *testing.T) {
	htmlStr := `<html><head><link rel="alternate" type="application/atom+xml" href="/feed.atom"></head></html>`

	got, ok := findFeedLink(htmlStr, "https://example.com/blog/")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "https://example.com/feed.atom" {
		t.Errorf("got %q, want %q", got, "https://example.com/feed.atom")
	}
}

func TestFindFeedLink_AttributeOrderDoesNotMatter(t *testing.T) {
	htmlStr := `<html><head><link href="/feed.xml" type="application/rss+xml" rel="alternate"></head></html>`

	got, ok := findFeedLink(htmlStr, "https://example.com")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "https://example.com/feed.xml" {
		t.Errorf("got %q, want %q", got, "https://example.com/feed.xml")
	}
}

func TestFindFeedLink_NoTag(t *testing.T) {
	htmlStr := `<html><head><title>Example</title></head></html>`

	if _, ok := findFeedLink(htmlStr, "https://example.com"); ok {
		t.Error("expected ok=false")
	}
}
