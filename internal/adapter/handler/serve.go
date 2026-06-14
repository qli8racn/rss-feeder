package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// articleDTO は記事を JSON で表現するための DTO。
type articleDTO struct {
	ID          int64     `json:"id"`
	FeedID      int64     `json:"feed_id"`
	FeedURL     string    `json:"feed_url"`
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	PublishedAt time.Time `json:"published_at"`
	Read        bool      `json:"read"`
	Bookmarked  bool      `json:"bookmarked"`
	FetchedAt   time.Time `json:"fetched_at"`
}

func toArticleDTO(a domain.Article) articleDTO {
	return articleDTO{
		ID:          a.ID,
		FeedID:      a.FeedID,
		FeedURL:     a.FeedURL,
		URL:         a.URL,
		Title:       a.Title,
		Content:     a.Content,
		PublishedAt: a.PublishedAt,
		Read:        a.Read,
		Bookmarked:  a.Bookmarked,
		FetchedAt:   a.FetchedAt,
	}
}

func toArticleDTOs(articles []domain.Article) []articleDTO {
	dtos := make([]articleDTO, len(articles))
	for i, a := range articles {
		dtos[i] = toArticleDTO(a)
	}
	return dtos
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// NewMux は Web ブラウザから記事を閲覧するための HTTP ハンドラ（JSON API + 静的ファイル配信）を構築する。
func NewMux(listUC *usecase.ListUsecase, searchUC *usecase.SearchUsecase, bookmarkUC *usecase.BookmarkUsecase, auditUC *usecase.AuditUsecase, staticDir string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST"},
	}))

	r.Get("/api/articles", handleListArticles(listUC))
	r.Get("/api/articles/search", handleSearchArticles(searchUC))
	r.Post("/api/articles/{id}/bookmark", handleBookmarkArticle(bookmarkUC, auditUC))
	r.Handle("/*", http.FileServer(http.Dir(staticDir)))

	return r
}

func handleListArticles(uc *usecase.ListUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mode := usecase.ListModeAll
		switch r.URL.Query().Get("mode") {
		case "unread":
			mode = usecase.ListModeUnread
		case "bookmarked":
			mode = usecase.ListModeBookmarked
		case "", "all":
			mode = usecase.ListModeAll
		default:
			writeJSONError(w, http.StatusBadRequest, "mode は all, unread, bookmarked のいずれかを指定してください")
			return
		}

		articles, err := uc.Execute(r.Context(), mode)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, toArticleDTOs(articles))
	}
}

func handleSearchArticles(uc *usecase.SearchUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		keyword := r.URL.Query().Get("q")
		if keyword == "" {
			writeJSONError(w, http.StatusBadRequest, "q は必須です")
			return
		}
		bookmarkedOnly := r.URL.Query().Get("bookmarked") == "true"

		articles, err := uc.Execute(r.Context(), keyword, bookmarkedOnly)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, toArticleDTOs(articles))
	}
}

func handleBookmarkArticle(bookmarkUC *usecase.BookmarkUsecase, auditUC *usecase.AuditUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "id は整数で指定してください")
			return
		}

		article, err := bookmarkUC.Execute(r.Context(), id)
		if err != nil {
			if errors.Is(err, usecase.ErrArticleNotFound) {
				writeJSONError(w, http.StatusNotFound, fmt.Sprintf("記事が見つかりません: ID %d", id))
				return
			}
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if err := auditUC.Execute(r.Context(), domain.AuditLog{
			Action:    domain.ActionBookmark,
			ArticleID: &article.ID,
		}); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, toArticleDTO(*article))
	}
}
