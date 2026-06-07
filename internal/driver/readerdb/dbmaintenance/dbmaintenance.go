package dbmaintenance

import (
	"context"
	"database/sql"

	adaptermaint "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/dbmaintenance"
	"github.com/samber/do/v2"
)

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

func (m *maintainer) IntegrityCheck(ctx context.Context) ([]string, error) {
	rows, err := m.db.QueryContext(ctx, `PRAGMA integrity_check`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, rows.Err()
}
