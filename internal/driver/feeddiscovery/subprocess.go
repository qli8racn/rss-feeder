package feeddiscovery

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"strings"

	adapterfeeddiscovery "github.com/qli8racn/rss-feeder/internal/adapter/driver/feeddiscovery"
)

type subprocessAgent struct {
	binPath string
	sem     chan struct{}
}

// NewSubprocessAgent は bin/rss-agent discover-feed をサブプロセスとして実行する
// feeddiscovery.Agent 実装を返す。
// maxConcurrency は同時実行数の上限。1未満の場合は1に補正する
// （0だとセマフォへの送信が永久にブロックし、全リクエストがハングするため）。
func NewSubprocessAgent(binPath string, maxConcurrency int) adapterfeeddiscovery.Agent {
	if maxConcurrency < 1 {
		maxConcurrency = 1
	}
	return &subprocessAgent{binPath: binPath, sem: make(chan struct{}, maxConcurrency)}
}

func (a *subprocessAgent) Discover(ctx context.Context, url string) (string, error) {
	select {
	case a.sem <- struct{}{}:
		defer func() { <-a.sem }()
	case <-ctx.Done():
		return "", ctx.Err()
	}

	cmd := exec.CommandContext(ctx, a.binPath, "discover-feed", url)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
			return "", adapterfeeddiscovery.ErrAgentUnavailable
		}
		return "", fmt.Errorf("rss-agent discover-feed に失敗しました: %s: %w", strings.TrimSpace(stderr.String()), err)
	}

	return strings.TrimSpace(stdout.String()), nil
}
