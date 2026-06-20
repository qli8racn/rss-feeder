package usecase

import (
	"context"
	"time"

	"github.com/qli8racn/rss-feeder/internal/adapter/driver/feedenrich"
)

// enrichTimeout はサブプロセス呼び出し（Claude API呼び出しを含む）が無期限にハングしないための上限。
// resolve_feed_url.go の stepThreeTimeout と同じ考え方で同じ値を採用する。
const enrichTimeout = 30 * time.Second

// TriggerEnrichUsecase は指定フィードの新規記事に対する要約・カテゴライズをトリガーする。
type TriggerEnrichUsecase struct {
	agent feedenrich.Agent
}

func NewTriggerEnrichUsecase(agent feedenrich.Agent) *TriggerEnrichUsecase {
	return &TriggerEnrichUsecase{agent: agent}
}

func (uc *TriggerEnrichUsecase) Execute(ctx context.Context, feedURL string, limit int) error {
	ctx, cancel := context.WithTimeout(ctx, enrichTimeout)
	defer cancel()
	return uc.agent.Enrich(ctx, feedURL, limit)
}
