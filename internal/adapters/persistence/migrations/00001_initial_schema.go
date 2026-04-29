package migrations

import (
	"context"
	"database/sql"

	"golinks/internal/adapters/persistence"
	"golinks/internal/platform/database"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(upInitialSchema, downInitialSchema)
}

// upInitialSchema creates the initial schema (linktable, queries, tags) by
// asking GORM to materialise every registered model. The dialect-specific
// SQL (AUTOINCREMENT vs BIGSERIAL, DATETIME vs TIMESTAMP, etc.) is handled
// internally by GORM's dialector.
func upInitialSchema(ctx context.Context, _ *sql.DB) error {
	db, err := database.GormFromContext(ctx)
	if err != nil {
		return err
	}
	return db.WithContext(ctx).AutoMigrate(persistence.Models()...)
}

// downInitialSchema drops the schema. Tables are dropped in reverse-dependency
// order so foreign keys don't block.
func downInitialSchema(ctx context.Context, _ *sql.DB) error {
	db, err := database.GormFromContext(ctx)
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Migrator().DropTable(persistence.ModelsReverse()...)
}
