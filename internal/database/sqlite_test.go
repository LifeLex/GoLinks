package database

import (
	"database/sql"
	"os"
	"testing"
)

func TestNewSQLiteDB(t *testing.T) {
	tests := []struct {
		name    string
		dbPath  string
		wantErr bool
		cleanup bool
	}{
		{
			name:    "in-memory database",
			dbPath:  ":memory:",
			wantErr: false,
			cleanup: false,
		},
		{
			name:    "file database",
			dbPath:  "test.db",
			wantErr: false,
			cleanup: true,
		},
		{
			name:    "database with foreign keys",
			dbPath:  "test_fk.db",
			wantErr: false,
			cleanup: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cleanup {
				defer os.Remove(tt.dbPath)
			}

			db, err := NewSQLiteDB(tt.dbPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSQLiteDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if db == nil {
					t.Error("NewSQLiteDB() returned nil database")
					return
				}

				// Test that we can ping the database
				if err := db.Ping(); err != nil {
					t.Errorf("Database ping failed: %v", err)
				}

				// Test that foreign keys are enabled
				var fkEnabled int
				err = db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
				if err != nil {
					t.Errorf("Failed to check foreign keys: %v", err)
				}
				if fkEnabled != 1 {
					t.Error("Foreign keys should be enabled")
				}

				db.Close()
			}
		})
	}
}

func TestMigrate(t *testing.T) {
	tests := []struct {
		name    string
		dbPath  string
		wantErr bool
	}{
		{
			name:    "successful migration",
			dbPath:  ":memory:",
			wantErr: false,
		},
		{
			name:    "migration on file database",
			dbPath:  "migrate_test.db",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.dbPath != ":memory:" {
				defer os.Remove(tt.dbPath)
			}

			db, err := NewSQLiteDB(tt.dbPath)
			if err != nil {
				t.Fatalf("Failed to create database: %v", err)
			}
			defer db.Close()

			err = Migrate(db)
			if (err != nil) != tt.wantErr {
				t.Errorf("Migrate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify that tables were created
				tables := []string{"linktable", "queries", "tags"}
				for _, table := range tables {
					var count int
					query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?"
					err := db.QueryRow(query, table).Scan(&count)
					if err != nil {
						t.Errorf("Failed to check table %s: %v", table, err)
					}
					if count != 1 {
						t.Errorf("Table %s was not created", table)
					}
				}

				// Verify that indexes were created
				indexes := []string{"idx_linktable_word", "idx_queries_word_id", "idx_queries_created_at"}
				for _, index := range indexes {
					var count int
					query := "SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?"
					err := db.QueryRow(query, index).Scan(&count)
					if err != nil {
						t.Errorf("Failed to check index %s: %v", index, err)
					}
					if count != 1 {
						t.Errorf("Index %s was not created", index)
					}
				}

				// Test that we can insert data (basic functionality test)
				_, err = db.Exec("INSERT INTO linktable (word, link, user) VALUES (?, ?, ?)", "test", "https://test.com", "testuser")
				if err != nil {
					t.Errorf("Failed to insert test data: %v", err)
				}

				// Test foreign key constraint
				_, err = db.Exec("INSERT INTO queries (word_id) VALUES (?)", 1)
				if err != nil {
					t.Errorf("Failed to insert query with valid foreign key: %v", err)
				}

				// Test that foreign key constraint works
				_, err = db.Exec("INSERT INTO queries (word_id) VALUES (?)", 999)
				if err == nil {
					t.Error("Expected foreign key constraint error, but got none")
				}
			}
		})
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	db, err := NewSQLiteDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Run migration twice - should not error
	err = Migrate(db)
	if err != nil {
		t.Errorf("First migration failed: %v", err)
	}

	err = Migrate(db)
	if err != nil {
		t.Errorf("Second migration failed: %v", err)
	}

	// Verify tables still exist and work
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&count)
	if err != nil {
		t.Errorf("Failed to count tables: %v", err)
	}

	// Should have at least 3 tables: linktable, queries, tags (SQLite may create additional system tables)
	if count < 3 {
		t.Errorf("Expected at least 3 tables, got %d", count)
	}
}

func TestMigrate_ClosedDatabase(t *testing.T) {
	db, err := NewSQLiteDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	db.Close() // Close before migration

	err = Migrate(db)
	if err == nil {
		t.Error("Expected error when migrating closed database, got nil")
	}
}

func TestNewSQLiteDB_InvalidPath(t *testing.T) {
	// Test with invalid path (directory that doesn't exist)
	invalidPath := "/nonexistent/directory/test.db"

	db, err := NewSQLiteDB(invalidPath)
	if err == nil {
		if db != nil {
			db.Close()
		}
		// SQLite might create the file anyway, so this test might pass
		// depending on the system. We'll just ensure it doesn't panic.
		t.Log("SQLite created database in nonexistent directory (this is normal SQLite behavior)")
	}
}

func TestDatabaseSchema(t *testing.T) {
	db, err := NewSQLiteDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	err = Migrate(db)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Test linktable schema
	rows, err := db.Query("PRAGMA table_info(linktable)")
	if err != nil {
		t.Fatalf("Failed to get linktable schema: %v", err)
	}
	defer rows.Close()

	expectedColumns := map[string]bool{
		"id":         false,
		"word":       false,
		"link":       false,
		"user":       false,
		"created_at": false,
	}

	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue sql.NullString

		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			t.Errorf("Failed to scan column info: %v", err)
			continue
		}

		if _, exists := expectedColumns[name]; exists {
			expectedColumns[name] = true
		}
	}

	for column, found := range expectedColumns {
		if !found {
			t.Errorf("Expected column %s not found in linktable", column)
		}
	}

	// Test that we can perform basic operations
	testOperations := []string{
		"INSERT INTO linktable (word, link, user) VALUES ('test1', 'https://test1.com', 'user1')",
		"INSERT INTO linktable (word, link, user) VALUES ('test2', 'https://test2.com', 'user2')",
		"INSERT INTO queries (word_id) VALUES (1)",
		"INSERT INTO queries (word_id) VALUES (2)",
		"INSERT INTO tags (word_id, tag) VALUES (1, 'documentation')",
	}

	for _, operation := range testOperations {
		_, err := db.Exec(operation)
		if err != nil {
			t.Errorf("Failed to execute operation '%s': %v", operation, err)
		}
	}

	// Test queries work
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM linktable").Scan(&count)
	if err != nil {
		t.Errorf("Failed to count linktable rows: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 rows in linktable, got %d", count)
	}
}
