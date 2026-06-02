package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"golinks/internal/domain"
	"golinks/internal/logger"

	"github.com/mattn/go-sqlite3"
)

// UserRepository handles database operations for users.
type UserRepository struct {
	db     *sql.DB
	logger *logger.Logger
}

// NewUserRepository creates a new user repository.
func NewUserRepository(db *sql.DB, log *logger.Logger) *UserRepository {
	log.Info("User repository initialized")
	return &UserRepository{db: db, logger: log}
}

// CountUsers returns the total number of users. Used to decide bootstrap.
func (r *UserRepository) CountUsers(ctx context.Context) (int, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return n, nil
}

// CountAdmins returns the number of users holding the admin role. Used to guard
// against removing the last admin.
func (r *UserRepository) CountAdmins(ctx context.Context) (int, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role = ?`, domain.RoleAdmin).Scan(&n); err != nil {
		return 0, fmt.Errorf("failed to count admins: %w", err)
	}
	return n, nil
}

// Create inserts a new user, mapping a unique-email violation to
// domain.ErrEmailTaken. The user's ID is set on success.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, role, created_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
		user.Email, user.PasswordHash, user.Role,
	)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return domain.ErrEmailTaken
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	user.ID = int(id)
	return nil
}

// GetByEmail looks up a user by email (case-insensitive). It returns
// (nil, nil) when no user exists, mirroring ShortcutRepository.GetByWord.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.scanUser(r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, role, created_at FROM users WHERE lower(email) = lower(?)`,
		email,
	))
}

// GetByID looks up a user by id. It returns (nil, nil) when none exists.
func (r *UserRepository) GetByID(ctx context.Context, id int) (*domain.User, error) {
	return r.scanUser(r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, role, created_at FROM users WHERE id = ?`,
		id,
	))
}

func (r *UserRepository) scanUser(row *sql.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}
	return &u, nil
}

// List returns all users ordered by creation time, oldest first.
func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, email, password_hash, role, created_at FROM users ORDER BY created_at ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}
	return users, nil
}

// Delete removes a user by id. Associated sessions are removed by the
// ON DELETE CASCADE foreign key.
func (r *UserRepository) Delete(ctx context.Context, id int) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}
