package usecase

import (
	"context"

	auditrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/auditlog"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

type AuditUsecase struct {
	auditRepo auditrepo.Repository
}

func NewAuditUsecase(repo auditrepo.Repository) *AuditUsecase {
	return &AuditUsecase{auditRepo: repo}
}

func (uc *AuditUsecase) Execute(ctx context.Context, log domain.AuditLog) error {
	return uc.auditRepo.Save(ctx, log)
}
