package anthropic

import (
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

func TestEstimateCostUSD_KnownModel(t *testing.T) {
	usage := anthropic.Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000}
	got := estimateCostUSD(anthropic.ModelClaudeHaiku4_5, usage)
	want := 1.00 + 5.00 // Haiku 4.5: $1/$5 per 1M トークン
	if got != want {
		t.Errorf("estimateCostUSD: got %v, want %v", got, want)
	}
}

func TestEstimateCostUSD_UnknownModel(t *testing.T) {
	usage := anthropic.Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000}
	got := estimateCostUSD("claude-unknown-model", usage)
	if got != 0 {
		t.Errorf("estimateCostUSD: got %v, want 0 for unknown model", got)
	}
}

func TestEstimateCostUSD_CacheTokens(t *testing.T) {
	// cache_creation は通常入力の1.25倍、cache_read は0.1倍として概算される。
	usage := anthropic.Usage{
		CacheCreationInputTokens: 1_000_000,
		CacheReadInputTokens:     1_000_000,
	}
	got := estimateCostUSD(anthropic.ModelClaudeHaiku4_5, usage)
	want := 1.25*1.00 + 0.1*1.00 // Haiku 4.5 の入力単価 $1 に対して適用
	if got != want {
		t.Errorf("estimateCostUSD: got %v, want %v", got, want)
	}
}

func TestEstimateCostUSD_ZeroUsage(t *testing.T) {
	got := estimateCostUSD(anthropic.ModelClaudeOpus4_8, anthropic.Usage{})
	if got != 0 {
		t.Errorf("estimateCostUSD: got %v, want 0", got)
	}
}

func TestAddUsage_AccumulatesAcrossMultipleTurns(t *testing.T) {
	var total anthropic.Usage
	total = addUsage(total, anthropic.Usage{InputTokens: 10, OutputTokens: 5, CacheCreationInputTokens: 1, CacheReadInputTokens: 2})
	total = addUsage(total, anthropic.Usage{InputTokens: 20, OutputTokens: 8, CacheCreationInputTokens: 3, CacheReadInputTokens: 4})

	if total.InputTokens != 30 {
		t.Errorf("InputTokens: got %d, want 30", total.InputTokens)
	}
	if total.OutputTokens != 13 {
		t.Errorf("OutputTokens: got %d, want 13", total.OutputTokens)
	}
	if total.CacheCreationInputTokens != 4 {
		t.Errorf("CacheCreationInputTokens: got %d, want 4", total.CacheCreationInputTokens)
	}
	if total.CacheReadInputTokens != 6 {
		t.Errorf("CacheReadInputTokens: got %d, want 6", total.CacheReadInputTokens)
	}
}

func TestAddUsage_ZeroDeltaIsNoOp(t *testing.T) {
	total := anthropic.Usage{InputTokens: 10, OutputTokens: 5}
	got := addUsage(total, anthropic.Usage{})
	if got.InputTokens != total.InputTokens || got.OutputTokens != total.OutputTokens {
		t.Errorf("addUsage: got input=%d output=%d, want input=%d output=%d (unchanged)",
			got.InputTokens, got.OutputTokens, total.InputTokens, total.OutputTokens)
	}
}
