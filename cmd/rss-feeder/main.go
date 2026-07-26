package main

import (
	"context"
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
	userrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/user"
	adapterrss "github.com/qli8racn/rss-feeder/internal/adapter/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/adapter/handler/cli"
	"github.com/qli8racn/rss-feeder/internal/domain"
	driverfeeddiscovery "github.com/qli8racn/rss-feeder/internal/driver/feeddiscovery"
	driverfeedenrich "github.com/qli8racn/rss-feeder/internal/driver/feedenrich"
	driverhtmlfetch "github.com/qli8racn/rss-feeder/internal/driver/htmlfetch"
	"github.com/qli8racn/rss-feeder/internal/driver/readerdb"
	dbrepoarticle "github.com/qli8racn/rss-feeder/internal/driver/readerdb/article"
	dbrepoauditlog "github.com/qli8racn/rss-feeder/internal/driver/readerdb/auditlog"
	dbrepodbmaint "github.com/qli8racn/rss-feeder/internal/driver/readerdb/dbmaintenance"
	dbrepofeed "github.com/qli8racn/rss-feeder/internal/driver/readerdb/feed"
	dbrepouser "github.com/qli8racn/rss-feeder/internal/driver/readerdb/user"
	driverrss "github.com/qli8racn/rss-feeder/internal/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/migration"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func main() {
	rssAgentPath := flag.String("rss-agent-path", "bin/rss-agent", "フィードURL自動探索のAIフォールバックに使う rss-agent バイナリのパス")
	flag.Parse()

	i := do.New()

	do.Provide(i, readerdb.NewClient)
	do.Provide(i, dbrepoarticle.NewRepository)
	do.Provide(i, dbrepofeed.NewRepository)
	do.Provide(i, dbrepouser.NewRepository)
	do.Provide(i, driverrss.NewReader)
	do.Provide(i, dbrepoauditlog.NewRepository)
	do.Provide(i, dbrepodbmaint.NewMaintainer)
	do.Provide(i, driverhtmlfetch.NewFetcher)

	db := do.MustInvoke[*sql.DB](i)
	if err := migration.Run(db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// cmd/rss-feeder はユーザー概念に対応しない（要件のスコープ外）ため、常に
	// デフォルトユーザーとして暗黙的に動作する。
	resolveUserUC := usecase.NewResolveUserUsecase(do.MustInvoke[userrepo.Repository](i))
	user, err := resolveUserUC.Execute(context.Background(), domain.DefaultUserName)
	if err != nil {
		log.Fatalf("default user resolution failed: %v", err)
	}
	userID := user.ID

	fetchUC := usecase.NewFetchUsecase(
		do.MustInvoke[articlerepo.Repository](i),
		do.MustInvoke[feedrepo.Repository](i),
		do.MustInvoke[adapterrss.RSSReader](i),
		userID,
	)
	listUC := usecase.NewListUsecase(
		do.MustInvoke[articlerepo.Repository](i),
		userID,
	)
	bookmarkUC := usecase.NewBookmarkUsecase(
		do.MustInvoke[articlerepo.Repository](i),
		userID,
	)
	resetUC := usecase.NewResetUsecase(
		do.MustInvoke[articlerepo.Repository](i),
		userID,
	)
	checkArticleUC := usecase.NewCheckArticleUsecase(
		do.MustInvoke[articlerepo.Repository](i),
		userID,
	)
	checkBookmarkedUC := usecase.NewCheckBookmarkedUsecase(
		do.MustInvoke[articlerepo.Repository](i),
		userID,
	)
	auditUC := usecase.NewAuditUsecase(
		do.MustInvoke[auditlogrepo.Repository](i),
	)
	maintenanceUC := usecase.NewMaintenanceUsecase(
		do.MustInvoke[dbmaintrepo.Maintainer](i),
	)
	searchUC := usecase.NewSearchUsecase(
		do.MustInvoke[articlerepo.Repository](i),
		userID,
	)
	addFeedUC := usecase.NewAddFeedUsecase(
		do.MustInvoke[feedrepo.Repository](i),
		userID,
	)
	listFeedsUC := usecase.NewListFeedsUsecase(
		do.MustInvoke[feedrepo.Repository](i),
		userID,
	)
	removeFeedUC := usecase.NewRemoveFeedUsecase(
		do.MustInvoke[feedrepo.Repository](i),
		userID,
	)
	categoriesUC := usecase.NewListCategoriesUsecase(
		do.MustInvoke[articlerepo.Repository](i),
		userID,
	)
	backfillMetadataUC := usecase.NewBackfillMetadataUsecase(
		do.MustInvoke[articlerepo.Repository](i),
		do.MustInvoke[feedrepo.Repository](i),
		do.MustInvoke[adapterrss.RSSReader](i),
		userID,
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
