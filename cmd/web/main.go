package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/samber/do/v2"

	"github.com/qli8racn/rss-feeder/internal/adapter/driver/htmlfetch"
	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	auditlogrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/auditlog"
	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
	adapterrss "github.com/qli8racn/rss-feeder/internal/adapter/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/adapter/handler/web"
	driverfeeddiscovery "github.com/qli8racn/rss-feeder/internal/driver/feeddiscovery"
	driverhtmlfetch "github.com/qli8racn/rss-feeder/internal/driver/htmlfetch"
	"github.com/qli8racn/rss-feeder/internal/driver/readerdb"
	dbrepoarticle "github.com/qli8racn/rss-feeder/internal/driver/readerdb/article"
	dbrepoauditlog "github.com/qli8racn/rss-feeder/internal/driver/readerdb/auditlog"
	dbrepofeed "github.com/qli8racn/rss-feeder/internal/driver/readerdb/feed"
	driverrss "github.com/qli8racn/rss-feeder/internal/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/migration"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func main() {
	port := flag.Int("port", 8080, "HTTP サーバーのポート番号")
	staticDir := flag.String("static-dir", "web/static", "フロントエンド静的ファイルのディレクトリ")
	rssAgentPath := flag.String("rss-agent-path", "bin/rss-agent", "フィードURL自動探索のAIフォールバックに使う rss-agent バイナリのパス")
	feedDiscoveryConcurrency := flag.Int("feed-discovery-concurrency", 2, "フィードURL自動探索のAIフォールバック（rss-agentサブプロセス）の同時実行数の上限")
	flag.Parse()

	i := do.New()
	do.Provide(i, readerdb.NewClient)
	do.Provide(i, dbrepoarticle.NewRepository)
	do.Provide(i, dbrepoauditlog.NewRepository)
	do.Provide(i, dbrepofeed.NewRepository)
	do.Provide(i, driverrss.NewReader)
	do.Provide(i, driverhtmlfetch.NewFetcher)

	db := do.MustInvoke[*sql.DB](i)
	if err := migration.Run(db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	listUC := usecase.NewListUsecase(do.MustInvoke[articlerepo.Repository](i))
	searchUC := usecase.NewSearchUsecase(do.MustInvoke[articlerepo.Repository](i))
	bookmarkUC := usecase.NewBookmarkUsecase(do.MustInvoke[articlerepo.Repository](i))
	auditUC := usecase.NewAuditUsecase(do.MustInvoke[auditlogrepo.Repository](i))
	categoriesUC := usecase.NewListCategoriesUsecase(do.MustInvoke[articlerepo.Repository](i))
	fetchUC := usecase.NewFetchUsecase(
		do.MustInvoke[articlerepo.Repository](i),
		do.MustInvoke[feedrepo.Repository](i),
		do.MustInvoke[adapterrss.RSSReader](i),
	)
	addFeedUC := usecase.NewAddFeedUsecase(do.MustInvoke[feedrepo.Repository](i))
	listFeedsUC := usecase.NewListFeedsUsecase(do.MustInvoke[feedrepo.Repository](i))
	removeFeedUC := usecase.NewRemoveFeedUsecase(do.MustInvoke[feedrepo.Repository](i))
	feedDiscoveryAgent := driverfeeddiscovery.NewSubprocessAgent(*rssAgentPath, *feedDiscoveryConcurrency)
	resolveFeedURLUC := usecase.NewResolveFeedURLUsecase(
		do.MustInvoke[adapterrss.RSSReader](i),
		do.MustInvoke[htmlfetch.Fetcher](i),
		feedDiscoveryAgent,
	)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "DELETE"},
	}))

	r.Get("/api/articles", web.ListArticlesHandler(listUC))
	r.Get("/api/articles/search", web.SearchArticlesHandler(searchUC))
	r.Post("/api/articles/fetch", web.FetchLatestHandler(fetchUC))
	r.Post("/api/articles/{id}/bookmark", web.BookmarkArticleHandler(bookmarkUC, auditUC))
	r.Get("/api/categories", web.ListCategoriesHandler(categoriesUC))
	r.Get("/api/feeds", web.ListFeedsHandler(listFeedsUC))
	r.Post("/api/feeds", web.AddFeedHandler(addFeedUC, resolveFeedURLUC))
	r.Delete("/api/feeds/{id}", web.RemoveFeedHandler(removeFeedUC))
	r.Handle("/*", http.FileServer(http.Dir(*staticDir)))

	addr := fmt.Sprintf(":%d", *port)
	fmt.Printf("Listening on http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
