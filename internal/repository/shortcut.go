package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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

// Create creates a new shortcut along with its tags. The link row and its tag
// rows are written in a single transaction so a tag-write failure never leaves
// a half-tagged link behind.
func (r *ShortcutRepository) Create(ctx context.Context, shortcut *domain.Shortcut) error {
	start := time.Now()
	r.logger.Debug("Creating shortcut: word='%s' link='%s' user='%s' tags=%v", shortcut.Word, shortcut.Link, shortcut.User, shortcut.Tags)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.logger.Error("Failed to begin transaction: %v", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Rollback is a no-op once the transaction has been committed.
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx,
		`INSERT INTO linktable (word, link, user, created_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
		shortcut.Word, shortcut.Link, shortcut.User,
	)
	if err != nil {
		r.logger.Error("Database insert failed: %v", err)
		return fmt.Errorf("failed to create shortcut: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		r.logger.Error("Failed to get last insert ID: %v", err)
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	shortcut.ID = int(id)

	// INSERT OR IGNORE leans on the unique (word_id, tag) index to drop
	// duplicates. The Postgres port becomes ON CONFLICT DO NOTHING.
	for _, tag := range shortcut.Tags {
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO tags (word_id, tag) VALUES (?, ?)`,
			shortcut.ID, tag,
		); err != nil {
			r.logger.Error("Failed to insert tag '%s' for shortcut %d: %v", tag, shortcut.ID, err)
			return fmt.Errorf("failed to insert tag: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		r.logger.Error("Failed to commit transaction: %v", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("Shortcut created successfully: id=%d (%v)", shortcut.ID, time.Since(start))
	return nil
}

// DeleteByWord removes a keyword and everything tied to it — all of its
// linktable revisions plus their dependent tag and query-log rows — in a single
// transaction. It returns the number of linktable rows removed, so the caller
// can distinguish a real delete from "nothing matched".
func (r *ShortcutRepository) DeleteByWord(ctx context.Context, word string) (int64, error) {
	start := time.Now()
	r.logger.Debug("Deleting keyword: word='%s'", word)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.logger.Error("Failed to begin transaction: %v", err)
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// tags and queries reference linktable.id without ON DELETE CASCADE, so the
	// dependents must go first or the foreign-key check (foreign_keys=on) rejects
	// the linktable delete. The Postgres port could lean on cascades instead.
	const subSelect = `SELECT id FROM linktable WHERE word = ?`
	if _, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE word_id IN (`+subSelect+`)`, word); err != nil {
		r.logger.Error("Failed to delete tags for word '%s': %v", word, err)
		return 0, fmt.Errorf("failed to delete tags: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM queries WHERE word_id IN (`+subSelect+`)`, word); err != nil {
		r.logger.Error("Failed to delete queries for word '%s': %v", word, err)
		return 0, fmt.Errorf("failed to delete queries: %w", err)
	}

	result, err := tx.ExecContext(ctx, `DELETE FROM linktable WHERE word = ?`, word)
	if err != nil {
		r.logger.Error("Failed to delete linktable rows for word '%s': %v", word, err)
		return 0, fmt.Errorf("failed to delete shortcut: %w", err)
	}
	removed, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to read rows affected: %w", err)
	}

	if err := tx.Commit(); err != nil {
		r.logger.Error("Failed to commit transaction: %v", err)
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("Deleted keyword '%s': %d row(s) (%v)", word, removed, time.Since(start))
	return removed, nil
}

// GetAllKeywords retrieves all keywords with their latest links and tags.
func (r *ShortcutRepository) GetAllKeywords(ctx context.Context) ([]domain.KeywordInfo, error) {
	start := time.Now()
	r.logger.Debug("Getting all keywords")

	// MAX(id) with bare columns relies on SQLite returning the bare-column
	// values from the same row as the max. Portable rewrite for Postgres:
	// DISTINCT ON (word) ... ORDER BY word, id DESC.
	query := `
		SELECT word, link, created_at, MAX(id) as max_id
		FROM linktable
		GROUP BY word
		ORDER BY max_id DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		r.logger.Error("Database query failed: %v", err)
		return nil, fmt.Errorf("failed to get all keywords: %w", err)
	}
	defer rows.Close()

	keywords, ids, err := scanKeywords(rows)
	if err != nil {
		r.logger.Error("Failed to scan keywords: %v", err)
		return nil, err
	}

	if err := r.attachTags(ctx, keywords, ids); err != nil {
		return nil, err
	}

	r.logger.Debug("All keywords retrieved successfully: %d keywords (%v)", len(keywords), time.Since(start))
	return keywords, nil
}

// Search returns the latest link per word whose word, link, or any tag matches
// the query as a case-insensitive substring, most-recent first.
//
// This is deliberately LIKE-based rather than FTS5: it needs no build tags and
// is portable to Postgres ILIKE. See ROADMAP.md "Search" decision.
func (r *ShortcutRepository) Search(ctx context.Context, query string, limit int) ([]domain.KeywordInfo, error) {
	start := time.Now()
	r.logger.Debug("Searching keywords: q='%s' limit=%d", query, limit)

	pattern := likePattern(query)
	sqlQuery := `
		WITH latest AS (
			SELECT word, link, created_at, MAX(id) AS id
			FROM linktable
			GROUP BY word
		)
		SELECT l.word, l.link, l.created_at, l.id
		FROM latest l
		LEFT JOIN tags t ON t.word_id = l.id
		WHERE lower(l.word) LIKE ? ESCAPE '\'
		   OR lower(l.link) LIKE ? ESCAPE '\'
		   OR lower(t.tag)  LIKE ? ESCAPE '\'
		GROUP BY l.word
		ORDER BY l.id DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, sqlQuery, pattern, pattern, pattern, limit)
	if err != nil {
		r.logger.Error("Search query failed: %v", err)
		return nil, fmt.Errorf("failed to search keywords: %w", err)
	}
	defer rows.Close()

	keywords, ids, err := scanKeywords(rows)
	if err != nil {
		r.logger.Error("Failed to scan search results: %v", err)
		return nil, err
	}

	if err := r.attachTags(ctx, keywords, ids); err != nil {
		return nil, err
	}

	r.logger.Debug("Search returned %d keywords (%v)", len(keywords), time.Since(start))
	return keywords, nil
}

// scanKeywords reads (word, link, created_at, id) rows into KeywordInfo values
// and returns the parallel slice of row ids used to load tags.
func scanKeywords(rows *sql.Rows) ([]domain.KeywordInfo, []int, error) {
	var (
		keywords []domain.KeywordInfo
		ids      []int
	)
	for rows.Next() {
		var (
			keyword domain.KeywordInfo
			id      int
		)
		if err := rows.Scan(&keyword.Word, &keyword.Link, &keyword.CreatedAt, &id); err != nil {
			return nil, nil, fmt.Errorf("failed to scan keyword: %w", err)
		}
		keywords = append(keywords, keyword)
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating keywords: %w", err)
	}
	return keywords, ids, nil
}

// attachTags batch-loads tags for the given link row ids and populates the
// Tags field of each keyword in place. keywords[i] corresponds to ids[i].
func (r *ShortcutRepository) attachTags(ctx context.Context, keywords []domain.KeywordInfo, ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	query := fmt.Sprintf(
		`SELECT word_id, tag FROM tags WHERE word_id IN (%s) ORDER BY tag`,
		placeholders,
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to load tags: %w", err)
	}
	defer rows.Close()

	tagsByWordID := make(map[int][]string, len(ids))
	for rows.Next() {
		var (
			wordID int
			tag    string
		)
		if err := rows.Scan(&wordID, &tag); err != nil {
			return fmt.Errorf("failed to scan tag: %w", err)
		}
		tagsByWordID[wordID] = append(tagsByWordID[wordID], tag)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating tags: %w", err)
	}

	for i := range keywords {
		keywords[i].Tags = tagsByWordID[ids[i]]
	}
	return nil
}

// likePattern lower-cases the query and escapes LIKE metacharacters so a user
// typing "50%" or "a_b" searches literally. Pair with `ESCAPE '\'` in SQL.
func likePattern(q string) string {
	escaper := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + strings.ToLower(escaper.Replace(strings.TrimSpace(q))) + "%"
}
