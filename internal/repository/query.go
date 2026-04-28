package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"golinks/internal/domain"
	"golinks/internal/logger"
)

// QueryRepository handles database operations for queries
type QueryRepository struct {
	db     *sql.DB
	logger *logger.Logger
}

// NewQueryRepository creates a new query repository
func NewQueryRepository(db *sql.DB, log *logger.Logger) *QueryRepository {
	log.Info("Query repository initialized")
	return &QueryRepository{
		db:     db,
		logger: log,
	}
}

// Create creates a new query log entry
func (r *QueryRepository) Create(ctx context.Context, wordID int) error {
	start := time.Now()
	r.logger.Debug("Creating query log for word ID: %d", wordID)

	query := `INSERT INTO queries (word_id, created_at) VALUES (?, CURRENT_TIMESTAMP)`

	_, err := r.db.ExecContext(ctx, query, wordID)
	duration := time.Since(start)

	if err != nil {
		r.logger.Error("Database insert failed: %v (%v)", err, duration)
		return fmt.Errorf("failed to create query log: %w", err)
	}

	r.logger.Debug("Query log created successfully (%v)", duration)
	return nil
}

// GetRecentQueries retrieves popular queries from the last N days
func (r *QueryRepository) GetRecentQueries(
	ctx context.Context, timeWindowDays, numResults int,
) ([]domain.PopularQuery, error) {
	start := time.Now()
	r.logger.Debug("Getting recent queries: %d days, max %d results", timeWindowDays, numResults)

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
		duration := time.Since(start)
		r.logger.Error("Database query failed: %v (%v)", err, duration)
		return nil, fmt.Errorf("failed to get recent queries: %w", err)
	}
	defer rows.Close()

	var queries []domain.PopularQuery
	for rows.Next() {
		var pq domain.PopularQuery
		err := rows.Scan(&pq.Count, &pq.Word, &pq.Link)
		if err != nil {
			duration := time.Since(start)
			r.logger.Error("Failed to scan popular query row: %v (%v)", err, duration)
			return nil, fmt.Errorf("failed to scan popular query: %w", err)
		}
		queries = append(queries, pq)
	}

	if err := rows.Err(); err != nil {
		duration := time.Since(start)
		r.logger.Error("Error iterating recent query rows: %v (%v)", err, duration)
		return nil, fmt.Errorf("error iterating recent queries: %w", err)
	}

	duration := time.Since(start)
	r.logger.Debug("Recent queries retrieved successfully: %d queries (%v)", len(queries), duration)
	return queries, nil
}
