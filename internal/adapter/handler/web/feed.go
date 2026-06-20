package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
	"github.com/qli8racn/rss-feeder/internal/adapter/handler/web/openapi"
	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// addFeedTimeout はフィードURLの探索（直接判定→標準探索→AIフォールバック）から
// 登録までの全体タイムアウト。直列実行される各ステップの既存タイムアウトの合計
// （30+15+30秒 ≒ 75秒）に安全マージンを加えた値。
const addFeedTimeout = 80 * time.Second

func toFeedDTO(f domain.Feed) openapi.Feed {
	var lastFetched *time.Time
	if !f.LastFetched.IsZero() {
		t := f.LastFetched
		lastFetched = &t
	}
	return openapi.Feed{
		ID:          f.ID,
		FeedURL:     f.FeedURL,
		Title:       f.Title,
		LastFetched: lastFetched,
		CreatedAt:   f.CreatedAt,
	}
}

func toFeedDTOs(feeds []domain.Feed) []openapi.Feed {
	dtos := make([]openapi.Feed, len(feeds))
	for i, f := range feeds {
		dtos[i] = toFeedDTO(f)
	}
	return dtos
}

// ListFeedsHandler は GET /api/feeds を処理する。
func ListFeedsHandler(uc *usecase.ListFeedsUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		feeds, err := uc.Execute(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, toFeedDTOs(feeds))
	}
}

// AddFeedHandler は POST /api/feeds を処理する。
// 入力URLを resolveFeedURLUC でフィードURLに解決した上で addFeedUC に渡す。
func AddFeedHandler(addFeedUC *usecase.AddFeedUsecase, resolveFeedURLUC *usecase.ResolveFeedURLUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req openapi.AddFeedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "リクエストボディの形式が不正です")
			return
		}
		if req.FeedURL == "" {
			writeJSONError(w, http.StatusBadRequest, "feed_url は必須です")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), addFeedTimeout)
		defer cancel()

		resolvedURL, err := resolveFeedURLUC.Execute(ctx, req.FeedURL)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				writeJSONError(w, http.StatusGatewayTimeout, "フィードURLの探索がタイムアウトしました")
				return
			}
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		feed, err := addFeedUC.Execute(ctx, resolvedURL)
		if err != nil {
			if errors.Is(err, feedrepo.ErrAlreadyExists) {
				writeJSONError(w, http.StatusConflict, err.Error())
				return
			}
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, toFeedDTO(*feed))
	}
}

// RemoveFeedHandler は DELETE /api/feeds/{id} を処理する。
func RemoveFeedHandler(uc *usecase.RemoveFeedUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "id は整数で指定してください")
			return
		}

		if err := uc.Execute(r.Context(), id); err != nil {
			if errors.Is(err, feedrepo.ErrNotFound) {
				writeJSONError(w, http.StatusNotFound, err.Error())
				return
			}
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
