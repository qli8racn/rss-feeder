package handler

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	adapterfile "github.com/qli8racn/rss-feeder/internal/adapter/driver/file"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func NewFetchCommand(feedsReader adapterfile.FeedsReader, uc *usecase.FetchUsecase) *cobra.Command {
	return &cobra.Command{
		Use:   "fetch",
		Short: "feeds.txt からフィードを取得して DB に保存する",
		RunE: func(cmd *cobra.Command, args []string) error {
			urls, err := feedsReader.Load()
			if err != nil {
				return fmt.Errorf("feeds.txt の読み込みに失敗: %w", err)
			}
			if len(urls) == 0 {
				fmt.Fprintln(os.Stderr, "警告: feeds.txt に有効な URL が見つかりませんでした")
				return nil
			}
			_, err = uc.Execute(cmd.Context(), urls)
			return err
		},
	}
}
