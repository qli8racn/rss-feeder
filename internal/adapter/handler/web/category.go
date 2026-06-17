package web

import (
	"net/http"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// ListCategoriesHandler は GET /api/categories を処理する。
func ListCategoriesHandler(uc *usecase.ListCategoriesUsecase) http.HandlerFunc {
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
