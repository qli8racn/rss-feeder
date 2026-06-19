package main

import (
	"fmt"
	"os"

	"github.com/samber/do/v2"
	"github.com/spf13/cobra"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
	"github.com/qli8racn/rss-feeder/internal/adapter/driver/htmlfetch"
	adapterrss "github.com/qli8racn/rss-feeder/internal/adapter/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/adapter/handler/agent"
	"github.com/qli8racn/rss-feeder/internal/config"
	driveranthropic "github.com/qli8racn/rss-feeder/internal/driver/anthropic"
	driverhtmlfetch "github.com/qli8racn/rss-feeder/internal/driver/htmlfetch"
	"github.com/qli8racn/rss-feeder/internal/driver/readerdb"
	dbrepoarticle "github.com/qli8racn/rss-feeder/internal/driver/readerdb/article"
	driverrss "github.com/qli8racn/rss-feeder/internal/driver/rss"
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
	do.Provide(i, driverrss.NewReader)
	do.Provide(i, driverhtmlfetch.NewFetcher)
	do.Provide(i, driveranthropic.NewSummarizeAgent)
	do.Provide(i, driveranthropic.NewPreferenceAgent)
	do.Provide(i, driveranthropic.NewEnrichAgent)
	do.Provide(i, driveranthropic.NewFeedDiscoveryAgent)

	summarizeUC := usecase.NewSummarizeUsecase(do.MustInvoke[adapteranthropic.SummarizeAgent](i))
	preferenceUC := usecase.NewPreferenceUsecase(do.MustInvoke[adapteranthropic.PreferenceAgent](i))
	enrichUC := usecase.NewEnrichUsecase(do.MustInvoke[adapteranthropic.EnrichAgent](i))
	discoverFeedUC := usecase.NewDiscoverFeedUsecase(
		do.MustInvoke[htmlfetch.Fetcher](i),
		do.MustInvoke[adapteranthropic.FeedDiscoveryAgent](i),
		do.MustInvoke[adapterrss.RSSReader](i),
	)

	root := &cobra.Command{
		Use:   "rss-agent",
		Short: "RSS 記事をAIで分析・要約するCLIツール",
	}
	// 実行時エラーとcobraの自動usage出力が混在して紛らわしくならないようにする
	// （全サブコマンドに一括適用するため、ルートコマンドで設定する）。
	root.SilenceUsage = true

	root.AddCommand(
		agent.NewSummarizeCommand(summarizeUC),
		agent.NewPreferenceCommand(preferenceUC),
		agent.NewEnrichCommand(enrichUC),
		agent.NewDiscoverFeedCommand(discoverFeedUC),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
