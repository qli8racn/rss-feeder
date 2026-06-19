package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/qli8racn/rss-feeder/internal/adapter/driver/feeddiscovery"
	"github.com/qli8racn/rss-feeder/internal/adapter/driver/htmlfetch"
	adapterrss "github.com/qli8racn/rss-feeder/internal/adapter/driver/rss"
)

// ErrFeedNotFound は入力URLからフィードURLを解決できなかった場合に返される。
var ErrFeedNotFound = errors.New("feed url could not be resolved")

// stepThreeTimeout はAIフォールバック（ステップ3）呼び出しのタイムアウト。
// 呼び出し元（handler）が設定する全体タイムアウトの残り予算内でネストする。
const stepThreeTimeout = 30 * time.Second

// ResolveFeedURLUsecase は入力URLからRSS/AtomフィードのURLを解決する。
// 1. 入力URL自体をRSS/Atomとしてパースできるか試す
// 2. <link rel="alternate"> をHTMLから探す（標準探索）
// 3. 1, 2 で見つからない場合のみ feeddiscovery.Agent（AIフォールバック）に問い合わせる
type ResolveFeedURLUsecase struct {
	rssReader   adapterrss.RSSReader
	htmlFetcher htmlfetch.Fetcher
	agent       feeddiscovery.Agent
}

func NewResolveFeedURLUsecase(
	rssReader adapterrss.RSSReader,
	htmlFetcher htmlfetch.Fetcher,
	agent feeddiscovery.Agent,
) *ResolveFeedURLUsecase {
	return &ResolveFeedURLUsecase{rssReader: rssReader, htmlFetcher: htmlFetcher, agent: agent}
}

func (uc *ResolveFeedURLUsecase) Execute(ctx context.Context, inputURL string) (string, error) {
	if _, err := url.ParseRequestURI(inputURL); err != nil {
		return "", fmt.Errorf("URLの形式が正しくありません: %w", err)
	}

	if _, _, err := uc.rssReader.Fetch(ctx, inputURL); err == nil {
		return inputURL, nil
	}

	if pageHTML, err := uc.htmlFetcher.Fetch(ctx, inputURL); err != nil {
		log.Printf("[resolve_feed_url] HTMLの取得に失敗しました %s: %v", inputURL, err)
	} else if candidate, ok := findFeedLink(pageHTML, inputURL); ok {
		if _, _, err := uc.rssReader.Fetch(ctx, candidate); err == nil {
			return candidate, nil
		} else {
			log.Printf("[resolve_feed_url] 標準探索で見つかった候補がRSS/Atomとして解釈できませんでした %s: %v", candidate, err)
		}
	}

	stepCtx, cancel := context.WithTimeout(ctx, stepThreeTimeout)
	defer cancel()
	if candidate, err := uc.agent.Discover(stepCtx, inputURL); err != nil {
		log.Printf("[resolve_feed_url] AIフォールバックに失敗しました %s: %v", inputURL, err)
	} else if candidate != "" {
		return candidate, nil
	}

	return "", ErrFeedNotFound
}

// findFeedLink は html から <link rel="alternate" type="application/rss+xml|application/atom+xml">
// を抽出し、baseURL を基準に相対URLを解決する。golang.org/x/net/html を使用。外部I/Oを持たない純粋関数。
func findFeedLink(htmlStr, baseURL string) (string, bool) {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", false
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", false
	}

	var found string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != "" || n == nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "link" {
			var rel, typ, href string
			for _, attr := range n.Attr {
				switch attr.Key {
				case "rel":
					rel = attr.Val
				case "type":
					typ = attr.Val
				case "href":
					href = attr.Val
				}
			}
			if href != "" && isAlternateRel(rel) && isFeedType(typ) {
				if resolved, err := base.Parse(href); err == nil {
					found = resolved.String()
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	if found == "" {
		return "", false
	}
	return found, true
}

func isAlternateRel(rel string) bool {
	for _, token := range strings.Fields(rel) {
		if token == "alternate" {
			return true
		}
	}
	return false
}

func isFeedType(typ string) bool {
	return typ == "application/rss+xml" || typ == "application/atom+xml"
}
