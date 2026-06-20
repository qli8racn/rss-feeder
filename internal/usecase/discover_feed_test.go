package usecase

import (
	"context"
	"errors"
	"testing"
)

type mockHTMLFetcher struct {
	fetchFn func(ctx context.Context, url string) (string, error)
}

func (m *mockHTMLFetcher) Fetch(ctx context.Context, url string) (string, error) {
	return m.fetchFn(ctx, url)
}

type mockFeedDiscoveryAgent struct {
	discoverFn func(ctx context.Context, url, html string) (string, error)
}

func (m *mockFeedDiscoveryAgent) Discover(ctx context.Context, url, html string) (string, error) {
	return m.discoverFn(ctx, url, html)
}

func TestDiscoverFeedUsecase_Success(t *testing.T) {
	uc := NewDiscoverFeedUsecase(
		&mockHTMLFetcher{fetchFn: func(_ context.Context, _ string) (string, error) {
			return "<html></html>", nil
		}},
		&mockFeedDiscoveryAgent{discoverFn: func(_ context.Context, _, _ string) (string, error) {
			return "https://example.com/feed.xml", nil
		}},
		&mockRSSReader{},
	)

	got, err := uc.Execute(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://example.com/feed.xml" {
		t.Errorf("got %q, want %q", got, "https://example.com/feed.xml")
	}
}

func TestDiscoverFeedUsecase_HTMLFetchError(t *testing.T) {
	fetchErr := errors.New("fetch error")
	uc := NewDiscoverFeedUsecase(
		&mockHTMLFetcher{fetchFn: func(_ context.Context, _ string) (string, error) {
			return "", fetchErr
		}},
		&mockFeedDiscoveryAgent{discoverFn: func(_ context.Context, _, _ string) (string, error) {
			t.Fatal("agent should not be called when HTML fetch fails")
			return "", nil
		}},
		&mockRSSReader{},
	)

	if _, err := uc.Execute(context.Background(), "https://example.com"); !errors.Is(err, fetchErr) {
		t.Errorf("expected fetch error, got %v", err)
	}
}

func TestDiscoverFeedUsecase_AgentError(t *testing.T) {
	agentErr := errors.New("agent error")
	uc := NewDiscoverFeedUsecase(
		&mockHTMLFetcher{fetchFn: func(_ context.Context, _ string) (string, error) {
			return "<html></html>", nil
		}},
		&mockFeedDiscoveryAgent{discoverFn: func(_ context.Context, _, _ string) (string, error) {
			return "", agentErr
		}},
		&mockRSSReader{},
	)

	if _, err := uc.Execute(context.Background(), "https://example.com"); !errors.Is(err, agentErr) {
		t.Errorf("expected agent error, got %v", err)
	}
}

func TestDiscoverFeedUsecase_CandidateNotValidFeed(t *testing.T) {
	validateErr := errors.New("invalid feed")
	uc := NewDiscoverFeedUsecase(
		&mockHTMLFetcher{fetchFn: func(_ context.Context, _ string) (string, error) {
			return "<html></html>", nil
		}},
		&mockFeedDiscoveryAgent{discoverFn: func(_ context.Context, _, _ string) (string, error) {
			return "https://example.com/not-a-feed", nil
		}},
		&mockRSSReader{err: validateErr},
	)

	if _, err := uc.Execute(context.Background(), "https://example.com"); !errors.Is(err, validateErr) {
		t.Errorf("expected validate error, got %v", err)
	}
}
