package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golinks/internal/domain"
	"golinks/internal/logger"
)

// SessionRepository handles database operations for login sessions. Only the
// SHA-256 hash of a session token is ever persisted.
type SessionRepository struct {
	db     *sql.DB
	logger *logger.Logger
}

// NewSessionRepository creates a new session repository.
func NewSessionRepository(db *sql.DB, log *logger.Logger) *SessionRepository {
	log.Info("Session repository initialized")
	return &SessionRepository{db: db, logger: log}
}

// Create persists a session.
func (r *SessionRepository) Create(ctx context.Context, s *domain.Session) error {
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at, created_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
		s.TokenHash, s.UserID, s.ExpiresAt,
	); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

// GetByTokenHash looks up a session by its token hash. It returns (nil, nil)
// when no session exists.
func (r *SessionRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error) {
	var s domain.Session
	err := r.db.QueryRowContext(ctx,
		`SELECT token_hash, user_id, expires_at, created_at FROM sessions WHERE token_hash = ?`,
		tokenHash,
	).Scan(&s.TokenHash, &s.UserID, &s.ExpiresAt, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &s, nil
}

// Delete removes a single session by token hash. It is idempotent.
func (r *SessionRepository) Delete(ctx context.Context, tokenHash string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// DeleteByUserID removes all sessions for a user (e.g. on account deletion).
func (r *SessionRepository) DeleteByUserID(ctx context.Context, userID int) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("failed to delete sessions for user: %w", err)
	}
	return nil
}

// DeleteExpired removes all sessions whose expiry is at or before now. Returns
// the number of rows removed.
func (r *SessionRepository) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, now)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired sessions: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to read rows affected: %w", err)
	}
	return n, nil
}
