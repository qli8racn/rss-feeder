package usecase

import (
	"context"
	"errors"
	"testing"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
)

type mockEnrichAgent struct {
	runFn func(ctx context.Context, opts adapteranthropic.EnrichOptions) (int, error)
}

func (m *mockEnrichAgent) Run(ctx context.Context, opts adapteranthropic.EnrichOptions) (int, error) {
	return m.runFn(ctx, opts)
}

func TestEnrichUsecase_ReturnsCount(t *testing.T) {
	uc := NewEnrichUsecase(&mockEnrichAgent{
		runFn: func(_ context.Context, opts adapteranthropic.EnrichOptions) (int, error) {
			if !opts.Force {
				t.Errorf("Force: got false, want true")
			}
			return 3, nil
		},
	})

	n, err := uc.Execute(context.Background(), adapteranthropic.EnrichOptions{Force: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 3 {
		t.Errorf("count: got %d, want 3", n)
	}
}

func TestEnrichUsecase_AgentError(t *testing.T) {
	agentErr := errors.New("agent error")
	uc := NewEnrichUsecase(&mockEnrichAgent{
		runFn: func(_ context.Context, _ adapteranthropic.EnrichOptions) (int, error) {
			return 0, agentErr
		},
	})

	if _, err := uc.Execute(context.Background(), adapteranthropic.EnrichOptions{}); !errors.Is(err, agentErr) {
		t.Errorf("expected agent error, got %v", err)
	}
}
