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

			result, err := uc.Execute(cmd.Context(), urls)

			for i, fr := range result.Feeds {
				fmt.Printf("[%d/%d] %s\n", i+1, len(urls), urls[i])
				if fr.Err != nil {
					fmt.Printf("  エラー: %v\n", fr.Err)
					continue
				}
				for _, a := range fr.Saved {
					fmt.Printf("  + %s  %s\n", a.Title, a.URL)
				}
				if fr.Skipped > 0 {
					fmt.Printf("  スキップ: %d 件\n", fr.Skipped)
				}
			}

			fmt.Printf("\n完了: 新規 %d 件 / スキップ %d 件 / エラー %d 件\n",
				result.TotalSaved(), result.TotalSkipped(), result.TotalErrors())
			return err
		},
	}
}
