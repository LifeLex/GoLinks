// External test package — keeps the migration registration import (which
// transitively imports this package via internal/adapters/persistence/migrations)
// from creating a cycle. Tests still use only the public API.
package database_test

import (
	"testing"

	// Side-effect import: registers Goose migrations via init() so the
	// tests in this file see the same migration set production does.
	_ "golinks/internal/adapters/persistence/migrations"
	"golinks/internal/platform/database"
)

func TestOpenGorm_SQLite_Memory(t *testing.T) {
	db, err := database.OpenGorm(database.DriverSQLite, ":memory:")
	if err != nil {
		t.Fatalf("OpenGorm: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("DB(): %v", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		t.Errorf("ping failed: %v", err)
	}

	var fk int
	if err := sqlDB.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Errorf("foreign keys check: %v", err)
	}
	if fk != 1 {
		t.Error("foreign keys should be enabled by default")
	}
}

func TestOpenGorm_UnknownDriver(t *testing.T) {
	_, err := database.OpenGorm("mysql", "mysql://localhost")
	if err == nil {
		t.Error("expected error for unknown driver")
	}
}

// TestMigrate_CreatesSQLiteSchema verifies Goose runs the embedded SQLite
// migrations end-to-end: schema is materialised and the goose_db_version
// bookkeeping table exists alongside the app tables.
func TestMigrate_CreatesSQLiteSchema(t *testing.T) {
	db, err := database.OpenGorm(database.DriverSQLite, ":memory:")
	if err != nil {
		t.Fatalf("OpenGorm: %v", err)
	}

	if err := database.Migrate(db, database.DriverSQLite); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	for _, table := range []string{"linktable", "queries", "tags", "goose_db_version"} {
		if !db.Migrator().HasTable(table) {
			t.Errorf("expected table %q to exist after Migrate", table)
		}
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	db, err := database.OpenGorm(database.DriverSQLite, ":memory:")
	if err != nil {
		t.Fatalf("OpenGorm: %v", err)
	}

	if err := database.Migrate(db, database.DriverSQLite); err != nil {
		t.Errorf("first migration: %v", err)
	}
	if err := database.Migrate(db, database.DriverSQLite); err != nil {
		t.Errorf("second migration: %v", err)
	}
}

func TestMigrate_UnknownDriver(t *testing.T) {
	db, err := database.OpenGorm(database.DriverSQLite, ":memory:")
	if err != nil {
		t.Fatalf("OpenGorm: %v", err)
	}
	if err := database.Migrate(db, "mysql"); err == nil {
		t.Error("expected error for unsupported driver")
	}
}
