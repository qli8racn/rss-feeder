package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/samber/do/v2"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	auditlogrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/auditlog"
	"github.com/qli8racn/rss-feeder/internal/adapter/handler"
	"github.com/qli8racn/rss-feeder/internal/driver/readerdb"
	dbrepoarticle "github.com/qli8racn/rss-feeder/internal/driver/readerdb/article"
	dbrepoauditlog "github.com/qli8racn/rss-feeder/internal/driver/readerdb/auditlog"
	"github.com/qli8racn/rss-feeder/internal/migration"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func main() {
	port := flag.Int("port", 8080, "HTTP サーバーのポート番号")
	staticDir := flag.String("static-dir", "web/static", "フロントエンド静的ファイルのディレクトリ")
	flag.Parse()

	i := do.New()
	do.Provide(i, readerdb.NewClient)
	do.Provide(i, dbrepoarticle.NewRepository)
	do.Provide(i, dbrepoauditlog.NewRepository)

	db := do.MustInvoke[*sql.DB](i)
	if err := migration.Run(db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	listUC := usecase.NewListUsecase(do.MustInvoke[articlerepo.Repository](i))
	searchUC := usecase.NewSearchUsecase(do.MustInvoke[articlerepo.Repository](i))
	bookmarkUC := usecase.NewBookmarkUsecase(do.MustInvoke[articlerepo.Repository](i))
	auditUC := usecase.NewAuditUsecase(do.MustInvoke[auditlogrepo.Repository](i))
	categoriesUC := usecase.NewListCategoriesUsecase(do.MustInvoke[articlerepo.Repository](i))

	mux := handler.NewMux(listUC, searchUC, bookmarkUC, auditUC, categoriesUC, *staticDir)

	addr := fmt.Sprintf(":%d", *port)
	fmt.Printf("Listening on http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
