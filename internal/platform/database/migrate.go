package database

import (
	"context"
	"fmt"

	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
)

// gormCtxKey is the unexported context-value key used to pass a *gorm.DB into
// Goose Go-based migrations. Migrations retrieve the GORM connection via
// GormFromContext.
type gormCtxKey struct{}

// WithGorm attaches db to ctx so Goose Go-based migrations can recover it.
// The migration runner (Migrate) calls this; migration code calls
// GormFromContext to retrieve.
func WithGorm(ctx context.Context, db *gorm.DB) context.Context {
	return context.WithValue(ctx, gormCtxKey{}, db)
}

// GormFromContext returns the *gorm.DB attached by WithGorm. Returns an error
// if none is present — that's a programmer error in the migration runner,
// since a migration cannot do its job without a connection.
func GormFromContext(ctx context.Context) (*gorm.DB, error) {
	db, ok := ctx.Value(gormCtxKey{}).(*gorm.DB)
	if !ok || db == nil {
		return nil, fmt.Errorf("database: no *gorm.DB in context (call WithGorm before goose.UpContext)")
	}
	return db, nil
}

// Migrate runs every pending migration registered with Goose. Migrations live
// at internal/adapters/persistence/migrations/ as Go functions that delegate
// schema work to GORM, so a single migration set runs against any
// GORM-supported dialect — no per-engine SQL files.
//
// Callers must blank-import the migrations package somewhere in the binary
// (typically cmd/server/main.go) so init() registers them with Goose.
func Migrate(db *gorm.DB, driver Driver) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to obtain underlying *sql.DB: %w", err)
	}

	dialect, err := gooseDialect(driver)
	if err != nil {
		return err
	}
	if err := goose.SetDialect(dialect); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	// "." is a valid filesystem path; Goose will find no .sql files there
	// (we use Go migrations exclusively) and run only the registered set.
	ctx := WithGorm(context.Background(), db)
	if err := goose.UpContext(ctx, sqlDB, "."); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

// gooseDialect translates our Driver values to the dialect strings Goose
// expects (Goose calls SQLite "sqlite3").
func gooseDialect(driver Driver) (string, error) {
	switch driver {
	case DriverSQLite:
		return "sqlite3", nil
	case DriverPostgres:
		return "postgres", nil
	default:
		return "", fmt.Errorf("unsupported driver for migrations: %q", driver)
	}
}
