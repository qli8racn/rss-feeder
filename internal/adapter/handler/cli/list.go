package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// cliListPerPage は CLI（list/search）でカテゴリ絞り込みを行う際の取得件数上限。
// CLI にはページネーションの概念がないため、実用上「全件」とみなせる十分大きな値を指定する。
const cliListPerPage = 10000

func NewListCommand(uc *usecase.ListUsecase) *cobra.Command {
	var flagAll, flagBookmarked bool
	var flagCategory string

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

			var (
				articles []domain.Article
				err      error
			)
			if flagCategory != "" {
				articles, _, err = uc.ExecuteFiltered(cmd.Context(), usecase.ListFilterOptions{
					Mode:     mode,
					Category: flagCategory,
					Sort:     "published_at",
					Order:    "desc",
					PerPage:  cliListPerPage,
				})
			} else {
				articles, err = uc.Execute(cmd.Context(), mode)
			}
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
	cmd.Flags().StringVar(&flagCategory, "category", "", "指定したカテゴリの記事のみ表示")
	cmd.MarkFlagsMutuallyExclusive("all", "bookmarked")

	return cmd
}
