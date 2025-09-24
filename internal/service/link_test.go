package service

import (
	"context"
	"testing"
	"time"

	"golinks/internal/domain"
	"golinks/internal/logger"
)

// Mock repositories for testing
type mockShortcutRepository struct {
	shortcuts map[string]*domain.Shortcut
	createErr error
}

func (m *mockShortcutRepository) GetByWord(ctx context.Context, word string) (*domain.Shortcut, error) {
	if shortcut, exists := m.shortcuts[word]; exists {
		return shortcut, nil
	}
	return nil, nil
}

func (m *mockShortcutRepository) Create(ctx context.Context, shortcut *domain.Shortcut) error {
	if m.createErr != nil {
		return m.createErr
	}
	shortcut.ID = len(m.shortcuts) + 1
	m.shortcuts[shortcut.Word] = shortcut
	return nil
}

func (m *mockShortcutRepository) GetAllKeywords(ctx context.Context) ([]domain.KeywordInfo, error) {
	var keywords []domain.KeywordInfo
	for word, shortcut := range m.shortcuts {
		keywords = append(keywords, domain.KeywordInfo{
			Word:      word,
			Link:      shortcut.Link,
			CreatedAt: shortcut.CreatedAt,
		})
	}
	return keywords, nil
}

type mockQueryRepository struct {
	queries   []domain.Query
	createErr error
}

func (m *mockQueryRepository) Create(ctx context.Context, wordID int) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.queries = append(m.queries, domain.Query{
		ID:        len(m.queries) + 1,
		WordID:    wordID,
		CreatedAt: time.Now(),
	})
	return nil
}

func (m *mockQueryRepository) GetRecentQueries(ctx context.Context, timeWindowDays, numResults int) ([]domain.PopularQuery, error) {
	// Simple mock implementation
	return []domain.PopularQuery{
		{Count: 5, Word: "docs", Link: "https://docs.example.com"},
		{Count: 3, Word: "github", Link: "https://github.com"},
	}, nil
}

func TestLinkService_GetLink(t *testing.T) {
	tests := []struct {
		name       string
		shortcuts  map[string]*domain.Shortcut
		word       string
		searchTerm string
		want       string
		wantErr    bool
	}{
		{
			name: "simple URL redirect",
			shortcuts: map[string]*domain.Shortcut{
				"docs": {
					ID:   1,
					Word: "docs",
					Link: "https://docs.example.com",
					User: "testuser",
				},
			},
			word:       "docs",
			searchTerm: "",
			want:       "https://docs.example.com",
			wantErr:    false,
		},
		{
			name: "URL with variable substitution",
			shortcuts: map[string]*domain.Shortcut{
				"search": {
					ID:   1,
					Word: "search",
					Link: "https://google.com/search?q={*}",
					User: "testuser",
				},
			},
			word:       "search",
			searchTerm: "golang",
			want:       "https://google.com/search?q=golang",
			wantErr:    false,
		},
		{
			name: "keyword reference redirect",
			shortcuts: map[string]*domain.Shortcut{
				"d": {
					ID:   1,
					Word: "d",
					Link: "docs",
					User: "testuser",
				},
				"docs": {
					ID:   2,
					Word: "docs",
					Link: "https://docs.example.com",
					User: "testuser",
				},
			},
			word:       "d",
			searchTerm: "",
			want:       "https://docs.example.com",
			wantErr:    false,
		},
		{
			name:       "word not found",
			shortcuts:  map[string]*domain.Shortcut{},
			word:       "nonexistent",
			searchTerm: "",
			want:       "",
			wantErr:    true,
		},
		{
			name: "word with spaces - should split",
			shortcuts: map[string]*domain.Shortcut{
				"search": {
					ID:   1,
					Word: "search",
					Link: "https://google.com/search?q={*}",
					User: "testuser",
				},
			},
			word:       "search golang",
			searchTerm: "",
			want:       "https://google.com/search?q=golang",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortcutRepo := &mockShortcutRepository{shortcuts: tt.shortcuts}
			queryRepo := &mockQueryRepository{}
			mockLogger := logger.New(logger.Config{Level: "debug", Format: "text"})
			service := NewLinkService(shortcutRepo, queryRepo, mockLogger)

			got, err := service.GetLink(context.Background(), tt.word, tt.searchTerm)

			if (err != nil) != tt.wantErr {
				t.Errorf("LinkService.GetLink() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("LinkService.GetLink() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLinkService_UpdateLink(t *testing.T) {
	tests := []struct {
		name      string
		shortcuts map[string]*domain.Shortcut
		request   domain.LinkRequest
		userID    string
		wantErr   bool
	}{
		{
			name:      "valid URL",
			shortcuts: map[string]*domain.Shortcut{},
			request: domain.LinkRequest{
				Word: "docs",
				Link: "https://docs.example.com",
			},
			userID:  "testuser",
			wantErr: false,
		},
		{
			name:      "empty word",
			shortcuts: map[string]*domain.Shortcut{},
			request: domain.LinkRequest{
				Word: "",
				Link: "https://docs.example.com",
			},
			userID:  "testuser",
			wantErr: true,
		},
		{
			name:      "word ending with slash",
			shortcuts: map[string]*domain.Shortcut{},
			request: domain.LinkRequest{
				Word: "docs/",
				Link: "https://docs.example.com",
			},
			userID:  "testuser",
			wantErr: true,
		},
		{
			name:      "recursive link",
			shortcuts: map[string]*domain.Shortcut{},
			request: domain.LinkRequest{
				Word: "test",
				Link: "test",
			},
			userID:  "testuser",
			wantErr: true,
		},
		{
			name:      "invalid URL format",
			shortcuts: map[string]*domain.Shortcut{},
			request: domain.LinkRequest{
				Word: "docs",
				Link: "example.com", // Missing http:// or https://
			},
			userID:  "testuser",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortcutRepo := &mockShortcutRepository{shortcuts: tt.shortcuts}
			queryRepo := &mockQueryRepository{}
			mockLogger := logger.New(logger.Config{Level: "debug", Format: "text"})
			service := NewLinkService(shortcutRepo, queryRepo, mockLogger)

			err := service.UpdateLink(context.Background(), tt.request, tt.userID)

			if (err != nil) != tt.wantErr {
				t.Errorf("LinkService.UpdateLink() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLinkService_GetRecentQueries(t *testing.T) {
	shortcutRepo := &mockShortcutRepository{shortcuts: map[string]*domain.Shortcut{}}
	queryRepo := &mockQueryRepository{}
	mockLogger := logger.New(logger.Config{Level: "debug", Format: "text"})
	service := NewLinkService(shortcutRepo, queryRepo, mockLogger)

	queries, err := service.GetRecentQueries(context.Background())

	if err != nil {
		t.Errorf("LinkService.GetRecentQueries() error = %v", err)
	}

	if len(queries) == 0 {
		t.Error("LinkService.GetRecentQueries() returned empty results")
	}

	// Check that we got expected mock data
	if queries[0].Word != "docs" || queries[0].Count != 5 {
		t.Errorf("LinkService.GetRecentQueries() unexpected first result: %+v", queries[0])
	}
}

func TestLinkService_GetAllKeywords(t *testing.T) {
	shortcuts := map[string]*domain.Shortcut{
		"docs": {
			ID:        1,
			Word:      "docs",
			Link:      "https://docs.example.com",
			User:      "testuser",
			CreatedAt: time.Now(),
		},
		"github": {
			ID:        2,
			Word:      "github",
			Link:      "https://github.com",
			User:      "testuser",
			CreatedAt: time.Now(),
		},
	}

	shortcutRepo := &mockShortcutRepository{shortcuts: shortcuts}
	queryRepo := &mockQueryRepository{}
	mockLogger := logger.New(logger.Config{Level: "debug", Format: "text"})
	service := NewLinkService(shortcutRepo, queryRepo, mockLogger)

	keywords, err := service.GetAllKeywords(context.Background())

	if err != nil {
		t.Errorf("LinkService.GetAllKeywords() error = %v", err)
	}

	// Should return only URLs
	if len(keywords) != 2 {
		t.Errorf("LinkService.GetAllKeywords() expected 2 keywords, got %d", len(keywords))
	}

	// Check that we have both keywords
	keywordMap := make(map[string]bool)
	for _, keyword := range keywords {
		keywordMap[keyword.Word] = true
	}

	if !keywordMap["docs"] {
		t.Error("LinkService.GetAllKeywords() missing 'docs' keyword")
	}

	if !keywordMap["github"] {
		t.Error("LinkService.GetAllKeywords() missing 'github' keyword")
	}
}

// Test utility functions
func Test_isURL(t *testing.T) {
	tests := []struct {
		name string
		link string
		want bool
	}{
		{"http URL", "http://example.com", true},
		{"https URL", "https://example.com", true},
		{"not a URL", "docs", false},
		{"empty string", "", false},
		{"ftp URL", "ftp://example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isURL(tt.link); got != tt.want {
				t.Errorf("isURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processResultLink(t *testing.T) {
	tests := []struct {
		name       string
		link       string
		searchTerm string
		want       string
	}{
		{
			name:       "no substitution",
			link:       "https://example.com",
			searchTerm: "test",
			want:       "https://example.com",
		},
		{
			name:       "simple substitution",
			link:       "https://google.com/search?q={*}",
			searchTerm: "golang",
			want:       "https://google.com/search?q=golang",
		},
		{
			name:       "multiple substitutions",
			link:       "https://example.com/{*}/docs/{*}",
			searchTerm: "api",
			want:       "https://example.com/api/docs/api",
		},
		{
			name:       "URL encoding",
			link:       "https://google.com/search?q={*}",
			searchTerm: "hello world",
			want:       "https://google.com/search?q=hello+world",
		},
		{
			name:       "empty search term",
			link:       "https://example.com/{*}",
			searchTerm: "",
			want:       "https://example.com/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := processResultLink(tt.link, tt.searchTerm); got != tt.want {
				t.Errorf("processResultLink() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_moveLastWord(t *testing.T) {
	tests := []struct {
		name     string
		moveFrom string
		moveTo   string
		wantFrom string
		wantTo   string
	}{
		{
			name:     "simple move",
			moveFrom: "search golang",
			moveTo:   "",
			wantFrom: "search",
			wantTo:   "golang",
		},
		{
			name:     "move to existing",
			moveFrom: "search golang",
			moveTo:   "tutorial",
			wantFrom: "search",
			wantTo:   "golang tutorial",
		},
		{
			name:     "single word",
			moveFrom: "golang",
			moveTo:   "",
			wantFrom: "",
			wantTo:   "golang",
		},
		{
			name:     "empty from",
			moveFrom: "",
			moveTo:   "test",
			wantFrom: "",
			wantTo:   "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFrom, gotTo := moveLastWord(tt.moveFrom, tt.moveTo)
			if gotFrom != tt.wantFrom {
				t.Errorf("moveLastWord() gotFrom = %v, want %v", gotFrom, tt.wantFrom)
			}
			if gotTo != tt.wantTo {
				t.Errorf("moveLastWord() gotTo = %v, want %v", gotTo, tt.wantTo)
			}
		})
	}
}
