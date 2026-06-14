package rss

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const rss20XML = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test RSS Feed</title>
    <link>https://example.com</link>
    <item>
      <title>Article One</title>
      <link>https://example.com/one</link>
      <description>Content of article one</description>
      <pubDate>Mon, 02 Jan 2006 15:04:05 +0000</pubDate>
    </item>
    <item>
      <title>Article Two</title>
      <link>https://example.com/two</link>
      <description>Content of article two</description>
      <pubDate>Tue, 03 Jan 2006 12:00:00 +0000</pubDate>
    </item>
  </channel>
</rss>`

const rss20WithThumbnailXML = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:media="http://search.yahoo.com/mrss/">
  <channel>
    <title>Test RSS Feed</title>
    <link>https://example.com</link>
    <item>
      <title>Article With Media Thumbnail</title>
      <link>https://example.com/media-thumb</link>
      <description>Content</description>
      <media:thumbnail url="https://example.com/media-thumb.jpg"/>
    </item>
    <item>
      <title>Article With Enclosure</title>
      <link>https://example.com/enclosure</link>
      <description>Content</description>
      <enclosure url="https://example.com/enclosure.jpg" type="image/jpeg" length="1234"/>
    </item>
    <item>
      <title>Article Without Thumbnail</title>
      <link>https://example.com/none</link>
      <description>Content</description>
    </item>
  </channel>
</rss>`

const atomXML = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Test Atom Feed</title>
  <entry>
    <title>Atom Article</title>
    <link href="https://example.com/atom-one"/>
    <summary>Atom article summary</summary>
    <updated>2024-01-15T10:00:00Z</updated>
  </entry>
</feed>`

func newTestReader() *reader {
	r, _ := NewReader(nil)
	return r.(*reader)
}

func TestFetch_RSS20(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(rss20XML))
	}))
	defer srv.Close()

	r := newTestReader()
	title, articles, err := r.Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != "Test RSS Feed" {
		t.Errorf("title: got %q, want %q", title, "Test RSS Feed")
	}
	if len(articles) != 2 {
		t.Fatalf("articles count: got %d, want 2", len(articles))
	}
	if articles[0].Title != "Article One" {
		t.Errorf("articles[0].Title: got %q, want %q", articles[0].Title, "Article One")
	}
	if articles[0].URL != "https://example.com/one" {
		t.Errorf("articles[0].URL: got %q, want %q", articles[0].URL, "https://example.com/one")
	}
	if articles[0].PublishedAt.IsZero() {
		t.Error("articles[0].PublishedAt should not be zero")
	}
}

func TestFetch_Atom(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		w.Write([]byte(atomXML))
	}))
	defer srv.Close()

	r := newTestReader()
	title, articles, err := r.Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != "Test Atom Feed" {
		t.Errorf("title: got %q, want %q", title, "Test Atom Feed")
	}
	if len(articles) != 1 {
		t.Fatalf("articles count: got %d, want 1", len(articles))
	}
	if articles[0].Title != "Atom Article" {
		t.Errorf("articles[0].Title: got %q, want %q", articles[0].Title, "Atom Article")
	}
	if articles[0].URL != "https://example.com/atom-one" {
		t.Errorf("articles[0].URL: got %q, want %q", articles[0].URL, "https://example.com/atom-one")
	}
	if articles[0].PublishedAt.IsZero() {
		t.Error("articles[0].PublishedAt should not be zero")
	}
}

func TestFetch_Thumbnail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(rss20WithThumbnailXML))
	}))
	defer srv.Close()

	r := newTestReader()
	title, articles, err := r.Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(articles) != 3 {
		t.Fatalf("articles count: got %d, want 3", len(articles))
	}

	if articles[0].Publisher != title {
		t.Errorf("Publisher: got %q, want %q", articles[0].Publisher, title)
	}

	if got := articles[0].ThumbnailURL; got != "https://example.com/media-thumb.jpg" {
		t.Errorf("media:thumbnail: got %q", got)
	}
	if got := articles[1].ThumbnailURL; got != "https://example.com/enclosure.jpg" {
		t.Errorf("enclosure: got %q", got)
	}
	if got := articles[2].ThumbnailURL; got != "" {
		t.Errorf("no thumbnail: got %q, want empty", got)
	}
}

func TestFetch_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	r := newTestReader()
	_, _, err := r.Fetch(context.Background(), srv.URL)
	if err == nil {
		t.Error("expected error for 500 response, got nil")
	}
}

func TestFetch_InvalidXML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("this is not xml"))
	}))
	defer srv.Close()

	r := newTestReader()
	_, _, err := r.Fetch(context.Background(), srv.URL)
	if err == nil {
		t.Error("expected error for invalid XML, got nil")
	}
}
