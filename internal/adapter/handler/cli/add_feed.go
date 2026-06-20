package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// NewAddFeedCommand は resolveFeedURLUC で入力URLをフィードURLに解決した上で addFeedUC に渡す。
// 登録成功後、そのフィードの記事取得（fetchUC）→新規保存記事の要約・カテゴライズ（triggerEnrichUC）を
// 続けて行う。どちらも best-effort であり、失敗してもフィード登録自体の成功は変えない
// （警告として標準エラー出力に出すのみ）。
func NewAddFeedCommand(
	addFeedUC *usecase.AddFeedUsecase,
	resolveFeedURLUC *usecase.ResolveFeedURLUsecase,
	fetchUC *usecase.FetchUsecase,
	triggerEnrichUC *usecase.TriggerEnrichUsecase,
) *cobra.Command {
	return &cobra.Command{
		Use:   "add-feed <url>",
		Short: "RSS フィード URL を DB に登録する",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedURL, err := resolveFeedURLUC.Execute(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if _, err := addFeedUC.Execute(cmd.Context(), resolvedURL); err != nil {
				return err
			}
			fmt.Printf("登録しました: %s\n", resolvedURL)

			result, err := fetchUC.Execute(cmd.Context(), []string{resolvedURL})
			if err != nil {
				fmt.Fprintf(os.Stderr, "警告: 記事の取得に失敗しました: %v\n", err)
				return nil
			}
			if saved := result.TotalSaved(); saved > 0 {
				if err := triggerEnrichUC.Execute(cmd.Context(), resolvedURL, saved); err != nil {
					fmt.Fprintf(os.Stderr, "警告: 要約・カテゴライズに失敗しました: %v\n", err)
				}
			}
			return nil
		},
	}
}
