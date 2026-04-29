// Package database owns the connection factory and migration runner. It is the
// only place in the codebase that knows about driver-specific concerns —
// every other package consumes a *gorm.DB. Phase 4 will extend OpenGorm to
// dispatch on a driver argument; today only SQLite is wired.
package database

import (
	"fmt"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Driver is the database backend selector. Today only DriverSQLite is
// implemented; DriverPostgres is reserved for Phase 4.
type Driver string

const (
	DriverSQLite   Driver = "sqlite"
	DriverPostgres Driver = "postgres"
)

// OpenGorm opens a GORM connection for the requested driver. The url format
// is driver-specific:
//
//   - sqlite:   a filesystem path, ":memory:", or a full DSN. If pragmas aren't
//     present in the URL, foreign keys are enabled by default.
//   - postgres: a libpq-style DSN, e.g.
//     "postgres://user:pass@host:5432/dbname?sslmode=disable".
func OpenGorm(driver Driver, url string) (*gorm.DB, error) {
	switch driver {
	case DriverSQLite:
		return openSQLite(url)
	case DriverPostgres:
		return openPostgres(url)
	default:
		return nil, fmt.Errorf("unsupported database driver %q", driver)
	}
}

// openSQLite opens a SQLite connection through the pure-Go glebarez driver.
// CGO is not required.
func openSQLite(url string) (*gorm.DB, error) {
	url = ensureSQLitePragmas(url)
	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}
	db, err := gorm.Open(sqlite.Open(url), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}
	return db, nil
}

// ensureSQLitePragmas appends the foreign-keys pragma if the caller hasn't
// already passed one in the connection string. modernc.org/sqlite (used by
// glebarez) takes pragmas via the `_pragma` query parameter.
func ensureSQLitePragmas(url string) string {
	if strings.Contains(url, "_pragma=") {
		return url
	}
	separator := "?"
	if strings.Contains(url, "?") {
		separator = "&"
	}
	return url + separator + "_pragma=foreign_keys(1)"
}

// openPostgres opens a Postgres connection through gorm.io/driver/postgres
// (which uses pgx under the hood — pure Go, no CGO).
func openPostgres(url string) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}
	db, err := gorm.Open(postgres.Open(url), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres database: %w", err)
	}
	return db, nil
}
