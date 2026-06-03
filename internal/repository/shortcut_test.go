package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"golinks/internal/database"
	"golinks/internal/domain"
	"golinks/internal/logger"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates an in-memory SQLite database for testing, using the same
// migrations as production so the test schema can never drift from the real one.
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
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

func TestShortcutRepository_TagsRoundTrip(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewShortcutRepository(db, logger.New(logger.Config{Level: "error", Format: "text"}))

	sc := &domain.Shortcut{
		Word: "grafana",
		Link: "https://grafana.example.com",
		User: "user1",
		Tags: []string{"infra", "monitoring", "infra"}, // duplicate is ignored by the unique index
	}
	if err := repo.Create(context.Background(), sc); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	keywords, err := repo.GetAllKeywords(context.Background())
	if err != nil {
		t.Fatalf("GetAllKeywords() error = %v", err)
	}
	if len(keywords) != 1 {
		t.Fatalf("got %d keywords, want 1", len(keywords))
	}
	// Tags come back ordered alphabetically (idx scan) and de-duplicated.
	got := keywords[0].Tags
	if len(got) != 2 || got[0] != "infra" || got[1] != "monitoring" {
		t.Errorf("tags = %v, want [infra monitoring]", got)
	}
}

func TestShortcutRepository_Search(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewShortcutRepository(db, logger.New(logger.Config{Level: "error", Format: "text"}))

	seed := []*domain.Shortcut{
		{Word: "grafana", Link: "https://grafana.example.com", User: "u", Tags: []string{"infra", "monitoring"}},
		{Word: "github", Link: "https://github.com", User: "u", Tags: []string{"code"}},
		{Word: "calendar", Link: "https://cal.example.com", User: "u"},
	}
	for _, s := range seed {
		if err := repo.Create(context.Background(), s); err != nil {
			t.Fatalf("seed Create() error = %v", err)
		}
	}

	tests := []struct {
		name      string
		query     string
		wantWords []string
	}{
		{"match by word", "graf", []string{"grafana"}},
		{"match by link host", "github.com", []string{"github"}},
		{"match by tag", "monitoring", []string{"grafana"}},
		{"case-insensitive", "GRAF", []string{"grafana"}},
		{"no match", "nonexistent", nil},
		{"literal percent is escaped", "%", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := repo.Search(context.Background(), tt.query, 50)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}
			if len(results) != len(tt.wantWords) {
				t.Fatalf("Search(%q) returned %d results %v, want %v", tt.query, len(results), wordsOf(results), tt.wantWords)
			}
			for i, w := range tt.wantWords {
				if results[i].Word != w {
					t.Errorf("result[%d].Word = %q, want %q", i, results[i].Word, w)
				}
			}
		})
	}
}

func TestShortcutRepository_DeleteByWord(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewShortcutRepository(db, logger.New(logger.Config{Level: "error", Format: "text"}))
	queryRepo := NewQueryRepository(db, logger.New(logger.Config{Level: "error", Format: "text"}))
	ctx := context.Background()

	// Two revisions of "docs" (with tags + a query-log hit), plus an unrelated "gh".
	for _, sc := range []*domain.Shortcut{
		{Word: "docs", Link: "https://docs.example.com", User: "u", Tags: []string{"infra"}},
		{Word: "docs", Link: "https://docs.example.com/v2", User: "u", Tags: []string{"infra", "wiki"}},
		{Word: "gh", Link: "https://github.com", User: "u", Tags: []string{"code"}},
	} {
		if err := repo.Create(ctx, sc); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	// Log a query hit against the latest docs revision so we exercise the
	// dependent-row cleanup (queries has no ON DELETE CASCADE).
	docs, _ := repo.GetByWord(ctx, "docs")
	if err := queryRepo.Create(ctx, docs.ID); err != nil {
		t.Fatalf("query Create() error = %v", err)
	}

	removed, err := repo.DeleteByWord(ctx, "docs")
	if err != nil {
		t.Fatalf("DeleteByWord() error = %v", err)
	}
	if removed != 2 {
		t.Errorf("removed = %d, want 2 (both revisions)", removed)
	}

	// "docs" is gone; "gh" survives.
	keywords, err := repo.GetAllKeywords(ctx)
	if err != nil {
		t.Fatalf("GetAllKeywords() error = %v", err)
	}
	if len(keywords) != 1 || keywords[0].Word != "gh" {
		t.Errorf("after delete, keywords = %v, want only [gh]", wordsOf(keywords))
	}

	// Deleting again removes nothing (idempotent at the repo level).
	if removed, _ := repo.DeleteByWord(ctx, "docs"); removed != 0 {
		t.Errorf("second delete removed = %d, want 0", removed)
	}
}

func wordsOf(keywords []domain.KeywordInfo) []string {
	out := make([]string, len(keywords))
	for i, k := range keywords {
		out[i] = k.Word
	}
	return out
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
