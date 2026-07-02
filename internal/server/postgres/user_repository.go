package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const (
	uniqueViolationCode        = "23505"
	usersLoginUniqueConstraint = "users_login_unique"
)

// UserRepository является PostgreSQL-адаптером репозитория пользователей.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository создаёт PostgreSQL-адаптер репозитория пользователей.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create сохраняет пользователя с уже подготовленным хэшем пароля.
func (r *UserRepository) Create(ctx context.Context, login string, passwordHash []byte) (model.User, error) {
	var user model.User

	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO gopherkeeper.users (login, password_hash)
		 VALUES ($1, $2)
		 RETURNING id, login, created_at`,
		login,
		passwordHash,
	).Scan(&user.ID, &user.Login, &user.CreatedAt)
	if err == nil {
		return user, nil
	}

	var postgresError *pgconn.PgError

	if errors.As(err, &postgresError) &&
		postgresError.Code == uniqueViolationCode &&
		postgresError.ConstraintName == usersLoginUniqueConstraint {

		return model.User{}, fmt.Errorf("create user: %w", model.ErrLoginAlreadyExists)
	}

	return model.User{}, fmt.Errorf("create user: %w", err)
}
