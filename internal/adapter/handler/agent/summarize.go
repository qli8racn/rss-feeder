package agent

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func NewSummarizeCommand(uc *usecase.SummarizeUsecase) *cobra.Command {
	var feedURL string
	var limit int

	cmd := &cobra.Command{
		Use:   "summarize",
		Short: "最新記事を要約する",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := uc.Execute(cmd.Context(), anthropic.SummarizeOptions{FeedURL: feedURL, Limit: limit})
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}
	cmd.Flags().StringVar(&feedURL, "feed", "", "絞り込むフィードのURL")
	cmd.Flags().IntVar(&limit, "limit", 10, "取得件数")

	return cmd
}
