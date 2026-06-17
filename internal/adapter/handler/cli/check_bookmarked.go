package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func NewCheckBookmarkedCommand(uc *usecase.CheckBookmarkedUsecase) *cobra.Command {
	return &cobra.Command{
		Use:    "check-bookmarked",
		Short:  "お気に入り記事数を確認する（Hook から使用）",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			count, err := uc.Execute(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Printf("お気に入り記事 %d 件はリセット対象外です\n", count)
			return nil
		},
	}
}
