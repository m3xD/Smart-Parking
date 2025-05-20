package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"smart_parking/internal/domain"
	"smart_parking/internal/repository"
	"time"

	"github.com/lib/pq" // Import driver PostgreSQL
)

type pgUserRepository struct {
	db *sql.DB
}

func NewPgUserRepository(db *sql.DB) repository.UserRepository {
	return &pgUserRepository{db: db}
}

func (r *pgUserRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	query := `INSERT INTO users (username, password_hash, role, created_at, updated_at) 
	           VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) 
	           RETURNING id, created_at, updated_at`
	// user.Password ở đây là password_hash
	err := r.db.QueryRowContext(ctx, query, user.Username, user.Password, user.Role).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code.Name() == "unique_violation" && pqErr.Constraint == "users_username_key" {
				return nil, fmt.Errorf("%w: tên người dùng '%s' đã tồn tại", repository.ErrDuplicateEntry, user.Username)
			}
		}
		return nil, fmt.Errorf("UserRepository.Create: %w", err)
	}
	user.CreatedAt = user.CreatedAt.In(time.UTC)
	user.UpdatedAt = user.UpdatedAt.In(time.UTC)
	return user, nil
}

func (r *pgUserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	user := &domain.User{}
	query := `SELECT id, username, password_hash, role, created_at, updated_at FROM users WHERE username = $1`
	err := r.db.QueryRowContext(ctx, query, username).Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("UserRepository.FindByUsername: %w", err)
	}
	user.CreatedAt = user.CreatedAt.In(time.UTC)
	user.UpdatedAt = user.UpdatedAt.In(time.UTC)
	return user, nil
}

func (r *pgUserRepository) FindByID(ctx context.Context, id int) (*domain.User, error) {
	user := &domain.User{}
	query := `SELECT id, username, password_hash, role, created_at, updated_at FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("UserRepository.FindByID: %w", err)
	}
	user.CreatedAt = user.CreatedAt.In(time.UTC)
	user.UpdatedAt = user.UpdatedAt.In(time.UTC)
	return user, nil
}
