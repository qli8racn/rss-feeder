package agent

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func NewEnrichCommand(uc *usecase.EnrichUsecase) *cobra.Command {
	var limit int
	var force bool

	cmd := &cobra.Command{
		Use:   "enrich",
		Short: "記事に要約・カテゴリを付与してDBに保存する",
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := uc.Execute(cmd.Context(), anthropic.EnrichOptions{Limit: limit, Force: force})
			if err != nil {
				return err
			}
			fmt.Printf("%d 件の記事を要約・分類しました\n", n)
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "処理件数")
	cmd.Flags().BoolVar(&force, "force", false, "要約済みの記事も含め、最新記事を対象に再処理する")

	return cmd
}
