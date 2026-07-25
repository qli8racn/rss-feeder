package agent

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func NewDiscoverCommand(uc *usecase.DiscoverUsecase) *cobra.Command {
	return &cobra.Command{
		Use:   "discover",
		Short: "ブックマーク趣向から未購読の RSS フィードを推薦する",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := uc.Execute(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}
}
