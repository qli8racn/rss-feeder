package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/samber/do/v2"
	"github.com/spf13/cobra"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	auditlogrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/auditlog"
	dbmaintrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/dbmaintenance"
	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
	adapterrss "github.com/qli8racn/rss-feeder/internal/adapter/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/adapter/handler"
	"github.com/qli8racn/rss-feeder/internal/driver/readerdb"
	dbrepoarticle "github.com/qli8racn/rss-feeder/internal/driver/readerdb/article"
	dbrepoauditlog "github.com/qli8racn/rss-feeder/internal/driver/readerdb/auditlog"
	dbrepodbmaint "github.com/qli8racn/rss-feeder/internal/driver/readerdb/dbmaintenance"
	dbrepofeed "github.com/qli8racn/rss-feeder/internal/driver/readerdb/feed"
	driverrss "github.com/qli8racn/rss-feeder/internal/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/migration"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func main() {
	i := do.New()

	do.Provide(i, readerdb.NewClient)
	do.Provide(i, dbrepoarticle.NewRepository)
	do.Provide(i, dbrepofeed.NewRepository)
	do.Provide(i, driverrss.NewReader)
	do.Provide(i, dbrepoauditlog.NewRepository)
	do.Provide(i, dbrepodbmaint.NewMaintainer)

	db := do.MustInvoke[*sql.DB](i)
	if err := migration.Run(db); err != nil {
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

	root := &cobra.Command{
		Use:   "rss-feeder",
		Short: "RSS フィードを取得・管理する CLI ツール",
	}

	root.AddCommand(
		handler.NewFetchCommand(listFeedsUC, fetchUC),
		handler.NewListCommand(listUC),
		handler.NewBookmarkCommand(bookmarkUC),
		handler.NewResetCommand(resetUC),
		handler.NewCheckArticleCommand(checkArticleUC),
		handler.NewCheckBookmarkedCommand(checkBookmarkedUC),
		handler.NewAuditCommand(auditUC),
		handler.NewMaintenanceCommand(maintenanceUC),
		handler.NewSearchCommand(searchUC),
		handler.NewAddFeedCommand(addFeedUC),
		handler.NewListFeedsCommand(listFeedsUC),
		handler.NewRemoveFeedCommand(removeFeedUC),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
