package domain_test

import (
	"testing"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

func TestArticle_ToggleBookmark(t *testing.T) {
	a := domain.Article{Bookmarked: false}
	a.ToggleBookmark()
	if !a.Bookmarked {
		t.Error("expected Bookmarked to be true after first toggle")
	}
	a.ToggleBookmark()
	if a.Bookmarked {
		t.Error("expected Bookmarked to be false after second toggle")
	}
}

func TestArticle_MarkAsRead(t *testing.T) {
	a := domain.Article{Read: false}
	a.MarkAsRead()
	if !a.Read {
		t.Error("expected Read to be true after MarkAsRead")
	}
}
