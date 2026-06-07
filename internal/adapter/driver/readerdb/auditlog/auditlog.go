package auditlog

import (
	"context"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

type Repository interface {
	Save(ctx context.Context, log domain.AuditLog) error
}
