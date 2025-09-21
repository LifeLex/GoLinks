package repository

import (
	"context"
	"database/sql"
	"fmt"

	"golinks/internal/domain"
)

// QueryRepository handles database operations for queries
type QueryRepository struct {
	db *sql.DB
}

// NewQueryRepository creates a new query repository
func NewQueryRepository(db *sql.DB) *QueryRepository {
	return &QueryRepository{db: db}
}

// Create creates a new query log entry
func (r *QueryRepository) Create(ctx context.Context, wordID int) error {
	query := `INSERT INTO queries (word_id, created_at) VALUES (?, CURRENT_TIMESTAMP)`

	_, err := r.db.ExecContext(ctx, query, wordID)
	if err != nil {
		return fmt.Errorf("failed to create query log: %w", err)
	}

	return nil
}

// GetRecentQueries retrieves popular queries from the last N days
func (r *QueryRepository) GetRecentQueries(
	ctx context.Context, timeWindowDays, numResults int,
) ([]domain.PopularQuery, error) {

	query := `
		SELECT COUNT(q.word_id) as count, s.word, s.link
		FROM queries q
		JOIN linktable s ON q.word_id = s.id
		WHERE q.created_at > datetime('now', '-' || ? || ' days')
		GROUP BY q.word_id
		ORDER BY count DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, timeWindowDays, numResults)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent queries: %w", err)
	}
	defer rows.Close()

	var queries []domain.PopularQuery
	for rows.Next() {
		var pq domain.PopularQuery
		err := rows.Scan(&pq.Count, &pq.Word, &pq.Link)
		if err != nil {
			return nil, fmt.Errorf("failed to scan popular query: %w", err)
		}
		queries = append(queries, pq)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recent queries: %w", err)
	}

	return queries, nil
}
