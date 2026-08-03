package user

import (
	"context"
	"database/sql"

	userrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/user"
	"github.com/qli8racn/rss-feeder/internal/domain"
	"github.com/samber/do/v2"
)

var _ userrepo.Repository = (*repository)(nil)

type repository struct {
	db *sql.DB
}

func NewRepository(i do.Injector) (userrepo.Repository, error) {
	return &repository{db: do.MustInvoke[*sql.DB](i)}, nil
}

func (r *repository) FindByName(ctx context.Context, name string) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, name, created_at FROM users WHERE name = ?`, name)

	var u domain.User
	var createdAt sql.NullTime
	if err := row.Scan(&u.ID, &u.Name, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if createdAt.Valid {
		u.CreatedAt = createdAt.Time
	}
	return &u, nil
}

func (r *repository) Create(ctx context.Context, name string) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx, `INSERT INTO users (name) VALUES (?) RETURNING id, created_at`, name)
	var u domain.User
	u.Name = name
	var createdAt sql.NullTime
	if err := row.Scan(&u.ID, &createdAt); err != nil {
		return nil, err
	}
	if createdAt.Valid {
		u.CreatedAt = createdAt.Time
	}
	return &u, nil
}
