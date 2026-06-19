package htmlfetch

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const sampleHTML = `<!DOCTYPE html><html><head><title>Example</title></head><body>hello</body></html>`

func newTestFetcher() *fetcher {
	f, _ := NewFetcher(nil)
	return f.(*fetcher)
}

func TestFetch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if _, err := w.Write([]byte(sampleHTML)); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer srv.Close()

	f := newTestFetcher()
	html, err := f.Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if html != sampleHTML {
		t.Errorf("html: got %q, want %q", html, sampleHTML)
	}
}

// TestFetch_ContentTypeWithCharset は実運用上ほぼ必須な `Content-Type: text/html; charset=utf-8`
// のようなパラメータ付きヘッダでも完全一致判定で誤って拒否しないことを確認する。
func TestFetch_ContentTypeWithCharset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write([]byte(sampleHTML)); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer srv.Close()

	f := newTestFetcher()
	html, err := f.Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if html != sampleHTML {
		t.Errorf("html: got %q, want %q", html, sampleHTML)
	}
}

func TestFetch_NonHTMLContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		if _, err := w.Write([]byte("<rss></rss>")); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer srv.Close()

	f := newTestFetcher()
	if _, err := f.Fetch(context.Background(), srv.URL); err == nil {
		t.Error("expected error for non-HTML content type, got nil")
	}
}

// TestFetch_TruncatesOversizedBody は maxBodyBytes を超えるレスポンスが
// エラーにならず、上限ちょうどのサイズに切り詰められることを確認する。
func TestFetch_TruncatesOversizedBody(t *testing.T) {
	oversized := bytes.Repeat([]byte("a"), maxBodyBytes+1024)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if _, err := w.Write(oversized); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer srv.Close()

	f := newTestFetcher()
	html, err := f.Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(html) != maxBodyBytes {
		t.Errorf("len(html): got %d, want %d", len(html), maxBodyBytes)
	}
}

func TestFetch_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	f := newTestFetcher()
	if _, err := f.Fetch(context.Background(), srv.URL); err == nil {
		t.Error("expected error for 500 response, got nil")
	}
}
