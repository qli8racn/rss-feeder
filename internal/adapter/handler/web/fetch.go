package web

import (
	"net/http"

	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// fetchResultDTO は最新フィード取得結果を JSON で表現するための DTO。
type fetchResultDTO struct {
	Saved   int `json:"saved"`
	Skipped int `json:"skipped"`
	Errors  int `json:"errors"`
}

func handleFetchLatest(fetchUC *usecase.FetchUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := fetchUC.ExecuteAll(r.Context())
		if err != nil && len(result.Feeds) == 0 {
			// フィード一覧の取得自体に失敗した場合のみ 500 として返す。
			// 個々のフィード取得の失敗は saved/skipped/errors の件数で表現するため、
			// その場合の err（ExecuteAll が TotalErrors()>0 のとき返す err）は無視してよい。
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, fetchResultDTO{
			Saved:   result.TotalSaved(),
			Skipped: result.TotalSkipped(),
			Errors:  result.TotalErrors(),
		})
	}
}
