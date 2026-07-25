package agent

import (
	"fmt"

	"github.com/spf13/cobra"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func NewCurateCommand(uc *usecase.CurateUsecase) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "curate",
		Short: "直近記事とブックマーク趣向から今日読む価値がある記事を推薦する",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := uc.Execute(cmd.Context(), adapteranthropic.CurateOptions{Limit: limit})
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 0, "考慮する直近記事数（デフォルト: 30）")
	return cmd
}
