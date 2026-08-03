package user

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/qli8racn/rss-feeder/internal/migration"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	if err := migration.Run(db); err != nil {
		t.Fatalf("migration: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close db: %v", err)
		}
	})
	return db
}

func newRepo(t *testing.T) *repository {
	t.Helper()
	return &repository{db: newTestDB(t)}
}

func TestUserRepository_FindByName_DefaultUserExists(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	u, err := r.FindByName(ctx, "default")
	if err != nil {
		t.Fatalf("FindByName: %v", err)
	}
	if u == nil {
		t.Fatal("expected default user (created by migration), got nil")
	}
	if u.Name != "default" {
		t.Errorf("Name: got %q, want %q", u.Name, "default")
	}
}

func TestUserRepository_FindByName_NotExists(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	u, err := r.FindByName(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("FindByName: %v", err)
	}
	if u != nil {
		t.Errorf("expected nil, got %+v", u)
	}
}

func TestUserRepository_Create_InsertsNewUser(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	u, err := r.Create(ctx, "alice")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if u.ID <= 0 {
		t.Errorf("ID: got %d, want > 0", u.ID)
	}
	if u.Name != "alice" {
		t.Errorf("Name: got %q, want %q", u.Name, "alice")
	}

	found, err := r.FindByName(ctx, "alice")
	if err != nil {
		t.Fatalf("FindByName after Create: %v", err)
	}
	if found == nil || found.ID != u.ID {
		t.Errorf("FindByName after Create: got %+v, want ID %d", found, u.ID)
	}
}

func TestUserRepository_Create_DuplicateNameFails(t *testing.T) {
	ctx := context.Background()
	r := newRepo(t)

	if _, err := r.Create(ctx, "alice"); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if _, err := r.Create(ctx, "alice"); err == nil {
		t.Error("expected error for duplicate name (UNIQUE制約違反), got nil")
	}
}
