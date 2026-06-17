package main

import (
	"context"
	"fmt"
	"os"

	"github.com/samber/do/v2"
	"github.com/spf13/cobra"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/config"
	driveranthropic "github.com/qli8racn/rss-feeder/internal/driver/anthropic"
	"github.com/qli8racn/rss-feeder/internal/driver/readerdb"
	dbrepoarticle "github.com/qli8racn/rss-feeder/internal/driver/readerdb/article"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if cfg.AnthropicAPIKey != "" {
		if err := os.Setenv("ANTHROPIC_API_KEY", cfg.AnthropicAPIKey); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	i := do.New()
	do.Provide(i, readerdb.NewClient)
	do.Provide(i, dbrepoarticle.NewRepository)

	root := &cobra.Command{
		Use:   "rss-agent",
		Short: "RSS 記事をAIで分析・要約するCLIツール",
	}

	var feedURL string
	var limit int
	summarizeCmd := &cobra.Command{
		Use:   "summarize",
		Short: "最新記事を要約する",
		RunE: func(cmd *cobra.Command, args []string) error {
			r := do.MustInvoke[articlerepo.Repository](i)
			result, err := driveranthropic.NewSummarizeAgent(r).Run(
				context.Background(),
				driveranthropic.SummarizeOptions{FeedURL: feedURL, Limit: limit},
			)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}
	summarizeCmd.Flags().StringVar(&feedURL, "feed", "", "絞り込むフィードのURL")
	summarizeCmd.Flags().IntVar(&limit, "limit", 10, "取得件数")

	preferenceCmd := &cobra.Command{
		Use:   "preference",
		Short: "ブックマークから趣向を分析する",
		RunE: func(cmd *cobra.Command, args []string) error {
			r := do.MustInvoke[articlerepo.Repository](i)
			result, err := driveranthropic.NewPreferenceAgent(r).Run(context.Background())
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}

	var enrichLimit int
	var enrichForce bool
	enrichCmd := &cobra.Command{
		Use:   "enrich",
		Short: "記事に要約・カテゴリを付与してDBに保存する",
		RunE: func(cmd *cobra.Command, args []string) error {
			r := do.MustInvoke[articlerepo.Repository](i)
			n, err := driveranthropic.NewEnrichAgent(r).Run(
				context.Background(),
				driveranthropic.EnrichOptions{Limit: enrichLimit, Force: enrichForce},
			)
			if err != nil {
				return err
			}
			fmt.Printf("%d 件の記事を要約・分類しました\n", n)
			return nil
		},
	}
	enrichCmd.Flags().IntVar(&enrichLimit, "limit", 10, "処理件数")
	enrichCmd.Flags().BoolVar(&enrichForce, "force", false, "要約済みの記事も含め、最新記事を対象に再処理する")

	root.AddCommand(summarizeCmd, preferenceCmd, enrichCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
