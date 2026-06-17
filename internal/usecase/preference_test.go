package usecase

import (
	"context"
	"errors"
	"testing"
)

type mockPreferenceAgent struct {
	runFn func(ctx context.Context) (string, error)
}

func (m *mockPreferenceAgent) Run(ctx context.Context) (string, error) {
	return m.runFn(ctx)
}

func TestPreferenceUsecase_ReturnsResult(t *testing.T) {
	uc := NewPreferenceUsecase(&mockPreferenceAgent{
		runFn: func(_ context.Context) (string, error) {
			return "分析結果", nil
		},
	})

	result, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "分析結果" {
		t.Errorf("result: got %q, want %q", result, "分析結果")
	}
}

func TestPreferenceUsecase_AgentError(t *testing.T) {
	agentErr := errors.New("agent error")
	uc := NewPreferenceUsecase(&mockPreferenceAgent{
		runFn: func(_ context.Context) (string, error) {
			return "", agentErr
		},
	})

	if _, err := uc.Execute(context.Background()); !errors.Is(err, agentErr) {
		t.Errorf("expected agent error, got %v", err)
	}
}
