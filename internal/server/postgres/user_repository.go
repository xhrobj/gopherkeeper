package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const usersLoginUniqueConstraint = "users_login_unique"

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
		postgresError.Code == pgerrcode.UniqueViolation &&
		postgresError.ConstraintName == usersLoginUniqueConstraint {

		return model.User{}, fmt.Errorf("create user: %w", model.ErrLoginAlreadyExists)
	}

	return model.User{}, fmt.Errorf("create user: %w", err)
}

// FindByLogin возвращает пользователя и хэш пароля по каноническому логину.
func (r *UserRepository) FindByLogin(ctx context.Context, login string) (model.User, []byte, error) {
	var user model.User
	var passwordHash []byte

	err := r.pool.QueryRow(
		ctx,
		`SELECT id, login, password_hash, created_at
		 FROM gopherkeeper.users
		 WHERE login = $1`,
		login,
	).Scan(&user.ID, &user.Login, &passwordHash, &user.CreatedAt)
	if err == nil {
		return user, passwordHash, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, nil, fmt.Errorf("find user by login: %w", model.ErrUserNotFound)
	}

	return model.User{}, nil, fmt.Errorf("find user by login: %w", err)
}

// FindByID возвращает публичные данные пользователя по внутреннему идентификатору.
func (r *UserRepository) FindByID(ctx context.Context, id int64) (model.User, error) {
	var user model.User

	err := r.pool.QueryRow(
		ctx,
		`SELECT id, login, created_at
		 FROM gopherkeeper.users
		 WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Login, &user.CreatedAt)
	if err == nil {
		return user, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, fmt.Errorf("find user by id: %w", model.ErrUserNotFound)
	}

	return model.User{}, fmt.Errorf("find user by id: %w", err)
}
