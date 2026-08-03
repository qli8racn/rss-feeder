package main

import (
	"database/sql"
	"flag"
	"log"
	"os"

	"github.com/samber/do/v2"
	"github.com/spf13/cobra"

	"github.com/qli8racn/rss-feeder/internal/adapter/driver/htmlfetch"
	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	auditlogrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/auditlog"
	dbmaintrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/dbmaintenance"
	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
	adapterrss "github.com/qli8racn/rss-feeder/internal/adapter/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/adapter/handler/cli"
	"github.com/qli8racn/rss-feeder/internal/config"
	driverfeeddiscovery "github.com/qli8racn/rss-feeder/internal/driver/feeddiscovery"
	driverfeedenrich "github.com/qli8racn/rss-feeder/internal/driver/feedenrich"
	driverhtmlfetch "github.com/qli8racn/rss-feeder/internal/driver/htmlfetch"
	"github.com/qli8racn/rss-feeder/internal/driver/readerdb"
	dbrepoarticle "github.com/qli8racn/rss-feeder/internal/driver/readerdb/article"
	dbrepoauditlog "github.com/qli8racn/rss-feeder/internal/driver/readerdb/auditlog"
	dbrepodbmaint "github.com/qli8racn/rss-feeder/internal/driver/readerdb/dbmaintenance"
	dbrepofeed "github.com/qli8racn/rss-feeder/internal/driver/readerdb/feed"
	"github.com/qli8racn/rss-feeder/internal/driver/readerpg"
	pgrepoarticle "github.com/qli8racn/rss-feeder/internal/driver/readerpg/article"
	pgrepoauditlog "github.com/qli8racn/rss-feeder/internal/driver/readerpg/auditlog"
	pgrepodbmaint "github.com/qli8racn/rss-feeder/internal/driver/readerpg/dbmaintenance"
	pgrepofeed "github.com/qli8racn/rss-feeder/internal/driver/readerpg/feed"
	driverrss "github.com/qli8racn/rss-feeder/internal/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/migration"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func main() {
	rssAgentPath := flag.String("rss-agent-path", "bin/rss-agent", "フィードURL自動探索のAIフォールバックに使う rss-agent バイナリのパス")
	flag.Parse()

	i := do.New()

	do.Provide(i, config.NewProvider)
	cfg, err := do.Invoke[*config.Config](i)
	if err != nil {
		log.Fatalf("%v", err)
	}

	if cfg.DB.IsSupabase() {
		do.Provide(i, readerpg.NewClient)
		do.Provide(i, pgrepoarticle.NewRepository)
		do.Provide(i, pgrepofeed.NewRepository)
		do.Provide(i, pgrepoauditlog.NewRepository)
		do.Provide(i, pgrepodbmaint.NewMaintainer)
	} else {
		do.Provide(i, readerdb.NewClient)
		do.Provide(i, dbrepoarticle.NewRepository)
		do.Provide(i, dbrepofeed.NewRepository)
		do.Provide(i, dbrepoauditlog.NewRepository)
		do.Provide(i, dbrepodbmaint.NewMaintainer)
	}
	do.Provide(i, driverrss.NewReader)
	do.Provide(i, driverhtmlfetch.NewFetcher)

	db, err := do.Invoke[*sql.DB](i)
	if err != nil {
		log.Fatalf("%v", err)
	}
	if err := migration.RunFor(cfg, db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	fetchUC := usecase.NewFetchUsecase(
		do.MustInvoke[articlerepo.Repository](i),
		do.MustInvoke[feedrepo.Repository](i),
		do.MustInvoke[adapterrss.RSSReader](i),
	)
	listUC := usecase.NewListUsecase(
		do.MustInvoke[articlerepo.Repository](i),
	)
	bookmarkUC := usecase.NewBookmarkUsecase(
		do.MustInvoke[articlerepo.Repository](i),
	)
	resetUC := usecase.NewResetUsecase(
		do.MustInvoke[articlerepo.Repository](i),
	)
	checkArticleUC := usecase.NewCheckArticleUsecase(
		do.MustInvoke[articlerepo.Repository](i),
	)
	checkBookmarkedUC := usecase.NewCheckBookmarkedUsecase(
		do.MustInvoke[articlerepo.Repository](i),
	)
	auditUC := usecase.NewAuditUsecase(
		do.MustInvoke[auditlogrepo.Repository](i),
	)
	maintenanceUC := usecase.NewMaintenanceUsecase(
		do.MustInvoke[dbmaintrepo.Maintainer](i),
	)
	searchUC := usecase.NewSearchUsecase(
		do.MustInvoke[articlerepo.Repository](i),
	)
	addFeedUC := usecase.NewAddFeedUsecase(
		do.MustInvoke[feedrepo.Repository](i),
	)
	listFeedsUC := usecase.NewListFeedsUsecase(
		do.MustInvoke[feedrepo.Repository](i),
	)
	removeFeedUC := usecase.NewRemoveFeedUsecase(
		do.MustInvoke[feedrepo.Repository](i),
	)
	categoriesUC := usecase.NewListCategoriesUsecase(
		do.MustInvoke[articlerepo.Repository](i),
	)
	backfillMetadataUC := usecase.NewBackfillMetadataUsecase(
		do.MustInvoke[articlerepo.Repository](i),
		do.MustInvoke[feedrepo.Repository](i),
		do.MustInvoke[adapterrss.RSSReader](i),
	)
	feedDiscoveryAgent := driverfeeddiscovery.NewSubprocessAgent(*rssAgentPath, 1)
	resolveFeedURLUC := usecase.NewResolveFeedURLUsecase(
		do.MustInvoke[adapterrss.RSSReader](i),
		do.MustInvoke[htmlfetch.Fetcher](i),
		feedDiscoveryAgent,
	)
	feedEnrichAgent := driverfeedenrich.NewSubprocessAgent(*rssAgentPath)
	triggerEnrichUC := usecase.NewTriggerEnrichUsecase(feedEnrichAgent)

	root := &cobra.Command{
		Use:   "rss-feeder",
		Short: "RSS フィードを取得・管理する CLI ツール",
	}

	root.AddCommand(
		cli.NewFetchCommand(fetchUC),
		cli.NewListCommand(listUC),
		cli.NewBookmarkCommand(bookmarkUC),
		cli.NewResetCommand(resetUC),
		cli.NewCheckArticleCommand(checkArticleUC),
		cli.NewCheckBookmarkedCommand(checkBookmarkedUC),
		cli.NewAuditCommand(auditUC),
		cli.NewMaintenanceCommand(maintenanceUC),
		cli.NewSearchCommand(searchUC),
		cli.NewAddFeedCommand(addFeedUC, resolveFeedURLUC, fetchUC, triggerEnrichUC),
		cli.NewListFeedsCommand(listFeedsUC),
		cli.NewRemoveFeedCommand(removeFeedUC),
		cli.NewCategoriesCommand(categoriesUC),
		cli.NewBackfillMetadataCommand(backfillMetadataUC),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
