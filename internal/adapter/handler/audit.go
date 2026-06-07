package handler

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func NewAuditCommand(uc *usecase.AuditUsecase) *cobra.Command {
	var action string
	var articleID int64
	var oldState, newState string

	cmd := &cobra.Command{
		Use:    "audit",
		Short:  "操作ログを audit_log に記録する（Hook から使用）",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			log := domain.AuditLog{
				Action:   domain.Action(action),
				OldState: oldState,
				NewState: newState,
			}
			if cmd.Flags().Changed("article-id") {
				log.ArticleID = &articleID
			}
			if err := uc.Execute(cmd.Context(), log); err != nil {
				return err
			}
			fmt.Printf("audit_log に記録しました: action=%s\n", action)
			return nil
		},
	}

	cmd.Flags().StringVar(&action, "action", "", "操作種別（fetch/bookmark/reset）")
	cmd.Flags().Int64Var(&articleID, "article-id", 0, "記事 ID")
	cmd.Flags().StringVar(&oldState, "old", "", "変更前の状態")
	cmd.Flags().StringVar(&newState, "new", "", "変更後の状態")
	_ = cmd.MarkFlagRequired("action")

	return cmd
}
