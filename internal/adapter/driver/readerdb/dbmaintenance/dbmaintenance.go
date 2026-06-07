package dbmaintenance

import "context"

type Maintainer interface {
	Vacuum(ctx context.Context) error
	IntegrityCheck(ctx context.Context) ([]string, error)
}
