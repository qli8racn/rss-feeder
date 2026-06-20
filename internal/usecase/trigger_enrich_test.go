package usecase

import (
	"context"
	"errors"
	"testing"
)

type mockFeedEnrichAgent struct {
	enrichFn func(ctx context.Context, feedURL string, limit int) error
}

func (m *mockFeedEnrichAgent) Enrich(ctx context.Context, feedURL string, limit int) error {
	return m.enrichFn(ctx, feedURL, limit)
}

func TestTriggerEnrichUsecase_Success(t *testing.T) {
	var gotFeedURL string
	var gotLimit int
	agent := &mockFeedEnrichAgent{
		enrichFn: func(_ context.Context, feedURL string, limit int) error {
			gotFeedURL, gotLimit = feedURL, limit
			return nil
		},
	}
	uc := NewTriggerEnrichUsecase(agent)

	if err := uc.Execute(context.Background(), "https://example.com/feed", 5); err != nil {
		t.Fatalf("Execute: unexpected error: %v", err)
	}
	if gotFeedURL != "https://example.com/feed" || gotLimit != 5 {
		t.Errorf("Execute: got feedURL=%q limit=%d, want %q, 5", gotFeedURL, gotLimit, "https://example.com/feed")
	}
}

func TestTriggerEnrichUsecase_PropagatesAgentError(t *testing.T) {
	wantErr := errors.New("agent failed")
	agent := &mockFeedEnrichAgent{
		enrichFn: func(context.Context, string, int) error { return wantErr },
	}
	uc := NewTriggerEnrichUsecase(agent)

	if err := uc.Execute(context.Background(), "https://example.com/feed", 5); !errors.Is(err, wantErr) {
		t.Errorf("Execute: got %v, want it to wrap %v", err, wantErr)
	}
}

func TestTriggerEnrichUsecase_ContextCancellationPropagatesToAgent(t *testing.T) {
	// Execute 内部で context.WithTimeout(ctx, enrichTimeout) によりラップされた ctx が
	// agent.Enrich にそのまま渡され、キャンセル（enrichTimeout経過時のタイムアウトと同じ仕組み）が
	// 伝播することを確認する。30秒待つ実際のタイムアウトの代わりに、親ctxを事前キャンセルして
	// 同じ Done()/Err() の伝播経路を検証する。
	agent := &mockFeedEnrichAgent{
		enrichFn: func(ctx context.Context, _ string, _ int) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	uc := &TriggerEnrichUsecase{agent: agent}

	// 呼び出し元のctxを既にキャンセル済みにしておくことで、enrichTimeout を待たずに
	// 即座に DeadlineExceeded 相当（ctx.Err()）が伝播することを検証する。
	parentCtx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := uc.Execute(parentCtx, "https://example.com/feed", 5); !errors.Is(err, context.Canceled) {
		t.Errorf("Execute: got %v, want context.Canceled to propagate from agent.Enrich", err)
	}
}
