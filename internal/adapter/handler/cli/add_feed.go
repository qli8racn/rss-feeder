package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// NewAddFeedCommand は resolveFeedURLUC で入力URLをフィードURLに解決した上で addFeedUC に渡す。
func NewAddFeedCommand(addFeedUC *usecase.AddFeedUsecase, resolveFeedURLUC *usecase.ResolveFeedURLUsecase) *cobra.Command {
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
			return nil
		},
	}
}
