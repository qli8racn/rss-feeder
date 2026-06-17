package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// articleDTO は記事を JSON で表現するための DTO。
type articleDTO struct {
	ID           int64     `json:"id"`
	FeedID       int64     `json:"feed_id"`
	FeedURL      string    `json:"feed_url"`
	URL          string    `json:"url"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	PublishedAt  time.Time `json:"published_at"`
	Read         bool      `json:"read"`
	Bookmarked   bool      `json:"bookmarked"`
	FetchedAt    time.Time `json:"fetched_at"`
	Publisher    string    `json:"publisher"`
	ThumbnailURL string    `json:"thumbnail_url"`
	Summary      string    `json:"summary"`
	Category     string    `json:"category"`
}

func toArticleDTO(a domain.Article) articleDTO {
	return articleDTO{
		ID:           a.ID,
		FeedID:       a.FeedID,
		FeedURL:      a.FeedURL,
		URL:          a.URL,
		Title:        a.Title,
		Content:      a.Content,
		PublishedAt:  a.PublishedAt,
		Read:         a.Read,
		Bookmarked:   a.Bookmarked,
		FetchedAt:    a.FetchedAt,
		Publisher:    a.Publisher,
		ThumbnailURL: a.ThumbnailURL,
		Summary:      a.Summary,
		Category:     a.Category,
	}
}

func toArticleDTOs(articles []domain.Article) []articleDTO {
	dtos := make([]articleDTO, len(articles))
	for i, a := range articles {
		dtos[i] = toArticleDTO(a)
	}
	return dtos
}

// pagedArticlesDTO は記事一覧（ページネーション付き）を JSON で表現するための DTO。
type pagedArticlesDTO struct {
	Articles []articleDTO `json:"articles"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PerPage  int          `json:"per_page"`
}

// parseListQuery は記事一覧・検索 API に共通の category/sort/order/page/per_page クエリパラメータを解析する。
func parseListQuery(r *http.Request) (category, sort, order string, page, perPage int, err error) {
	category = r.URL.Query().Get("category")

	sort = r.URL.Query().Get("sort")
	if sort == "" {
		sort = "published_at"
	} else if !articlerepo.ValidSortFields[sort] {
		return "", "", "", 0, 0, fmt.Errorf("sort は title, publisher, category, published_at のいずれかを指定してください")
	}

	order = r.URL.Query().Get("order")
	switch order {
	case "":
		order = "desc"
	case "asc", "desc":
	default:
		return "", "", "", 0, 0, fmt.Errorf("order は asc, desc のいずれかを指定してください")
	}

	page = 1
	if v := r.URL.Query().Get("page"); v != "" {
		p, convErr := strconv.Atoi(v)
		if convErr != nil || p < 1 {
			return "", "", "", 0, 0, fmt.Errorf("page は1以上の整数を指定してください")
		}
		page = p
	}

	perPage = articlerepo.DefaultPerPage
	if v := r.URL.Query().Get("per_page"); v != "" {
		pp, convErr := strconv.Atoi(v)
		if convErr != nil || pp < 1 {
			return "", "", "", 0, 0, fmt.Errorf("per_page は1以上の整数を指定してください")
		}
		perPage = pp
	}

	return category, sort, order, page, perPage, nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// NewMux は Web ブラウザから記事を閲覧するための HTTP ハンドラ（JSON API + 静的ファイル配信）を構築する。
func NewMux(listUC *usecase.ListUsecase, searchUC *usecase.SearchUsecase, bookmarkUC *usecase.BookmarkUsecase, auditUC *usecase.AuditUsecase, categoriesUC *usecase.ListCategoriesUsecase, staticDir string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST"},
	}))

	r.Get("/api/articles", handleListArticles(listUC))
	r.Get("/api/articles/search", handleSearchArticles(searchUC))
	r.Get("/api/categories", handleListCategories(categoriesUC))
	r.Post("/api/articles/{id}/bookmark", handleBookmarkArticle(bookmarkUC, auditUC))
	r.Handle("/*", http.FileServer(http.Dir(staticDir)))

	return r
}

func handleListArticles(uc *usecase.ListUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var mode usecase.ListMode
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

		category, sort, order, page, perPage, err := parseListQuery(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		articles, total, err := uc.ExecuteFiltered(r.Context(), usecase.ListFilterOptions{
			Mode:     mode,
			Category: category,
			Sort:     sort,
			Order:    order,
			Page:     page,
			PerPage:  perPage,
		})
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, pagedArticlesDTO{
			Articles: toArticleDTOs(articles),
			Total:    total,
			Page:     page,
			PerPage:  perPage,
		})
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

		category, sort, order, page, perPage, err := parseListQuery(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		articles, total, err := uc.ExecuteFiltered(r.Context(), usecase.SearchFilterOptions{
			Keyword:        keyword,
			BookmarkedOnly: bookmarkedOnly,
			Category:       category,
			Sort:           sort,
			Order:          order,
			Page:           page,
			PerPage:        perPage,
		})
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, pagedArticlesDTO{
			Articles: toArticleDTOs(articles),
			Total:    total,
			Page:     page,
			PerPage:  perPage,
		})
	}
}

func handleListCategories(uc *usecase.ListCategoriesUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		categories, err := uc.Execute(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if categories == nil {
			categories = []string{}
		}
		writeJSON(w, http.StatusOK, categories)
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
