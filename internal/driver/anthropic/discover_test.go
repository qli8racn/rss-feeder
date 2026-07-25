package anthropic

import (
	"testing"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

func TestToFeedJSONList_ConvertsFields(t *testing.T) {
	feeds := []domain.Feed{
		{FeedURL: "https://example.com/feed.xml", Title: "Example Blog"},
		{FeedURL: "https://another.com/rss", Title: "Another Site"},
	}

	got := toFeedJSONList(feeds)

	if len(got) != 2 {
		t.Fatalf("len: got %d, want 2", len(got))
	}
	if got[0].FeedURL != "https://example.com/feed.xml" || got[0].Title != "Example Blog" {
		t.Errorf("got[0]: %+v", got[0])
	}
	if got[1].FeedURL != "https://another.com/rss" || got[1].Title != "Another Site" {
		t.Errorf("got[1]: %+v", got[1])
	}
}

func TestToFeedJSONList_EmptyInput(t *testing.T) {
	got := toFeedJSONList([]domain.Feed{})
	if len(got) != 0 {
		t.Errorf("expected empty slice, got len %d", len(got))
	}
}
