package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/samber/do/v2"

	"github.com/qli8racn/rss-feeder/internal/config"
)

// closableLogger は *slog.Logger をラップし、ファイル出力時のクローズを管理する。
// DI コンテナの Shutdown 時に Shutdown() が呼ばれ、ファイルをクローズする。
type closableLogger struct {
	*slog.Logger
	closer io.Closer // nil の場合（stderr/stdout）はクローズ不要
}

func (l *closableLogger) Shutdown() {
	if l.closer != nil {
		_ = l.closer.Close() // シャットダウン時のクローズ失敗は回復不能のため無視する
	}
}

// NewLogger は config の log セクションをもとに *slog.Logger を構築して DI コンテナに提供する。
// ファイル出力の場合は追記モードで open し、DI シャットダウン時に Shutdown() でクローズする。
func NewLogger(i do.Injector) (*slog.Logger, error) {
	cfg := do.MustInvoke[*config.Config](i)

	w, closer, err := openWriter(cfg.Log.Output)
	if err != nil {
		return nil, err
	}

	handler := newHandler(w, cfg.Log.Format)
	logger := &closableLogger{
		Logger: slog.New(handler),
		closer: closer,
	}

	// DI コンテナに Shutdowner として登録することでプロセス終了時にファイルをクローズする。
	do.ProvideValue[do.Shutdowner](i, logger)

	return logger.Logger, nil
}

// openWriter は output 設定に応じた io.Writer と、クローズが必要な場合の io.Closer を返す。
func openWriter(output string) (io.Writer, io.Closer, error) {
	switch output {
	case "", "stderr":
		return os.Stderr, nil, nil
	case "stdout":
		return os.Stdout, nil, nil
	default:
		f, err := os.OpenFile(output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("ログファイルのオープンに失敗しました: %w", err)
		}
		return f, f, nil
	}
}

// newHandler は format 設定に応じた slog.Handler を返す。
func newHandler(w io.Writer, format string) slog.Handler {
	if format == "json" {
		return slog.NewJSONHandler(w, nil)
	}
	return slog.NewTextHandler(w, nil)
}
