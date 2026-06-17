package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func NewMaintenanceCommand(uc *usecase.MaintenanceUsecase) *cobra.Command {
	return &cobra.Command{
		Use:    "maintenance",
		Short:  "DB の VACUUM と整合性チェックを実行する（Hook から使用）",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := uc.Execute(cmd.Context())
			if err != nil {
				return err
			}
			if result.IntegrityOK {
				fmt.Println("整合性チェック: OK")
			} else {
				fmt.Printf("整合性チェック: 問題あり\n%s\n", strings.Join(result.Details, "\n"))
			}
			fmt.Println("VACUUM 完了")
			return nil
		},
	}
}
