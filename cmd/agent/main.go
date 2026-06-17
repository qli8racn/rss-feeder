package main

import (
	"fmt"
	"os"

	"github.com/samber/do/v2"
	"github.com/spf13/cobra"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
	"github.com/qli8racn/rss-feeder/internal/adapter/handler/agent"
	"github.com/qli8racn/rss-feeder/internal/config"
	driveranthropic "github.com/qli8racn/rss-feeder/internal/driver/anthropic"
	"github.com/qli8racn/rss-feeder/internal/driver/readerdb"
	dbrepoarticle "github.com/qli8racn/rss-feeder/internal/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/usecase"
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
	do.Provide(i, driveranthropic.NewSummarizeAgent)
	do.Provide(i, driveranthropic.NewPreferenceAgent)
	do.Provide(i, driveranthropic.NewEnrichAgent)

	summarizeUC := usecase.NewSummarizeUsecase(do.MustInvoke[adapteranthropic.SummarizeAgent](i))
	preferenceUC := usecase.NewPreferenceUsecase(do.MustInvoke[adapteranthropic.PreferenceAgent](i))
	enrichUC := usecase.NewEnrichUsecase(do.MustInvoke[adapteranthropic.EnrichAgent](i))

	root := &cobra.Command{
		Use:   "rss-agent",
		Short: "RSS 記事をAIで分析・要約するCLIツール",
	}

	root.AddCommand(
		agent.NewSummarizeCommand(summarizeUC),
		agent.NewPreferenceCommand(preferenceUC),
		agent.NewEnrichCommand(enrichUC),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
