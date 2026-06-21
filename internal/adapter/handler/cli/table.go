package cli

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

func printArticleTable(articles []domain.Article) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tタイトル\t出版元\tカテゴリ\t公開日時\t既読\tお気に入り")
	fmt.Fprintln(w, "---\t--------------------------------\t--------------------\t----------\t--------------------\t----\t----------")
	for _, a := range articles {
		read := "-"
		if a.Read {
			read = "✓"
		}
		bookmark := "-"
		if a.Bookmarked {
			bookmark = "★"
		}
		published := "-"
		if !a.PublishedAt.IsZero() {
			published = a.PublishedAt.Local().Format(time.DateTime)
		}
		publisher := a.Publisher
		if publisher == "" {
			publisher = a.FeedURL
		}
		category := a.Category
		if category == "" {
			category = "-"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n", a.ID, a.Title, publisher, category, published, read, bookmark)
	}
	w.Flush()
}
