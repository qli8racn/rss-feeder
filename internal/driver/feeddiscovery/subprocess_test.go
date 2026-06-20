package feeddiscovery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	adapterfeeddiscovery "github.com/qli8racn/rss-feeder/internal/adapter/driver/feeddiscovery"
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
	script := writeScript(t, "#!/bin/sh\necho \"https://example.com/feed.xml\"\n")
	agent := NewSubprocessAgent(script, 2)

	got, err := agent.Discover(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://example.com/feed.xml" {
		t.Errorf("got %q, want %q", got, "https://example.com/feed.xml")
	}
}

func TestSubprocessAgent_NonZeroExit(t *testing.T) {
	script := writeScript(t, "#!/bin/sh\necho \"feed not found\" >&2\nexit 1\n")
	agent := NewSubprocessAgent(script, 2)

	_, err := agent.Discover(context.Background(), "https://example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "feed not found") {
		t.Errorf("error should contain stderr output, got: %v", err)
	}
}

func TestSubprocessAgent_BinaryNotFound(t *testing.T) {
	agent := NewSubprocessAgent(filepath.Join(t.TempDir(), "does-not-exist"), 2)

	_, err := agent.Discover(context.Background(), "https://example.com")
	if !errors.Is(err, adapterfeeddiscovery.ErrAgentUnavailable) {
		t.Errorf("expected ErrAgentUnavailable, got %v", err)
	}
}

// TestSubprocessAgent_ConcurrencyLimit は maxConcurrency=1 のとき、複数の Discover 呼び出しが
// 同時に実行されず直列化されることを検証する。各実行が開始・終了のログをファイルに追記し、
// 連続する2つの "start" の間に必ず "end" が挟まることを確認することで直列化を確認する。
func TestSubprocessAgent_ConcurrencyLimit(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log.txt")
	script := writeScript(t, fmt.Sprintf(
		"#!/bin/sh\necho start >> %q\nsleep 0.1\necho end >> %q\necho \"https://example.com/feed.xml\"\n",
		logFile, logFile,
	))

	agent := NewSubprocessAgent(script, 1)

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := agent.Discover(context.Background(), "https://example.com"); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	b, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	lines := strings.Fields(strings.TrimSpace(string(b)))
	if len(lines) != 6 {
		t.Fatalf("expected 6 log lines (3x start/end), got %d: %v", len(lines), lines)
	}
	for i := 0; i < len(lines); i += 2 {
		if lines[i] != "start" || lines[i+1] != "end" {
			t.Errorf("executions overlapped (not serialized): %v", lines)
			break
		}
	}
}

func TestSubprocessAgent_ContextCanceledWhileWaiting(t *testing.T) {
	script := writeScript(t, "#!/bin/sh\nsleep 1\necho \"https://example.com/feed.xml\"\n")
	agent := NewSubprocessAgent(script, 1)

	// 同時実行数の上限(1)を埋めておく
	holderCtx, holderCancel := context.WithCancel(context.Background())
	defer holderCancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = agent.Discover(holderCtx, "https://example.com")
	}()
	time.Sleep(20 * time.Millisecond) // holder がセマフォを確保するのを待つ

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 即座にキャンセル

	_, err := agent.Discover(ctx, "https://example.com")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	holderCancel()
	<-done
}
