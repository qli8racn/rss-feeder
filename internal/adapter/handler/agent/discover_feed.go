package agent

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func NewDiscoverFeedCommand(uc *usecase.DiscoverFeedUsecase) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discover-feed <url>",
		Short: "URLからRSS/AtomフィードのURLを推測する",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			feedURL, err := uc.Execute(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			fmt.Println(feedURL)
			return nil
		},
	}
	return cmd
}
