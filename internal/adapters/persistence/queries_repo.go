package persistence

import (
	"context"
	"fmt"
	"time"

	"golinks/internal/core/links"
	"golinks/internal/platform/logger"

	"gorm.io/gorm"
)

// QueriesRepo implements links.QueryRepository against a *gorm.DB.
type QueriesRepo struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewQueriesRepo wires the GORM-backed query analytics repository.
func NewQueriesRepo(db *gorm.DB, log *logger.Logger) *QueriesRepo {
	log.Info("Queries repository initialized")
	return &QueriesRepo{db: db, logger: log}
}

// Create logs a single query hit.
func (r *QueriesRepo) Create(ctx context.Context, wordID int) error {
	start := time.Now()
	r.logger.Debug("Creating query log for word ID: %d", wordID)

	row := queryRow{WordID: uint(wordID), CreatedAt: time.Now()}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		r.logger.Error("Database insert failed: %v (%v)", err, time.Since(start))
		return fmt.Errorf("failed to create query log: %w", err)
	}

	r.logger.Debug("Query log created successfully (%v)", time.Since(start))
	return nil
}

// popularProjection is the result shape for GetRecentQueries.
type popularProjection struct {
	Count int
	Word  string
	Link  string
}

// GetRecentQueries returns the top numResults keywords by hit count over the
// last timeWindowDays days. The time window is computed in Go and passed as a
// parameter so the SQL stays portable across SQLite and Postgres.
func (r *QueriesRepo) GetRecentQueries(ctx context.Context, timeWindowDays, numResults int) ([]links.PopularQuery, error) {
	start := time.Now()
	r.logger.Debug("Getting recent queries: %d days, max %d results", timeWindowDays, numResults)

	since := time.Now().AddDate(0, 0, -timeWindowDays)

	var rows []popularProjection
	err := r.db.WithContext(ctx).
		Table("queries AS q").
		Select("COUNT(q.word_id) AS count, s.word, s.link").
		Joins("JOIN linktable s ON q.word_id = s.id").
		Where("q.created_at > ?", since).
		Group("q.word_id").
		Order("count DESC").
		Limit(numResults).
		Scan(&rows).Error
	if err != nil {
		r.logger.Error("Database query failed: %v (%v)", err, time.Since(start))
		return nil, fmt.Errorf("failed to get recent queries: %w", err)
	}

	queries := make([]links.PopularQuery, 0, len(rows))
	for _, row := range rows {
		queries = append(queries, links.PopularQuery{
			Count: row.Count,
			Word:  row.Word,
			Link:  row.Link,
		})
	}

	r.logger.Debug("Recent queries retrieved successfully: %d queries (%v)", len(queries), time.Since(start))
	return queries, nil
}
