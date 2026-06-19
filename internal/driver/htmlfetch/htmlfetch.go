package htmlfetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/samber/do/v2"

	adapterhtmlfetch "github.com/qli8racn/rss-feeder/internal/adapter/driver/htmlfetch"
)

const fetchTimeout = 15 * time.Second

// maxBodyBytes は取得するHTMLの最大バイト数。任意の外部URLを取得するため、
// 巨大なレスポンスでメモリを圧迫しないよう上限を設ける。
const maxBodyBytes = 5 * 1024 * 1024

type fetcher struct {
	client *http.Client
}

func NewFetcher(_ do.Injector) (adapterhtmlfetch.Fetcher, error) {
	return &fetcher{client: &http.Client{Timeout: fetchTimeout}}, nil
}

func (f *fetcher) Fetch(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("リクエストの作成に失敗しました %s: %w", url, err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTMLの取得に失敗しました %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTMLの取得に失敗しました %s: status %d", url, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		return "", fmt.Errorf("HTMLではないレスポンスです %s: Content-Type %q", url, contentType)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return "", fmt.Errorf("レスポンスの読み込みに失敗しました %s: %w", url, err)
	}

	return string(body), nil
}
