package readerpg

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/samber/do/v2"

	"github.com/qli8racn/rss-feeder/internal/config"
)

// NewClient は config.yml の db.supabase.dsn を用いて Supabase（Postgres）への接続を作る。
func NewClient(i do.Injector) (*sql.DB, error) {
	cfg := do.MustInvoke[*config.Config](i)
	db, err := sql.Open("pgx", cfg.DB.Supabase.DSN)
	if err != nil {
		return nil, fmt.Errorf("supabase(postgres)への接続に失敗しました: %w", err)
	}
	return db, nil
}
