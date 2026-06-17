package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/adapter/handler/web/openapi"
	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

func toArticleDTO(a domain.Article) openapi.Article {
	return openapi.Article{
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

func toArticleDTOs(articles []domain.Article) []openapi.Article {
	dtos := make([]openapi.Article, len(articles))
	for i, a := range articles {
		dtos[i] = toArticleDTO(a)
	}
	return dtos
}

// parseListQuery は記事一覧・検索 API に共通の category/sort/order/page/per_page クエリパラメータを解析する。
func parseListQuery(r *http.Request) (category, sort, order string, page, perPage int, err error) {
	category = r.URL.Query().Get("category")

	sort = r.URL.Query().Get("sort")
	if sort == "" {
		sort = string(openapi.SortPublishedAt)
	} else if !articlerepo.ValidSortFields[sort] {
		return "", "", "", 0, 0, fmt.Errorf("sort は title, publisher, category, published_at のいずれかを指定してください")
	}

	orderParam := openapi.Order(r.URL.Query().Get("order"))
	if orderParam == "" {
		orderParam = openapi.OrderDesc
	} else if !orderParam.Valid() {
		return "", "", "", 0, 0, fmt.Errorf("order は asc, desc のいずれかを指定してください")
	}
	order = string(orderParam)

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

// ListArticlesHandler は GET /api/articles を処理する。
func ListArticlesHandler(uc *usecase.ListUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		modeParam := openapi.Mode(r.URL.Query().Get("mode"))
		if modeParam == "" {
			modeParam = openapi.ModeAll
		} else if !modeParam.Valid() {
			writeJSONError(w, http.StatusBadRequest, "mode は all, unread, bookmarked のいずれかを指定してください")
			return
		}

		var mode usecase.ListMode
		switch modeParam {
		case openapi.ModeUnread:
			mode = usecase.ListModeUnread
		case openapi.ModeBookmarked:
			mode = usecase.ListModeBookmarked
		default:
			mode = usecase.ListModeAll
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

		writeJSON(w, http.StatusOK, openapi.PagedArticles{
			Articles: toArticleDTOs(articles),
			Total:    total,
			Page:     page,
			PerPage:  perPage,
		})
	}
}

// SearchArticlesHandler は GET /api/articles/search を処理する。
func SearchArticlesHandler(uc *usecase.SearchUsecase) http.HandlerFunc {
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

		writeJSON(w, http.StatusOK, openapi.PagedArticles{
			Articles: toArticleDTOs(articles),
			Total:    total,
			Page:     page,
			PerPage:  perPage,
		})
	}
}

// BookmarkArticleHandler は POST /api/articles/{id}/bookmark を処理する。
func BookmarkArticleHandler(bookmarkUC *usecase.BookmarkUsecase, auditUC *usecase.AuditUsecase) http.HandlerFunc {
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
