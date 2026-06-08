package handler

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func NewSearchCommand(uc *usecase.SearchUsecase) *cobra.Command {
	var flagBookmarked bool

	cmd := &cobra.Command{
		Use:   "search <keyword>",
		Short: "キーワードで記事を全文検索する",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			articles, err := uc.Execute(cmd.Context(), args[0], flagBookmarked)
			if err != nil {
				return err
			}

			if len(articles) == 0 {
				fmt.Println("該当記事が見つかりませんでした")
				return nil
			}

			printArticleTable(articles)
			fmt.Printf("\n該当: %d 件\n", len(articles))
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagBookmarked, "bookmarked", false, "お気に入り記事のみを検索対象にする")

	return cmd
}
