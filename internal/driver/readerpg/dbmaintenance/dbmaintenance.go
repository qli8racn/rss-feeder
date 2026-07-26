package dbmaintenance

import (
	"context"
	"database/sql"

	adaptermaint "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/dbmaintenance"
	"github.com/samber/do/v2"
)

var _ adaptermaint.Maintainer = (*maintainer)(nil)

type maintainer struct {
	db *sql.DB
}

func NewMaintainer(i do.Injector) (adaptermaint.Maintainer, error) {
	return &maintainer{db: do.MustInvoke[*sql.DB](i)}, nil
}

func (m *maintainer) Vacuum(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, `VACUUM`)
	return err
}

// IntegrityCheck: PostgresにはSQLiteの `PRAGMA integrity_check` に相当する組み込み機能が
// ないため、フェーズ1では未サポートである旨を説明的な戻り値として返す（design.md参照）。
func (m *maintainer) IntegrityCheck(ctx context.Context) ([]string, error) {
	return []string{"postgres: integrity check is not supported"}, nil
}
