package handler

import (
	"fmt"

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

			articles, err := uc.Execute(cmd.Context(), mode)
			if err != nil {
				return err
			}

			if len(articles) == 0 {
				fmt.Println("表示する記事がありません。")
				return nil
			}

			printArticleTable(articles)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagAll, "all", false, "全件表示")
	cmd.Flags().BoolVar(&flagBookmarked, "bookmarked", false, "お気に入りのみ表示")
	cmd.MarkFlagsMutuallyExclusive("all", "bookmarked")

	return cmd
}
