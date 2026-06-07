package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

type mockAuditRepo struct {
	saved   []domain.AuditLog
	saveErr error
}

func (m *mockAuditRepo) Save(_ context.Context, log domain.AuditLog) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = append(m.saved, log)
	return nil
}

func TestAuditUsecase_SavesLog(t *testing.T) {
	repo := &mockAuditRepo{}
	uc := NewAuditUsecase(repo)

	articleID := int64(42)
	log := domain.AuditLog{Action: domain.ActionBookmark, ArticleID: &articleID}

	if err := uc.Execute(context.Background(), log); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.saved) != 1 {
		t.Fatalf("saved count: got %d, want 1", len(repo.saved))
	}
	if repo.saved[0].Action != domain.ActionBookmark {
		t.Errorf("action: got %s, want %s", repo.saved[0].Action, domain.ActionBookmark)
	}
	if repo.saved[0].ArticleID == nil || *repo.saved[0].ArticleID != 42 {
		t.Errorf("article_id: got %v, want 42", repo.saved[0].ArticleID)
	}
}

func TestAuditUsecase_PropagatesRepoError(t *testing.T) {
	repo := &mockAuditRepo{saveErr: errors.New("db error")}
	uc := NewAuditUsecase(repo)

	err := uc.Execute(context.Background(), domain.AuditLog{Action: domain.ActionFetch})
	if err == nil {
		t.Error("expected error, got nil")
	}
}
