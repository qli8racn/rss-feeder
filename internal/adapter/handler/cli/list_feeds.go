package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

const msgNoFeeds = "登録済みのフィードがありません。add-feed コマンドで追加してください。"

func NewListFeedsCommand(uc *usecase.ListFeedsUsecase) *cobra.Command {
	return &cobra.Command{
		Use:   "list-feeds",
		Short: "登録済み RSS フィード一覧を表示する",
		RunE: func(cmd *cobra.Command, args []string) error {
			feeds, err := uc.Execute(cmd.Context())
			if err != nil {
				return err
			}
			if len(feeds) == 0 {
				fmt.Fprintln(os.Stderr, msgNoFeeds)
				return nil
			}
			for _, f := range feeds {
				title := f.Title
				if title == "" {
					title = "(未取得)"
				}
				lastFetched := "なし"
				if !f.LastFetched.IsZero() {
					lastFetched = f.LastFetched.Format("2006-01-02 15:04")
				}
				fmt.Printf("[%d] %s\n    %s  最終取得: %s\n", f.ID, f.FeedURL, title, lastFetched)
			}
			return nil
		},
	}
}
