package anthropic

import "testing"

func TestParseLimitInput_EmptyUsesDefault(t *testing.T) {
	got, err := parseLimitInput("", 10, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 10 {
		t.Errorf("limit: got %d, want 10", got)
	}
}

func TestParseLimitInput_ZeroUsesDefault(t *testing.T) {
	got, err := parseLimitInput(`{"limit":0}`, 10, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 10 {
		t.Errorf("limit: got %d, want 10", got)
	}
}

func TestParseLimitInput_NegativeUsesDefault(t *testing.T) {
	got, err := parseLimitInput(`{"limit":-1}`, 10, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 10 {
		t.Errorf("limit: got %d, want 10", got)
	}
}

func TestParseLimitInput_WithinRange(t *testing.T) {
	got, err := parseLimitInput(`{"limit":20}`, 10, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 20 {
		t.Errorf("limit: got %d, want 20", got)
	}
}

func TestParseLimitInput_ClampedToMax(t *testing.T) {
	got, err := parseLimitInput(`{"limit":1000}`, 10, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 50 {
		t.Errorf("limit: got %d, want 50 (clamped)", got)
	}
}

func TestParseLimitInput_NoMaxMeansUnbounded(t *testing.T) {
	got, err := parseLimitInput(`{"limit":1000}`, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 1000 {
		t.Errorf("limit: got %d, want 1000 (no upper bound)", got)
	}
}

func TestParseLimitInput_InvalidJSON(t *testing.T) {
	if _, err := parseLimitInput("not json", 10, 50); err == nil {
		t.Error("expected error for invalid JSON input, got nil")
	}
}
