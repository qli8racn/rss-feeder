package feedenrich

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	adapterfeedenrich "github.com/qli8racn/rss-feeder/internal/adapter/driver/feedenrich"
)

// writeScript はテスト用のシェルスクリプトを作成し、そのパスを返す。
func writeScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-rss-agent.sh")
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}
	return path
}

func TestSubprocessAgent_Success(t *testing.T) {
	script := writeScript(t, "#!/bin/sh\necho \"5 件の記事を要約・分類しました\"\n")
	agent := NewSubprocessAgent(script)

	if err := agent.Enrich(context.Background(), "https://example.com/feed.xml", 5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubprocessAgent_NonZeroExit(t *testing.T) {
	script := writeScript(t, "#!/bin/sh\necho \"enrich failed\" >&2\nexit 1\n")
	agent := NewSubprocessAgent(script)

	err := agent.Enrich(context.Background(), "https://example.com/feed.xml", 5)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "enrich failed") {
		t.Errorf("error should contain stderr output, got: %v", err)
	}
}

func TestSubprocessAgent_BinaryNotFound(t *testing.T) {
	agent := NewSubprocessAgent(filepath.Join(t.TempDir(), "does-not-exist"))

	err := agent.Enrich(context.Background(), "https://example.com/feed.xml", 5)
	if !errors.Is(err, adapterfeedenrich.ErrAgentUnavailable) {
		t.Errorf("expected ErrAgentUnavailable, got %v", err)
	}
}

func TestSubprocessAgent_ContextCanceled(t *testing.T) {
	script := writeScript(t, "#!/bin/sh\nsleep 1\n")
	agent := NewSubprocessAgent(script)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := agent.Enrich(ctx, "https://example.com/feed.xml", 5)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
