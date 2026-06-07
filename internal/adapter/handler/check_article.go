package handler

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func NewCheckArticleCommand(uc *usecase.CheckArticleUsecase) *cobra.Command {
	return &cobra.Command{
		Use:    "check-article <id>",
		Short:  "記事 ID の存在を検証する（Hook から使用）",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("ID は整数で指定してください: %s", args[0])
			}
			if err := uc.Execute(cmd.Context(), id); err != nil {
				if errors.Is(err, usecase.ErrArticleNotFound) {
					return fmt.Errorf("記事が見つかりません: ID %d", id)
				}
				return err
			}
			fmt.Printf("OK: 記事 ID %d は存在します\n", id)
			return nil
		},
	}
}
