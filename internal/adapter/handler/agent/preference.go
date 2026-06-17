package agent

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func NewPreferenceCommand(uc *usecase.PreferenceUsecase) *cobra.Command {
	return &cobra.Command{
		Use:   "preference",
		Short: "ブックマークから趣向を分析する",
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
