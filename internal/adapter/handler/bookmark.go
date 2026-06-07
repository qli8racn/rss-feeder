package handler

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func NewBookmarkCommand(uc *usecase.BookmarkUsecase) *cobra.Command {
	return &cobra.Command{
		Use:   "bookmark <id>",
		Short: "記事のお気に入りをトグルする",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("ID は整数で指定してください: %s", args[0])
			}

			article, err := uc.Execute(cmd.Context(), id)
			if err != nil {
				if errors.Is(err, usecase.ErrArticleNotFound) {
					return fmt.Errorf("記事が見つかりません: ID %d", id)
				}
				return err
			}

			if article.Bookmarked {
				fmt.Printf("お気に入りに登録しました: [%d] %s\n", article.ID, article.Title)
			} else {
				fmt.Printf("お気に入りを解除しました: [%d] %s\n", article.ID, article.Title)
			}
			return nil
		},
	}
}
