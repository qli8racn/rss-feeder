package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func NewCategoriesCommand(uc *usecase.ListCategoriesUsecase) *cobra.Command {
	return &cobra.Command{
		Use:   "categories",
		Short: "記事に付与済みのカテゴリ一覧を表示する",
		RunE: func(cmd *cobra.Command, args []string) error {
			categories, err := uc.Execute(cmd.Context())
			if err != nil {
				return err
			}
			if len(categories) == 0 {
				fmt.Println("カテゴリが付与された記事がありません。")
				return nil
			}
			for _, c := range categories {
				fmt.Println(c)
			}
			return nil
		},
	}
}
