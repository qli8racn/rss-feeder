package htmlfetch

import "context"

// Fetcher は指定 URL の HTML を取得する。
type Fetcher interface {
	Fetch(ctx context.Context, url string) (html string, err error)
}
