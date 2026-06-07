package usecase

import (
	"context"
	"errors"
	"testing"
)

type mockMaintainer struct {
	vacuumErr    error
	integrityRes []string
	integrityErr error
}

func (m *mockMaintainer) Vacuum(_ context.Context) error { return m.vacuumErr }
func (m *mockMaintainer) IntegrityCheck(_ context.Context) ([]string, error) {
	return m.integrityRes, m.integrityErr
}

func TestMaintenanceUsecase_IntegrityOK(t *testing.T) {
	uc := NewMaintenanceUsecase(&mockMaintainer{integrityRes: []string{"ok"}})

	result, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IntegrityOK {
		t.Error("expected IntegrityOK to be true")
	}
}

func TestMaintenanceUsecase_IntegrityFailure(t *testing.T) {
	uc := NewMaintenanceUsecase(&mockMaintainer{
		integrityRes: []string{"FAIL", "*** in table articles ***"},
	})

	result, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IntegrityOK {
		t.Error("expected IntegrityOK to be false")
	}
	if len(result.Details) != 2 {
		t.Errorf("details count: got %d, want 2", len(result.Details))
	}
}

func TestMaintenanceUsecase_VacuumError(t *testing.T) {
	uc := NewMaintenanceUsecase(&mockMaintainer{vacuumErr: errors.New("disk full")})

	if _, err := uc.Execute(context.Background()); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestMaintenanceUsecase_IntegrityCheckError(t *testing.T) {
	uc := NewMaintenanceUsecase(&mockMaintainer{integrityErr: errors.New("db error")})

	if _, err := uc.Execute(context.Background()); err == nil {
		t.Error("expected error, got nil")
	}
}
