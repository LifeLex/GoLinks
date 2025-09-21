package repository

import (
	"context"
	"database/sql"
	"fmt"

	"golinks/internal/domain"
)

// ShortcutRepository handles database operations for shortcuts
type ShortcutRepository struct {
	db *sql.DB
}

// NewShortcutRepository creates a new shortcut repository
func NewShortcutRepository(db *sql.DB) *ShortcutRepository {
	return &ShortcutRepository{db: db}
}

// GetByWord retrieves the most recent shortcut by word
func (r *ShortcutRepository) GetByWord(ctx context.Context, word string) (*domain.Shortcut, error) {

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

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get shortcut by word: %w", err)
	}

	return &shortcut, nil
}

// Create creates a new shortcut
func (r *ShortcutRepository) Create(ctx context.Context, shortcut *domain.Shortcut) error {

	query := `
		INSERT INTO linktable (word, link, user, created_at) 
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	`

	result, err := r.db.ExecContext(ctx, query, shortcut.Word, shortcut.Link, shortcut.User)
	if err != nil {
		return fmt.Errorf("failed to create shortcut: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	shortcut.ID = int(id)
	return nil
}

// GetAllKeywords retrieves all keywords with their latest links
func (r *ShortcutRepository) GetAllKeywords(ctx context.Context) ([]domain.KeywordInfo, error) {

	query := `
		SELECT word, link, created_at, MAX(id) as max_id
		FROM linktable 
		GROUP BY word 
		ORDER BY max_id DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all keywords: %w", err)
	}
	defer rows.Close()

	var keywords []domain.KeywordInfo
	for rows.Next() {
		var keyword domain.KeywordInfo
		var maxID int
		err := rows.Scan(&keyword.Word, &keyword.Link, &keyword.CreatedAt, &maxID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan keyword: %w", err)
		}
		keywords = append(keywords, keyword)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating keywords: %w", err)
	}

	return keywords, nil
}
