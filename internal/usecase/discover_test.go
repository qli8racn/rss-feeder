package usecase

import (
	"context"
	"errors"
	"testing"
)

type mockDiscoverAgent struct {
	runFn func(ctx context.Context) (string, error)
}

func (m *mockDiscoverAgent) Run(ctx context.Context) (string, error) {
	return m.runFn(ctx)
}

func TestDiscoverUsecase_ReturnsResult(t *testing.T) {
	uc := NewDiscoverUsecase(&mockDiscoverAgent{
		runFn: func(_ context.Context) (string, error) {
			return "推薦フィードリスト", nil
		},
	})

	result, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "推薦フィードリスト" {
		t.Errorf("result: got %q, want %q", result, "推薦フィードリスト")
	}
}

func TestDiscoverUsecase_AgentError(t *testing.T) {
	agentErr := errors.New("agent error")
	uc := NewDiscoverUsecase(&mockDiscoverAgent{
		runFn: func(_ context.Context) (string, error) {
			return "", agentErr
		},
	})

	if _, err := uc.Execute(context.Background()); !errors.Is(err, agentErr) {
		t.Errorf("expected agent error, got %v", err)
	}
}
