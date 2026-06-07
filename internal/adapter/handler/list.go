package handler

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func NewListCommand(uc *usecase.ListUsecase) *cobra.Command {
	var flagAll, flagBookmarked bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "保存済み記事を一覧表示する",
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := usecase.ListModeUnread
			switch {
			case flagAll:
				mode = usecase.ListModeAll
			case flagBookmarked:
				mode = usecase.ListModeBookmarked
			}

			articles, err := uc.Execute(context.Background(), mode)
			if err != nil {
				return err
			}

			if len(articles) == 0 {
				fmt.Println("表示する記事がありません。")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tタイトル\t公開日時\t既読\tお気に入り")
			fmt.Fprintln(w, "---\t--------------------------------\t--------------------\t----\t----------")
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
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", a.ID, a.Title, published, read, bookmark)
			}
			w.Flush()
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagAll, "all", false, "全件表示")
	cmd.Flags().BoolVar(&flagBookmarked, "bookmarked", false, "お気に入りのみ表示")
	cmd.MarkFlagsMutuallyExclusive("all", "bookmarked")

	return cmd
}
