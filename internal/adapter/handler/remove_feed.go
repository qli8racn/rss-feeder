package handler

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func NewRemoveFeedCommand(uc *usecase.RemoveFeedUsecase) *cobra.Command {
	return &cobra.Command{
		Use:   "remove-feed <id>",
		Short: "RSS フィードを DB から削除する（記事も連動削除）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("id は整数で指定してください: %s", args[0])
			}
			if err := uc.Execute(cmd.Context(), id); err != nil {
				return err
			}
			fmt.Printf("削除しました: id=%d\n", id)
			return nil
		},
	}
}
