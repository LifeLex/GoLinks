// Package migrations holds the Goose migrations for GoLinks.
//
// Each migration is a Go function (registered via init()) that delegates
// schema work to GORM. GORM handles dialect translation, so a single set of
// migrations runs against any GORM-supported backend (SQLite, Postgres, …)
// — there are no per-engine SQL files.
//
// Importing this package registers every migration with Goose. The
// composition root does this with a blank import:
//
//	_ "golinks/internal/adapters/persistence/migrations"
//
// The migration runner (internal/platform/database.Migrate) attaches a
// *gorm.DB to the context via database.WithGorm; migrations retrieve it via
// database.GormFromContext.
//
// Conventions:
//
//   - Filenames follow Goose's NNNNN_name.go pattern. The leading digits set
//     the migration version.
//   - Each file calls goose.AddMigrationNoTxContext in init() to register
//     itself. We use NoTx because GORM's Migrator API doesn't compose well
//     with a *sql.Tx.
//   - For schema-only changes (CREATE TABLE, ADD COLUMN, …) prefer
//     gormDB.AutoMigrate(...) and gormDB.Migrator() — they're dialect-aware.
//   - For dialect-specific operations (FTS5 virtual tables, tsvector
//     indexes), the migration may dispatch on the dialect via raw db.Exec
//     calls. Keep the file in this folder; just put the dispatch inside the
//     Go function rather than splitting into per-engine files.
package migrations
