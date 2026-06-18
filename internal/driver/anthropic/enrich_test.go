package anthropic

import "testing"

func TestTruncateRunes_ShorterThanMax(t *testing.T) {
	got := truncateRunes("hello", 10)
	if got != "hello" {
		t.Errorf("truncateRunes: got %q, want %q", got, "hello")
	}
}

func TestTruncateRunes_EqualToMax(t *testing.T) {
	got := truncateRunes("hello", 5)
	if got != "hello" {
		t.Errorf("truncateRunes: got %q, want %q", got, "hello")
	}
}

func TestTruncateRunes_LongerThanMax(t *testing.T) {
	got := truncateRunes("hello world", 5)
	if got != "hello" {
		t.Errorf("truncateRunes: got %q, want %q", got, "hello")
	}
}

func TestTruncateRunes_MultiByteRunes(t *testing.T) {
	// マルチバイト文字（日本語）を含む文字列でも、バイト単位ではなくルーン単位で
	// 切り詰められ、文字が破壊されないことを確認する。
	got := truncateRunes("こんにちは世界", 5)
	if got != "こんにちは" {
		t.Errorf("truncateRunes: got %q, want %q", got, "こんにちは")
	}
}

func TestTruncateRunes_Empty(t *testing.T) {
	got := truncateRunes("", 5)
	if got != "" {
		t.Errorf("truncateRunes: got %q, want empty", got)
	}
}
