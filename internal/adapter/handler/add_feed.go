package handler

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func NewAddFeedCommand(uc *usecase.AddFeedUsecase) *cobra.Command {
	return &cobra.Command{
		Use:   "add-feed <url>",
		Short: "RSS フィード URL を DB に登録する",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			if err := uc.Execute(cmd.Context(), url); err != nil {
				return err
			}
			fmt.Printf("登録しました: %s\n", url)
			return nil
		},
	}
}
