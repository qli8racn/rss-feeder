package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func NewBackfillMetadataCommand(uc *usecase.BackfillMetadataUsecase) *cobra.Command {
	return &cobra.Command{
		Use:   "backfill-metadata",
		Short: "登録済みフィードを再取得し、既存記事の出版元・サムネイルを補完する",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := uc.Execute(cmd.Context())
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
				if fr.Updated > 0 {
					fmt.Printf("  補完: %d 件\n", fr.Updated)
				}
			}

			fmt.Printf("\n完了: 補完 %d 件 / エラー %d 件\n", result.TotalUpdated(), result.TotalErrors())
			return err
		},
	}
}
