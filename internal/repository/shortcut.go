package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"golinks/internal/domain"
	"golinks/internal/logger"
)

// ShortcutRepository handles database operations for shortcuts
type ShortcutRepository struct {
	db     *sql.DB
	logger *logger.Logger
}

// NewShortcutRepository creates a new shortcut repository
func NewShortcutRepository(db *sql.DB, log *logger.Logger) *ShortcutRepository {
	log.Info("Shortcut repository initialized")
	return &ShortcutRepository{
		db:     db,
		logger: log,
	}
}

// GetByWord retrieves the most recent shortcut by word
func (r *ShortcutRepository) GetByWord(ctx context.Context, word string) (*domain.Shortcut, error) {
	start := time.Now()
	r.logger.Debug("Getting shortcut by word: %s", word)

	query := `
		SELECT id, word, link, user, created_at 
		FROM linktable 
		WHERE word = ? 
		ORDER BY id DESC 
		LIMIT 1
	`

	var shortcut domain.Shortcut
	err := r.db.QueryRowContext(ctx, query, word).Scan(
		&shortcut.ID,
		&shortcut.Word,
		&shortcut.Link,
		&shortcut.User,
		&shortcut.CreatedAt,
	)

	duration := time.Since(start)

	if err == sql.ErrNoRows {
		r.logger.Debug("No shortcut found for word '%s' (%v)", word, duration)
		return nil, nil
	}
	if err != nil {
		r.logger.Error("Database query failed for word '%s': %v (%v)", word, err, duration)
		return nil, fmt.Errorf("failed to get shortcut by word: %w", err)
	}

	r.logger.Debug("Shortcut retrieved: id=%d user='%s' (%v)", shortcut.ID, shortcut.User, duration)
	return &shortcut, nil
}

// Create creates a new shortcut
func (r *ShortcutRepository) Create(ctx context.Context, shortcut *domain.Shortcut) error {
	start := time.Now()
	r.logger.Debug("Creating shortcut: word='%s' link='%s' user='%s'", shortcut.Word, shortcut.Link, shortcut.User)

	query := `
		INSERT INTO linktable (word, link, user, created_at) 
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	`

	result, err := r.db.ExecContext(ctx, query, shortcut.Word, shortcut.Link, shortcut.User)
	duration := time.Since(start)

	if err != nil {
		r.logger.Error("Database insert failed: %v (%v)", err, duration)
		return fmt.Errorf("failed to create shortcut: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		r.logger.Error("Failed to get last insert ID: %v (%v)", err, duration)
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	shortcut.ID = int(id)
	r.logger.Info("Shortcut created successfully: id=%d (%v)", shortcut.ID, duration)
	return nil
}

// GetAllKeywords retrieves all keywords with their latest links
func (r *ShortcutRepository) GetAllKeywords(ctx context.Context) ([]domain.KeywordInfo, error) {
	start := time.Now()
	r.logger.Debug("Getting all keywords")

	query := `
		SELECT word, link, created_at, MAX(id) as max_id
		FROM linktable 
		GROUP BY word 
		ORDER BY max_id DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		duration := time.Since(start)
		r.logger.Error("Database query failed: %v (%v)", err, duration)
		return nil, fmt.Errorf("failed to get all keywords: %w", err)
	}
	defer rows.Close()

	var keywords []domain.KeywordInfo
	for rows.Next() {
		var keyword domain.KeywordInfo
		var maxID int
		err := rows.Scan(&keyword.Word, &keyword.Link, &keyword.CreatedAt, &maxID)
		if err != nil {
			duration := time.Since(start)
			r.logger.Error("Failed to scan keyword row: %v (%v)", err, duration)
			return nil, fmt.Errorf("failed to scan keyword: %w", err)
		}
		keywords = append(keywords, keyword)
	}

	if err := rows.Err(); err != nil {
		duration := time.Since(start)
		r.logger.Error("Error iterating keyword rows: %v (%v)", err, duration)
		return nil, fmt.Errorf("error iterating keywords: %w", err)
	}

	duration := time.Since(start)
	r.logger.Debug("All keywords retrieved successfully: %d keywords (%v)", len(keywords), duration)
	return keywords, nil
}
