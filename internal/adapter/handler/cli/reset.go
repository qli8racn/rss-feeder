package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func NewResetCommand(uc *usecase.ResetUsecase) *cobra.Command {
	var flagYes bool

	cmd := &cobra.Command{
		Use:   "reset [-y]",
		Short: "お気に入り以外の記事を削除する",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			count, err := uc.Count(ctx)
			if err != nil {
				return err
			}
			if count == 0 {
				fmt.Println("削除対象の記事はありません。")
				return nil
			}

			if !flagYes {
				fmt.Printf("お気に入り以外の記事 %d 件を削除します。よろしいですか？ [y/N]: ", count)
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				answer := strings.TrimSpace(scanner.Text())
				if strings.ToLower(answer) != "y" {
					fmt.Println("キャンセルしました。")
					return nil
				}
			}

			result, err := uc.Execute(ctx)
			if err != nil {
				return err
			}

			fmt.Printf("削除完了: %d 件（お気に入り: %d 件 を保持）\n", result.Deleted, result.Bookmarked)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "確認をスキップして即時削除する")
	return cmd
}
