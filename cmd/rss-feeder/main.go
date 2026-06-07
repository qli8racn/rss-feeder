package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/samber/do/v2"
	"github.com/spf13/cobra"

	adapterfile "github.com/qli8racn/rss-feeder/internal/adapter/driver/file"
	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	auditlogrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/auditlog"
	dbmaintrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/dbmaintenance"
	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
	adapterrss "github.com/qli8racn/rss-feeder/internal/adapter/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/adapter/handler"
	"github.com/qli8racn/rss-feeder/internal/migration"
	driverfile "github.com/qli8racn/rss-feeder/internal/driver/file"
	"github.com/qli8racn/rss-feeder/internal/driver/readerdb"
	dbrepoarticle "github.com/qli8racn/rss-feeder/internal/driver/readerdb/article"
	dbrepoauditlog "github.com/qli8racn/rss-feeder/internal/driver/readerdb/auditlog"
	dbrepodbmaint "github.com/qli8racn/rss-feeder/internal/driver/readerdb/dbmaintenance"
	dbrepofeed "github.com/qli8racn/rss-feeder/internal/driver/readerdb/feed"
	driverrss "github.com/qli8racn/rss-feeder/internal/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func main() {
	i := do.New()

	do.Provide(i, readerdb.NewClient)
	do.Provide(i, dbrepoarticle.NewRepository)
	do.Provide(i, dbrepofeed.NewRepository)
	do.Provide(i, driverrss.NewReader)
	do.Provide(i, driverfile.NewFeedsReader)
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

	root := &cobra.Command{
		Use:   "rss-feeder",
		Short: "RSS フィードを取得・管理する CLI ツール",
	}

	root.AddCommand(
		handler.NewFetchCommand(
			do.MustInvoke[adapterfile.FeedsReader](i),
			fetchUC,
		),
		handler.NewListCommand(listUC),
		handler.NewBookmarkCommand(bookmarkUC),
		handler.NewResetCommand(resetUC),
		handler.NewCheckArticleCommand(checkArticleUC),
		handler.NewCheckBookmarkedCommand(checkBookmarkedUC),
		handler.NewAuditCommand(auditUC),
		handler.NewMaintenanceCommand(maintenanceUC),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
