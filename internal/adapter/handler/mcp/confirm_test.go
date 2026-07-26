package mcp

import (
	"errors"
	"testing"
)

func TestRequireConfirm(t *testing.T) {
	if err := requireConfirm(true); err != nil {
		t.Errorf("confirm=true: unexpected error: %v", err)
	}

	err := requireConfirm(false)
	if !errors.Is(err, ErrConfirmationRequired) {
		t.Errorf("confirm=false: got %v, want ErrConfirmationRequired", err)
	}
}
