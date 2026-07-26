package usecase

import (
	"context"
	"errors"
	"testing"
)

func TestMarkReadUsecase_Execute(t *testing.T) {
	repo := &mockListArticleRepo{}
	uc := NewMarkReadUsecase(repo)

	if err := uc.Execute(context.Background(), []int64{1, 2}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.markedIDs) != 2 || repo.markedIDs[0] != 1 || repo.markedIDs[1] != 2 {
		t.Errorf("markedIDs: got %+v, want [1 2]", repo.markedIDs)
	}
}

func TestMarkReadUsecase_Execute_EmptyIDs(t *testing.T) {
	repo := &mockListArticleRepo{}
	uc := NewMarkReadUsecase(repo)

	err := uc.Execute(context.Background(), nil)
	if !errors.Is(err, ErrMarkReadIDsRequired) {
		t.Errorf("err: got %v, want ErrMarkReadIDsRequired", err)
	}
	if len(repo.markedIDs) != 0 {
		t.Errorf("markedIDs: got %+v, want none", repo.markedIDs)
	}
}

func TestMarkReadUsecase_Execute_RepoError(t *testing.T) {
	wantErr := errors.New("db error")
	repo := &mockListArticleRepo{markErr: wantErr}
	uc := NewMarkReadUsecase(repo)

	err := uc.Execute(context.Background(), []int64{1})
	if !errors.Is(err, wantErr) {
		t.Errorf("err: got %v, want wrapping %v", err, wantErr)
	}
}
