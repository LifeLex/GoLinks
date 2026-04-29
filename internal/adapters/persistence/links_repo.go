// Package persistence implements the outbound persistence ports defined in
// internal/core. Persistence is GORM-based: a single implementation works
// against any dialect GORM supports (SQLite via glebarez today, Postgres in
// Phase 4). Dialect selection happens once at startup in cmd/server/main.go.
package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golinks/internal/core/links"
	"golinks/internal/platform/logger"

	"gorm.io/gorm"
)

// LinksRepo implements links.Repository against a *gorm.DB.
type LinksRepo struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewLinksRepo wires the GORM-backed repository.
func NewLinksRepo(db *gorm.DB, log *logger.Logger) *LinksRepo {
	log.Info("Links repository initialized")
	return &LinksRepo{db: db, logger: log}
}

// GetByWord returns the most recent shortcut for a keyword (highest id),
// or (nil, nil) if none exists.
func (r *LinksRepo) GetByWord(ctx context.Context, word string) (*links.Shortcut, error) {
	start := time.Now()
	r.logger.Debug("Getting shortcut by word: %s", word)

	var row shortcutRow
	err := r.db.WithContext(ctx).
		Where("word = ?", word).
		Order("id DESC").
		Take(&row).Error
	duration := time.Since(start)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		r.logger.Debug("No shortcut found for word '%s' (%v)", word, duration)
		return nil, nil
	}
	if err != nil {
		r.logger.Error("Database query failed for word '%s': %v (%v)", word, err, duration)
		return nil, fmt.Errorf("failed to get shortcut by word: %w", err)
	}

	r.logger.Debug("Shortcut retrieved: id=%d user='%s' (%v)", row.ID, row.User, duration)
	return toDomainShortcut(row), nil
}

// Create inserts a new shortcut row. Mutates s.ID with the generated row ID.
func (r *LinksRepo) Create(ctx context.Context, s *links.Shortcut) error {
	start := time.Now()
	r.logger.Debug("Creating shortcut: word='%s' link='%s' user='%s'", s.Word, s.Link, s.User)

	row := fromDomainShortcut(s)
	// Force CreatedAt to be set server-side rather than relying on the
	// caller's clock, to match the legacy SQL behaviour.
	row.CreatedAt = time.Now()
	row.ID = 0 // ensure GORM treats this as an insert, not an update

	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		r.logger.Error("Database insert failed: %v (%v)", err, time.Since(start))
		return fmt.Errorf("failed to create shortcut: %w", err)
	}

	s.ID = int(row.ID)
	s.CreatedAt = row.CreatedAt
	r.logger.Info("Shortcut created successfully: id=%d (%v)", s.ID, time.Since(start))
	return nil
}

// keywordProjection is the result shape for GetAllKeywords. GORM scans into it
// directly via Select(...).Scan().
type keywordProjection struct {
	Word      string
	Link      string
	CreatedAt time.Time
	MaxID     uint
}

// GetAllKeywords returns one row per keyword, taking the latest version
// (max id) of each, ordered by recency.
func (r *LinksRepo) GetAllKeywords(ctx context.Context) ([]links.KeywordInfo, error) {
	start := time.Now()
	r.logger.Debug("Getting all keywords")

	var rows []keywordProjection
	err := r.db.WithContext(ctx).
		Table("linktable").
		Select("word, link, created_at, MAX(id) AS max_id").
		Group("word").
		Order("max_id DESC").
		Scan(&rows).Error
	if err != nil {
		r.logger.Error("Database query failed: %v (%v)", err, time.Since(start))
		return nil, fmt.Errorf("failed to get all keywords: %w", err)
	}

	keywords := make([]links.KeywordInfo, 0, len(rows))
	for _, row := range rows {
		keywords = append(keywords, links.KeywordInfo{
			Word:      row.Word,
			Link:      row.Link,
			CreatedAt: row.CreatedAt,
		})
	}

	r.logger.Debug("All keywords retrieved successfully: %d keywords (%v)", len(keywords), time.Since(start))
	return keywords, nil
}
