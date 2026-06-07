package usecase

import (
	"context"
	"strings"

	adaptermaint "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/dbmaintenance"
)

type MaintenanceResult struct {
	IntegrityOK bool
	Details     []string
}

type MaintenanceUsecase struct {
	maintainer adaptermaint.Maintainer
}

func NewMaintenanceUsecase(m adaptermaint.Maintainer) *MaintenanceUsecase {
	return &MaintenanceUsecase{maintainer: m}
}

func (uc *MaintenanceUsecase) Execute(ctx context.Context) (MaintenanceResult, error) {
	if err := uc.maintainer.Vacuum(ctx); err != nil {
		return MaintenanceResult{}, err
	}
	lines, err := uc.maintainer.IntegrityCheck(ctx)
	if err != nil {
		return MaintenanceResult{}, err
	}
	ok := len(lines) == 1 && strings.EqualFold(lines[0], "ok")
	return MaintenanceResult{IntegrityOK: ok, Details: lines}, nil
}
