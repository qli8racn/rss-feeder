package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/qli8racn/rss-feeder/internal/domain"
)

type mockUserRepo struct {
	users       map[string]*domain.User
	findErr     error
	createErr   error
	nextID      int64
	createCalls int
}

func (m *mockUserRepo) FindByName(_ context.Context, name string) (*domain.User, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.users[name], nil
}

func (m *mockUserRepo) Create(_ context.Context, name string) (*domain.User, error) {
	m.createCalls++
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.nextID++
	u := &domain.User{ID: m.nextID, Name: name}
	if m.users == nil {
		m.users = map[string]*domain.User{}
	}
	m.users[name] = u
	return u, nil
}

func TestResolveUserUsecase_ReturnsExistingUser(t *testing.T) {
	repo := &mockUserRepo{users: map[string]*domain.User{
		"alice": {ID: 1, Name: "alice"},
	}}
	uc := NewResolveUserUsecase(repo)

	u, err := uc.Execute(context.Background(), "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != 1 || u.Name != "alice" {
		t.Errorf("got %+v, want ID=1 Name=alice", u)
	}
	if repo.createCalls != 0 {
		t.Errorf("Create should not be called when user already exists, got %d calls", repo.createCalls)
	}
}

func TestResolveUserUsecase_CreatesNewUser(t *testing.T) {
	repo := &mockUserRepo{}
	uc := NewResolveUserUsecase(repo)

	u, err := uc.Execute(context.Background(), "bob")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Name != "bob" {
		t.Errorf("Name: got %q, want %q", u.Name, "bob")
	}
	if repo.createCalls != 1 {
		t.Errorf("Create calls: got %d, want 1", repo.createCalls)
	}
}

func TestResolveUserUsecase_FindError(t *testing.T) {
	repo := &mockUserRepo{findErr: errors.New("db error")}
	uc := NewResolveUserUsecase(repo)

	if _, err := uc.Execute(context.Background(), "alice"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResolveUserUsecase_CreateRaceCondition_FallsBackToFindByName(t *testing.T) {
	// Create がUNIQUE制約違反で失敗しても、その時点で既に存在するユーザーを
	// FindByName で見つけて返す（他プロセスが同時に同名ユーザーを作成したレース条件を吸収する）。
	repo := &raceUserRepo{existing: &domain.User{ID: 5, Name: "alice"}}
	uc := NewResolveUserUsecase(repo)

	u, err := uc.Execute(context.Background(), "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != 5 {
		t.Errorf("got %+v, want the concurrently created user (ID=5)", u)
	}
}

// raceUserRepo は、初回 FindByName では nil を返すが、Create が失敗した後の
// 2回目の FindByName では既存ユーザーを返す（他プロセスによる同時作成のレース条件を再現する）。
type raceUserRepo struct {
	existing  *domain.User
	findCalls int
}

func (m *raceUserRepo) FindByName(_ context.Context, _ string) (*domain.User, error) {
	m.findCalls++
	if m.findCalls == 1 {
		return nil, nil
	}
	return m.existing, nil
}

func (m *raceUserRepo) Create(_ context.Context, _ string) (*domain.User, error) {
	return nil, errors.New("UNIQUE constraint failed: users.name")
}

func TestResolveUserUsecase_CreateErrorWithoutExistingUser(t *testing.T) {
	repo := &mockUserRepo{createErr: errors.New("db error")}
	uc := NewResolveUserUsecase(repo)

	if _, err := uc.Execute(context.Background(), "alice"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
