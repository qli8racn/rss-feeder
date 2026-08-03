package readerpg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/samber/do/v2"

	"github.com/qli8racn/rss-feeder/internal/config"
)

// pingTimeout は起動時の疎通確認に使うタイムアウト。Supabase側がスリープから復帰する
// 場合など多少の遅延を見込みつつ、DSN誤りなどに気づけるよう長すぎない値にする。
const pingTimeout = 5 * time.Second

// Supabase（無料プラン）はプロジェクトあたりの同時接続数に上限があるため、
// アプリ側のコネクションプールを控えめな値に抑える。cmd/web・cmd/rss-feeder・cmd/agent・
// cmd/mcp がそれぞれ別プロセスとして同時に起動されうる（内部的にrss-agentをサブプロセス
// 起動するケースもある）ことを踏まえ、1プロセスあたりの上限を小さくしておく。
const (
	maxOpenConns    = 5
	maxIdleConns    = 2
	connMaxLifetime = 5 * time.Minute
)

// NewClient は config.yml の db.supabase.dsn を用いて Supabase（Postgres）への接続を作る。
func NewClient(i do.Injector) (*sql.DB, error) {
	cfg, err := do.Invoke[*config.Config](i)
	if err != nil {
		return nil, err
	}
	if cfg.DB.Supabase.DSN == "" {
		return nil, errors.New("db.driver: supabase が指定されていますが db.supabase.dsn が設定されていません")
	}

	db, err := sql.Open("pgx", cfg.DB.Supabase.DSN)
	if err != nil {
		return nil, fmt.Errorf("supabase(postgres)への接続に失敗しました: %w", err)
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)

	// sql.Openは接続確認を行わないため、DSN誤り・ネットワーク不通等を起動時に検知するために
	// 明示的にPingする。
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("supabase(postgres)への疎通確認に失敗しました（dsnの設定を確認してください）: %w", err)
	}

	return db, nil
}
