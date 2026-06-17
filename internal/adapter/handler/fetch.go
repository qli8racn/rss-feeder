package handler

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func NewFetchCommand(fetchUC *usecase.FetchUsecase) *cobra.Command {
	return &cobra.Command{
		Use:   "fetch",
		Short: "登録済みフィードを取得して DB に保存する",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := fetchUC.ExecuteAll(cmd.Context())
			if err != nil && len(result.Feeds) == 0 {
				return err
			}
			if len(result.Feeds) == 0 {
				fmt.Fprintln(os.Stderr, msgNoFeeds)
				return nil
			}

			for i, fr := range result.Feeds {
				fmt.Printf("[%d/%d] %s\n", i+1, len(result.Feeds), fr.FeedURL)
				if fr.Err != nil {
					fmt.Printf("  エラー: %v\n", fr.Err)
					continue
				}
				for _, a := range fr.Saved {
					fmt.Printf("  + %s  %s\n", a.Title, a.URL)
				}
				if fr.Skipped > 0 {
					fmt.Printf("  スキップ: %d 件\n", fr.Skipped)
				}
			}

			fmt.Printf("\n完了: 新規 %d 件 / スキップ %d 件 / エラー %d 件\n",
				result.TotalSaved(), result.TotalSkipped(), result.TotalErrors())
			return err
		},
	}
}
