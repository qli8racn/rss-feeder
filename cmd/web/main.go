package main

import (
	"context"
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
	userrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/user"
	adapterrss "github.com/qli8racn/rss-feeder/internal/adapter/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/adapter/handler/web"
	"github.com/qli8racn/rss-feeder/internal/config"
	"github.com/qli8racn/rss-feeder/internal/domain"
	driverfeeddiscovery "github.com/qli8racn/rss-feeder/internal/driver/feeddiscovery"
	driverhtmlfetch "github.com/qli8racn/rss-feeder/internal/driver/htmlfetch"
	"github.com/qli8racn/rss-feeder/internal/driver/readerdb"
	dbrepoarticle "github.com/qli8racn/rss-feeder/internal/driver/readerdb/article"
	dbrepoauditlog "github.com/qli8racn/rss-feeder/internal/driver/readerdb/auditlog"
	dbrepofeed "github.com/qli8racn/rss-feeder/internal/driver/readerdb/feed"
	dbrepouser "github.com/qli8racn/rss-feeder/internal/driver/readerdb/user"
	"github.com/qli8racn/rss-feeder/internal/driver/readerpg"
	pgrepoarticle "github.com/qli8racn/rss-feeder/internal/driver/readerpg/article"
	pgrepoauditlog "github.com/qli8racn/rss-feeder/internal/driver/readerpg/auditlog"
	pgrepofeed "github.com/qli8racn/rss-feeder/internal/driver/readerpg/feed"
	pgrepouser "github.com/qli8racn/rss-feeder/internal/driver/readerpg/user"
	driverrss "github.com/qli8racn/rss-feeder/internal/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/migration"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func main() {
	port := flag.Int("port", 8080, "HTTP サーバーのポート番号")
	staticDir := flag.String("static-dir", "web/static", "フロントエンド静的ファイルのディレクトリ")
	openapiSpec := flag.String("openapi-spec", "docs/openapi.yaml", "APIリクエストのスキーマ検証に使う OpenAPI 仕様ファイルのパス")
	rssAgentPath := flag.String("rss-agent-path", "bin/rss-agent", "フィードURL自動探索のAIフォールバックに使う rss-agent バイナリのパス")
	feedDiscoveryConcurrency := flag.Int("feed-discovery-concurrency", 2, "フィードURL自動探索のAIフォールバック（rss-agentサブプロセス）の同時実行数の上限")
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
		do.Provide(i, pgrepoauditlog.NewRepository)
		do.Provide(i, pgrepofeed.NewRepository)
		do.Provide(i, pgrepouser.NewRepository)
	} else {
		do.Provide(i, readerdb.NewClient)
		do.Provide(i, dbrepoarticle.NewRepository)
		do.Provide(i, dbrepoauditlog.NewRepository)
		do.Provide(i, dbrepofeed.NewRepository)
		do.Provide(i, dbrepouser.NewRepository)
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

	// cmd/web はユーザー概念に対応しない（要件のスコープ外）ため、常に
	// デフォルトユーザーとして暗黙的に動作する。
	resolveUserUC := usecase.NewResolveUserUsecase(do.MustInvoke[userrepo.Repository](i))
	user, err := resolveUserUC.Execute(context.Background(), domain.DefaultUserName)
	if err != nil {
		log.Fatalf("default user resolution failed: %v", err)
	}
	userID := user.ID

	listUC := usecase.NewListUsecase(do.MustInvoke[articlerepo.Repository](i), userID)
	searchUC := usecase.NewSearchUsecase(do.MustInvoke[articlerepo.Repository](i), userID)
	bookmarkUC := usecase.NewBookmarkUsecase(do.MustInvoke[articlerepo.Repository](i), userID)
	auditUC := usecase.NewAuditUsecase(do.MustInvoke[auditlogrepo.Repository](i))
	categoriesUC := usecase.NewListCategoriesUsecase(do.MustInvoke[articlerepo.Repository](i), userID)
	fetchUC := usecase.NewFetchUsecase(
		do.MustInvoke[articlerepo.Repository](i),
		do.MustInvoke[feedrepo.Repository](i),
		do.MustInvoke[adapterrss.RSSReader](i),
		userID,
	)
	addFeedUC := usecase.NewAddFeedUsecase(do.MustInvoke[feedrepo.Repository](i), userID)
	listFeedsUC := usecase.NewListFeedsUsecase(do.MustInvoke[feedrepo.Repository](i), userID)
	removeFeedUC := usecase.NewRemoveFeedUsecase(do.MustInvoke[feedrepo.Repository](i), userID)
	feedDiscoveryAgent := driverfeeddiscovery.NewSubprocessAgent(*rssAgentPath, *feedDiscoveryConcurrency)
	resolveFeedURLUC := usecase.NewResolveFeedURLUsecase(
		do.MustInvoke[adapterrss.RSSReader](i),
		do.MustInvoke[htmlfetch.Fetcher](i),
		feedDiscoveryAgent,
	)

	requestValidator, err := web.NewRequestValidatorMiddleware(*openapiSpec)
	if err != nil {
		log.Fatalf("failed to build OpenAPI request validator: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "DELETE"},
	}))

	r.Route("/api", func(r chi.Router) {
		// OpenAPI 仕様外のパス（"/*" の静的ファイル配信）が 404 にならないよう、/api 配下にのみ適用する。
		r.Use(requestValidator)
		r.Get("/articles", web.ListArticlesHandler(listUC))
		r.Get("/articles/search", web.SearchArticlesHandler(searchUC))
		r.Post("/articles/fetch", web.FetchLatestHandler(fetchUC))
		r.Post("/articles/{id}/bookmark", web.BookmarkArticleHandler(bookmarkUC, auditUC))
		r.Get("/categories", web.ListCategoriesHandler(categoriesUC))
		r.Get("/feeds", web.ListFeedsHandler(listFeedsUC))
		r.Post("/feeds", web.AddFeedHandler(addFeedUC, resolveFeedURLUC, fetchUC))
		r.Delete("/feeds/{id}", web.RemoveFeedHandler(removeFeedUC))
	})
	r.Handle("/*", http.FileServer(http.Dir(*staticDir)))

	addr := fmt.Sprintf(":%d", *port)
	fmt.Printf("Listening on http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
