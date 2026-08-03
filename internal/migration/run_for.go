package migration

import (
	"database/sql"

	"github.com/qli8racn/rss-feeder/internal/config"
)

// RunFor は cfg.DB.IsSupabase() の値に応じて Run（SQLite）/ RunPostgres（Supabase）を
// 呼び分ける。4つの cmd/*/main.go に重複していた分岐をここに集約する。
func RunFor(cfg *config.Config, db *sql.DB) error {
	if cfg.DB.IsSupabase() {
		return RunPostgres(db)
	}
	return Run(db)
}
