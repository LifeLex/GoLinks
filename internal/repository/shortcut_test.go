package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"golinks/internal/domain"
	"golinks/internal/logger"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create tables
	migrations := []string{
		`CREATE TABLE linktable (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			word TEXT NOT NULL,
			link TEXT NOT NULL,
			user TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE queries (
			query_id INTEGER PRIMARY KEY AUTOINCREMENT,
			word_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (word_id) REFERENCES linktable(id)
		)`,
		`CREATE INDEX idx_linktable_word ON linktable(word)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			t.Fatalf("Failed to run migration: %v", err)
		}
	}

	return db
}

func TestShortcutRepository_GetByWord(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mockLogger := logger.New(logger.Config{Level: "debug", Format: "text"})
	repo := NewShortcutRepository(db, mockLogger)

	// Insert test data
	testShortcut := &domain.Shortcut{
		Word: "docs",
		Link: "https://docs.example.com",
		User: "testuser",
	}

	err := repo.Create(context.Background(), testShortcut)
	if err != nil {
		t.Fatalf("Failed to create test shortcut: %v", err)
	}

	tests := []struct {
		name    string
		word    string
		want    *domain.Shortcut
		wantErr bool
	}{
		{
			name: "existing word",
			word: "docs",
			want: &domain.Shortcut{
				ID:   1,
				Word: "docs",
				Link: "https://docs.example.com",
				User: "testuser",
			},
			wantErr: false,
		},
		{
			name:    "non-existing word",
			word:    "nonexistent",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "empty word",
			word:    "",
			want:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetByWord(context.Background(), tt.word)

			if (err != nil) != tt.wantErr {
				t.Errorf("ShortcutRepository.GetByWord() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want == nil && got != nil {
				t.Errorf("ShortcutRepository.GetByWord() = %v, want nil", got)
				return
			}

			if tt.want != nil && got == nil {
				t.Errorf("ShortcutRepository.GetByWord() = nil, want %v", tt.want)
				return
			}

			if tt.want != nil && got != nil {
				if got.Word != tt.want.Word || got.Link != tt.want.Link || got.User != tt.want.User {
					t.Errorf("ShortcutRepository.GetByWord() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestShortcutRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mockLogger := logger.New(logger.Config{Level: "debug", Format: "text"})
	repo := NewShortcutRepository(db, mockLogger)

	tests := []struct {
		name     string
		shortcut *domain.Shortcut
		wantErr  bool
	}{
		{
			name: "valid shortcut",
			shortcut: &domain.Shortcut{
				Word: "github",
				Link: "https://github.com",
				User: "testuser",
			},
			wantErr: false,
		},
		{
			name: "duplicate word (should succeed - allows multiple versions)",
			shortcut: &domain.Shortcut{
				Word: "github",
				Link: "https://github.com/explore",
				User: "testuser2",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalID := tt.shortcut.ID
			err := repo.Create(context.Background(), tt.shortcut)

			if (err != nil) != tt.wantErr {
				t.Errorf("ShortcutRepository.Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Check that ID was set
				if tt.shortcut.ID == originalID {
					t.Error("ShortcutRepository.Create() did not set ID")
				}

				// Verify the shortcut was actually created
				retrieved, err := repo.GetByWord(context.Background(), tt.shortcut.Word)
				if err != nil {
					t.Errorf("Failed to retrieve created shortcut: %v", err)
					return
				}

				if retrieved == nil {
					t.Error("Created shortcut not found")
					return
				}

				// Should get the most recent one (highest ID)
				if retrieved.Link != tt.shortcut.Link {
					t.Errorf("Retrieved shortcut link = %v, want %v", retrieved.Link, tt.shortcut.Link)
				}
			}
		})
	}
}

func TestShortcutRepository_GetAllKeywords(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mockLogger := logger.New(logger.Config{Level: "debug", Format: "text"})
	repo := NewShortcutRepository(db, mockLogger)

	// Insert test data
	testShortcuts := []*domain.Shortcut{
		{Word: "docs", Link: "https://docs.example.com", User: "user1"},
		{Word: "github", Link: "https://github.com", User: "user2"},
		{Word: "docs", Link: "https://docs.example.com/v2", User: "user1"}, // Updated version
	}

	for _, shortcut := range testShortcuts {
		err := repo.Create(context.Background(), shortcut)
		if err != nil {
			t.Fatalf("Failed to create test shortcut: %v", err)
		}
	}

	keywords, err := repo.GetAllKeywords(context.Background())
	if err != nil {
		t.Errorf("ShortcutRepository.GetAllKeywords() error = %v", err)
		return
	}

	// Should return 2 unique words (docs and github)
	if len(keywords) != 2 {
		t.Errorf("ShortcutRepository.GetAllKeywords() returned %d keywords, want 2", len(keywords))
	}

	// Check that we get the latest version of each word
	keywordMap := make(map[string]domain.KeywordInfo)
	for _, keyword := range keywords {
		keywordMap[keyword.Word] = keyword
	}

	if docsKeyword, exists := keywordMap["docs"]; exists {
		if docsKeyword.Link != "https://docs.example.com/v2" {
			t.Errorf("Expected latest docs link, got %s", docsKeyword.Link)
		}
	} else {
		t.Error("docs keyword not found")
	}

	if _, exists := keywordMap["github"]; !exists {
		t.Error("github keyword not found")
	}
}

func TestShortcutRepository_GetByWord_MostRecent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mockLogger := logger.New(logger.Config{Level: "debug", Format: "text"})
	repo := NewShortcutRepository(db, mockLogger)

	// Create multiple versions of the same word
	shortcuts := []*domain.Shortcut{
		{Word: "test", Link: "https://test1.com", User: "user1"},
		{Word: "test", Link: "https://test2.com", User: "user2"},
		{Word: "test", Link: "https://test3.com", User: "user3"},
	}

	for _, shortcut := range shortcuts {
		err := repo.Create(context.Background(), shortcut)
		if err != nil {
			t.Fatalf("Failed to create test shortcut: %v", err)
		}
		// Small delay to ensure different timestamps
		time.Sleep(time.Millisecond)
	}

	// Should get the most recent one
	result, err := repo.GetByWord(context.Background(), "test")
	if err != nil {
		t.Errorf("ShortcutRepository.GetByWord() error = %v", err)
		return
	}

	if result == nil {
		t.Error("ShortcutRepository.GetByWord() returned nil")
		return
	}

	// Should be the last one created (highest ID)
	if result.Link != "https://test3.com" {
		t.Errorf("Expected most recent link 'https://test3.com', got '%s'", result.Link)
	}

	if result.User != "user3" {
		t.Errorf("Expected most recent user 'user3', got '%s'", result.User)
	}
}

func TestShortcutRepository_DatabaseError(t *testing.T) {
	// Test with closed database to simulate database errors
	db := setupTestDB(t)
	db.Close() // Close immediately to cause errors

	mockLogger := logger.New(logger.Config{Level: "debug", Format: "text"})
	repo := NewShortcutRepository(db, mockLogger)

	// Test GetByWord with closed DB
	_, err := repo.GetByWord(context.Background(), "test")
	if err == nil {
		t.Error("Expected error with closed database, got nil")
	}

	// Test Create with closed DB
	shortcut := &domain.Shortcut{
		Word: "test",
		Link: "https://test.com",
		User: "testuser",
	}
	err = repo.Create(context.Background(), shortcut)
	if err == nil {
		t.Error("Expected error with closed database, got nil")
	}

	// Test GetAllKeywords with closed DB
	_, err = repo.GetAllKeywords(context.Background())
	if err == nil {
		t.Error("Expected error with closed database, got nil")
	}
}
