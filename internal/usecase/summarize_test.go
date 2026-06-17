package usecase

import (
	"context"
	"errors"
	"testing"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
)

type mockSummarizeAgent struct {
	runFn func(ctx context.Context, opts adapteranthropic.SummarizeOptions) (string, error)
}

func (m *mockSummarizeAgent) Run(ctx context.Context, opts adapteranthropic.SummarizeOptions) (string, error) {
	return m.runFn(ctx, opts)
}

func TestSummarizeUsecase_ReturnsResult(t *testing.T) {
	uc := NewSummarizeUsecase(&mockSummarizeAgent{
		runFn: func(_ context.Context, opts adapteranthropic.SummarizeOptions) (string, error) {
			if opts.Limit != 5 {
				t.Errorf("Limit: got %d, want 5", opts.Limit)
			}
			return "要約結果", nil
		},
	})

	result, err := uc.Execute(context.Background(), adapteranthropic.SummarizeOptions{Limit: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "要約結果" {
		t.Errorf("result: got %q, want %q", result, "要約結果")
	}
}

func TestSummarizeUsecase_AgentError(t *testing.T) {
	agentErr := errors.New("agent error")
	uc := NewSummarizeUsecase(&mockSummarizeAgent{
		runFn: func(_ context.Context, _ adapteranthropic.SummarizeOptions) (string, error) {
			return "", agentErr
		},
	})

	if _, err := uc.Execute(context.Background(), adapteranthropic.SummarizeOptions{}); !errors.Is(err, agentErr) {
		t.Errorf("expected agent error, got %v", err)
	}
}
