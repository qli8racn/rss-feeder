package feedenrich

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"strconv"
	"strings"

	adapterfeedenrich "github.com/qli8racn/rss-feeder/internal/adapter/driver/feedenrich"
)

type subprocessAgent struct {
	binPath string
}

// NewSubprocessAgent は bin/rss-agent enrich をサブプロセスとして実行する
// feedenrich.Agent 実装を返す。cmd/rss-feeder（CLI）の単発実行からのみ呼ばれるため、
// feeddiscovery.subprocessAgent と異なり同時実行数を制限するセマフォは持たない。
func NewSubprocessAgent(binPath string) adapterfeedenrich.Agent {
	return &subprocessAgent{binPath: binPath}
}

func (a *subprocessAgent) Enrich(ctx context.Context, feedURL string, limit int) error {
	cmd := exec.CommandContext(ctx, a.binPath, "enrich", "--feed", feedURL, "--force", "--limit", strconv.Itoa(limit))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
			return adapterfeedenrich.ErrAgentUnavailable
		}
		return fmt.Errorf("rss-agent enrich に失敗しました: %s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return nil
}
