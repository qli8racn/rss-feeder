package auditlog

import (
	"context"
	"database/sql"

	adapterauditlog "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/auditlog"
	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/samber/do/v2"
)

var _ adapterauditlog.Repository = (*repository)(nil)

type repository struct {
	db *sql.DB
}

func NewRepository(i do.Injector) (adapterauditlog.Repository, error) {
	return &repository{db: do.MustInvoke[*sql.DB](i)}, nil
}

func (r *repository) Save(ctx context.Context, log domain.AuditLog) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO audit_log (action, article_id, old_state, new_state)
		VALUES (?, ?, ?, ?)
	`, log.Action, log.ArticleID, nullableStr(log.OldState), nullableStr(log.NewState))
	return err
}

func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
