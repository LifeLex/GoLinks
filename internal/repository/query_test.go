package repository

import (
	"context"
	"testing"

	"golinks/internal/domain"
)

func TestQueryRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// First create a shortcut to reference
	shortcutRepo := NewShortcutRepository(db)
	shortcut := &domain.Shortcut{
		Word: "test",
		Link: "https://test.com",
		User: "testuser",
	}
	err := shortcutRepo.Create(context.Background(), shortcut)
	if err != nil {
		t.Fatalf("Failed to create test shortcut: %v", err)
	}

	queryRepo := NewQueryRepository(db)

	tests := []struct {
		name    string
		wordID  int
		wantErr bool
	}{
		{
			name:    "valid word ID",
			wordID:  shortcut.ID,
			wantErr: false,
		},
		{
			name:    "invalid word ID (foreign key constraint)",
			wordID:  999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := queryRepo.Create(context.Background(), tt.wordID)

			if (err != nil) != tt.wantErr {
				t.Errorf("QueryRepository.Create() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestQueryRepository_GetRecentQueries(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Setup test data
	shortcutRepo := NewShortcutRepository(db)
	queryRepo := NewQueryRepository(db)

	// Create shortcuts
	shortcuts := []*domain.Shortcut{
		{Word: "docs", Link: "https://docs.example.com", User: "user1"},
		{Word: "github", Link: "https://github.com", User: "user2"},
		{Word: "search", Link: "https://google.com/search?q={*}", User: "user3"},
	}

	for _, shortcut := range shortcuts {
		err := shortcutRepo.Create(context.Background(), shortcut)
		if err != nil {
			t.Fatalf("Failed to create test shortcut: %v", err)
		}
	}

	// Create queries with different frequencies
	queryData := []struct {
		word  string
		count int
	}{
		{"docs", 5},   // Most popular
		{"github", 3}, // Second most popular
		{"search", 1}, // Least popular
	}

	for _, data := range queryData {
		// Find the shortcut ID
		shortcut, err := shortcutRepo.GetByWord(context.Background(), data.word)
		if err != nil || shortcut == nil {
			t.Fatalf("Failed to find shortcut for word %s", data.word)
		}

		// Create multiple queries for this shortcut
		for i := 0; i < data.count; i++ {
			err := queryRepo.Create(context.Background(), shortcut.ID)
			if err != nil {
				t.Fatalf("Failed to create query for word %s: %v", data.word, err)
			}
		}
	}

	tests := []struct {
		name           string
		timeWindowDays int
		numResults     int
		expectedCount  int
		expectedFirst  string
		expectedSecond string
	}{
		{
			name:           "get top 2 recent queries",
			timeWindowDays: 1,
			numResults:     2,
			expectedCount:  2,
			expectedFirst:  "docs",   // Should be first (5 queries)
			expectedSecond: "github", // Should be second (3 queries)
		},
		{
			name:           "get all recent queries",
			timeWindowDays: 1,
			numResults:     10,
			expectedCount:  3,
			expectedFirst:  "docs",
			expectedSecond: "github",
		},
		{
			name:           "limit to 1 result",
			timeWindowDays: 1,
			numResults:     1,
			expectedCount:  1,
			expectedFirst:  "docs",
			expectedSecond: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queries, err := queryRepo.GetRecentQueries(context.Background(), tt.timeWindowDays, tt.numResults)

			if err != nil {
				t.Errorf("QueryRepository.GetRecentQueries() error = %v", err)
				return
			}

			if len(queries) != tt.expectedCount {
				t.Errorf("QueryRepository.GetRecentQueries() returned %d queries, want %d", len(queries), tt.expectedCount)
				return
			}

			if len(queries) > 0 && queries[0].Word != tt.expectedFirst {
				t.Errorf("QueryRepository.GetRecentQueries() first result = %s, want %s", queries[0].Word, tt.expectedFirst)
			}

			if len(queries) > 1 && tt.expectedSecond != "" && queries[1].Word != tt.expectedSecond {
				t.Errorf("QueryRepository.GetRecentQueries() second result = %s, want %s", queries[1].Word, tt.expectedSecond)
			}

			// Verify counts are correct
			if len(queries) > 0 {
				switch queries[0].Word {
				case "docs":
					if queries[0].Count != 5 {
						t.Errorf("Expected docs count 5, got %d", queries[0].Count)
					}
				case "github":
					if queries[0].Count != 3 {
						t.Errorf("Expected github count 3, got %d", queries[0].Count)
					}
				case "search":
					if queries[0].Count != 1 {
						t.Errorf("Expected search count 1, got %d", queries[0].Count)
					}
				}
			}
		})
	}
}

func TestQueryRepository_GetRecentQueries_TimeWindow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	shortcutRepo := NewShortcutRepository(db)
	queryRepo := NewQueryRepository(db)

	// Create a shortcut
	shortcut := &domain.Shortcut{
		Word: "test",
		Link: "https://test.com",
		User: "testuser",
	}
	err := shortcutRepo.Create(context.Background(), shortcut)
	if err != nil {
		t.Fatalf("Failed to create test shortcut: %v", err)
	}

	// Create a query
	err = queryRepo.Create(context.Background(), shortcut.ID)
	if err != nil {
		t.Fatalf("Failed to create query: %v", err)
	}

	// Test with different time windows
	tests := []struct {
		name           string
		timeWindowDays int
		expectedCount  int
	}{
		{
			name:           "recent queries (1 day)",
			timeWindowDays: 1,
			expectedCount:  1, // Should find the query
		},
		{
			name:           "very old queries (0 days)",
			timeWindowDays: 0,
			expectedCount:  0, // Should not find queries from "today"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queries, err := queryRepo.GetRecentQueries(context.Background(), tt.timeWindowDays, 10)

			if err != nil {
				t.Errorf("QueryRepository.GetRecentQueries() error = %v", err)
				return
			}

			if len(queries) != tt.expectedCount {
				t.Errorf("QueryRepository.GetRecentQueries() with %d day window returned %d queries, want %d",
					tt.timeWindowDays, len(queries), tt.expectedCount)
			}
		})
	}
}

func TestQueryRepository_DatabaseError(t *testing.T) {
	// Test with closed database to simulate database errors
	db := setupTestDB(t)
	db.Close() // Close immediately to cause errors

	repo := NewQueryRepository(db)

	// Test Create with closed DB
	err := repo.Create(context.Background(), 1)
	if err == nil {
		t.Error("Expected error with closed database, got nil")
	}

	// Test GetRecentQueries with closed DB
	_, err = repo.GetRecentQueries(context.Background(), 1, 10)
	if err == nil {
		t.Error("Expected error with closed database, got nil")
	}
}

func TestQueryRepository_EmptyResults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewQueryRepository(db)

	// Test GetRecentQueries with no data
	queries, err := repo.GetRecentQueries(context.Background(), 1, 10)

	if err != nil {
		t.Errorf("QueryRepository.GetRecentQueries() error = %v", err)
		return
	}

	if len(queries) != 0 {
		t.Errorf("QueryRepository.GetRecentQueries() with no data returned %d queries, want 0", len(queries))
	}
}
