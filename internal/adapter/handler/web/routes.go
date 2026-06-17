package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// NewMux は Web ブラウザから記事を閲覧するための HTTP ハンドラ（JSON API + 静的ファイル配信）を構築する。
func NewMux(listUC *usecase.ListUsecase, searchUC *usecase.SearchUsecase, bookmarkUC *usecase.BookmarkUsecase, auditUC *usecase.AuditUsecase, categoriesUC *usecase.ListCategoriesUsecase, fetchUC *usecase.FetchUsecase, staticDir string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST"},
	}))

	r.Get("/api/articles", handleListArticles(listUC))
	r.Get("/api/articles/search", handleSearchArticles(searchUC))
	r.Post("/api/articles/fetch", handleFetchLatest(fetchUC))
	r.Post("/api/articles/{id}/bookmark", handleBookmarkArticle(bookmarkUC, auditUC))
	r.Get("/api/categories", handleListCategories(categoriesUC))
	r.Handle("/*", http.FileServer(http.Dir(staticDir)))

	return r
}
