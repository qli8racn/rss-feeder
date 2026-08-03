package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/samber/do/v2"
	"github.com/spf13/cobra"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
	"github.com/qli8racn/rss-feeder/internal/adapter/driver/htmlfetch"
	userrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/user"
	adapterrss "github.com/qli8racn/rss-feeder/internal/adapter/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/adapter/handler/agent"
	"github.com/qli8racn/rss-feeder/internal/config"
	"github.com/qli8racn/rss-feeder/internal/domain"
	driveranthropic "github.com/qli8racn/rss-feeder/internal/driver/anthropic"
	driverhtmlfetch "github.com/qli8racn/rss-feeder/internal/driver/htmlfetch"
	driverlogger "github.com/qli8racn/rss-feeder/internal/driver/logger"
	"github.com/qli8racn/rss-feeder/internal/driver/readerdb"
	dbrepoarticle "github.com/qli8racn/rss-feeder/internal/driver/readerdb/article"
	dbrepofeed "github.com/qli8racn/rss-feeder/internal/driver/readerdb/feed"
	dbrepouser "github.com/qli8racn/rss-feeder/internal/driver/readerdb/user"
	"github.com/qli8racn/rss-feeder/internal/driver/readerpg"
	pgrepoarticle "github.com/qli8racn/rss-feeder/internal/driver/readerpg/article"
	pgrepofeed "github.com/qli8racn/rss-feeder/internal/driver/readerpg/feed"
	pgrepouser "github.com/qli8racn/rss-feeder/internal/driver/readerpg/user"
	driverrss "github.com/qli8racn/rss-feeder/internal/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/migration"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func main() {
	if err := config.SetupAnthropicAPIKey(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	i := do.New()
	do.Provide(i, config.NewProvider)
	do.Provide(i, driverlogger.NewLogger)
	cfg, err := do.Invoke[*config.Config](i)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if cfg.DB.IsSupabase() {
		do.Provide(i, readerpg.NewClient)
		do.Provide(i, pgrepoarticle.NewRepository)
		do.Provide(i, pgrepofeed.NewRepository)
		do.Provide(i, pgrepouser.NewRepository)
	} else {
		do.Provide(i, readerdb.NewClient)
		do.Provide(i, dbrepoarticle.NewRepository)
		do.Provide(i, dbrepofeed.NewRepository)
		do.Provide(i, dbrepouser.NewRepository)
	}
	do.Provide(i, driverrss.NewReader)
	do.Provide(i, driverhtmlfetch.NewFetcher)

	db, err := do.Invoke[*sql.DB](i)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// cmd/agent（rss-agent）はcmd/web・cmd/rss-feederからサブプロセスとして頻繁に起動されるが、
	// usersテーブルを含むスキーマの存在は前提にできない（cmd/agent単独起動や、親プロセスの
	// マイグレーションより前に子プロセスが起動する競合を考慮）ため無条件に実行する。SQLite側は
	// PRAGMA user_versionによるバージョン管理（internal/migration/migration.go）により、既に
	// 最新スキーマまで移行済みのDBに対しては読み取り専用の2クエリで即returnするため、頻繁な
	// 起動でも書き込みロックを取り合う心配はない。
	if err := migration.RunFor(cfg, db); err != nil {
		fmt.Fprintf(os.Stderr, "migration failed: %v\n", err)
		os.Exit(1)
	}

	// cmd/agent（bin/rss-agent）はユーザー概念に対応しない（要件のスコープ外）ため、常に
	// デフォルトユーザーとして暗黙的に動作する。preference・curate・discoverAgentは
	// *domain.User をDIコンテナ経由で取得する（internal/driver/anthropic/preference.go 参照）。
	resolveUserUC := usecase.NewResolveUserUsecase(do.MustInvoke[userrepo.Repository](i))
	user, err := resolveUserUC.Execute(context.Background(), domain.DefaultUserName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "default user resolution failed: %v\n", err)
		os.Exit(1)
	}
	do.ProvideValue(i, user)

	do.Provide(i, driveranthropic.NewSummarizeAgent)
	do.Provide(i, driveranthropic.NewPreferenceAgent)
	do.Provide(i, driveranthropic.NewEnrichAgent)
	do.Provide(i, driveranthropic.NewFeedDiscoveryAgent)
	do.Provide(i, driveranthropic.NewCurateAgent)
	do.Provide(i, driveranthropic.NewDiscoverAgent)

	summarizeUC := usecase.NewSummarizeUsecase(do.MustInvoke[adapteranthropic.SummarizeAgent](i))
	preferenceUC := usecase.NewPreferenceUsecase(do.MustInvoke[adapteranthropic.PreferenceAgent](i))
	enrichUC := usecase.NewEnrichUsecase(do.MustInvoke[adapteranthropic.EnrichAgent](i))
	discoverFeedUC := usecase.NewDiscoverFeedUsecase(
		do.MustInvoke[htmlfetch.Fetcher](i),
		do.MustInvoke[adapteranthropic.FeedDiscoveryAgent](i),
		do.MustInvoke[adapterrss.RSSReader](i),
	)
	curateUC := usecase.NewCurateUsecase(do.MustInvoke[adapteranthropic.CurateAgent](i))
	discoverUC := usecase.NewDiscoverUsecase(do.MustInvoke[adapteranthropic.DiscoverAgent](i))

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
		agent.NewCurateCommand(curateUC),
		agent.NewDiscoverCommand(discoverUC),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
