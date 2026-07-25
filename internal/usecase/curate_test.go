package usecase

import (
	"context"
	"errors"
	"testing"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
)

type mockCurateAgent struct {
	runFn func(ctx context.Context, opts adapteranthropic.CurateOptions) (string, error)
}

func (m *mockCurateAgent) Run(ctx context.Context, opts adapteranthropic.CurateOptions) (string, error) {
	return m.runFn(ctx, opts)
}

func TestCurateUsecase_ReturnsResult(t *testing.T) {
	uc := NewCurateUsecase(&mockCurateAgent{
		runFn: func(_ context.Context, _ adapteranthropic.CurateOptions) (string, error) {
			return "推薦記事リスト", nil
		},
	})

	result, err := uc.Execute(context.Background(), adapteranthropic.CurateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "推薦記事リスト" {
		t.Errorf("result: got %q, want %q", result, "推薦記事リスト")
	}
}

func TestCurateUsecase_PassesOptions(t *testing.T) {
	var capturedOpts adapteranthropic.CurateOptions
	uc := NewCurateUsecase(&mockCurateAgent{
		runFn: func(_ context.Context, opts adapteranthropic.CurateOptions) (string, error) {
			capturedOpts = opts
			return "", nil
		},
	})

	_, _ = uc.Execute(context.Background(), adapteranthropic.CurateOptions{Limit: 10})
	if capturedOpts.Limit != 10 {
		t.Errorf("Limit: got %d, want 10", capturedOpts.Limit)
	}
}

func TestCurateUsecase_AgentError(t *testing.T) {
	agentErr := errors.New("agent error")
	uc := NewCurateUsecase(&mockCurateAgent{
		runFn: func(_ context.Context, _ adapteranthropic.CurateOptions) (string, error) {
			return "", agentErr
		},
	})

	if _, err := uc.Execute(context.Background(), adapteranthropic.CurateOptions{}); !errors.Is(err, agentErr) {
		t.Errorf("expected agent error, got %v", err)
	}
}
