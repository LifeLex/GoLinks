package links

import (
	"context"
	"testing"
	"time"

	"golinks/internal/platform/logger"
)

type mockShortcutRepository struct {
	shortcuts map[string]*Shortcut
	createErr error
}

func (m *mockShortcutRepository) GetByWord(_ context.Context, word string) (*Shortcut, error) {
	if shortcut, exists := m.shortcuts[word]; exists {
		return shortcut, nil
	}
	return nil, nil
}

func (m *mockShortcutRepository) Create(_ context.Context, shortcut *Shortcut) error {
	if m.createErr != nil {
		return m.createErr
	}
	shortcut.ID = len(m.shortcuts) + 1
	m.shortcuts[shortcut.Word] = shortcut
	return nil
}

func (m *mockShortcutRepository) GetAllKeywords(_ context.Context) ([]KeywordInfo, error) {
	keywords := make([]KeywordInfo, 0, len(m.shortcuts))
	for word, shortcut := range m.shortcuts {
		keywords = append(keywords, KeywordInfo{
			Word:      word,
			Link:      shortcut.Link,
			CreatedAt: shortcut.CreatedAt,
		})
	}
	return keywords, nil
}

type mockQueryRepository struct {
	creates   int
	createErr error
}

func (m *mockQueryRepository) Create(_ context.Context, _ int) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.creates++
	return nil
}

func (m *mockQueryRepository) GetRecentQueries(_ context.Context, _, _ int) ([]PopularQuery, error) {
	return []PopularQuery{
		{Count: 5, Word: "docs", Link: "https://docs.example.com"},
		{Count: 3, Word: "github", Link: "https://github.com"},
	}, nil
}

func TestService_GetLink(t *testing.T) {
	tests := []struct {
		name       string
		shortcuts  map[string]*Shortcut
		word       string
		searchTerm string
		want       string
		wantErr    bool
	}{
		{
			name:       "simple URL redirect",
			shortcuts:  map[string]*Shortcut{"docs": {ID: 1, Word: "docs", Link: "https://docs.example.com", User: "u"}},
			word:       "docs",
			searchTerm: "",
			want:       "https://docs.example.com",
			wantErr:    false,
		},
		{
			name:       "URL with variable substitution",
			shortcuts:  map[string]*Shortcut{"search": {ID: 1, Word: "search", Link: "https://google.com/search?q={*}", User: "u"}},
			word:       "search",
			searchTerm: "golang",
			want:       "https://google.com/search?q=golang",
			wantErr:    false,
		},
		{
			name: "keyword reference redirect",
			shortcuts: map[string]*Shortcut{
				"d":    {ID: 1, Word: "d", Link: "docs", User: "u"},
				"docs": {ID: 2, Word: "docs", Link: "https://docs.example.com", User: "u"},
			},
			word:       "d",
			searchTerm: "",
			want:       "https://docs.example.com",
			wantErr:    false,
		},
		{
			name:       "word not found",
			shortcuts:  map[string]*Shortcut{},
			word:       "nonexistent",
			searchTerm: "",
			want:       "",
			wantErr:    true,
		},
		{
			name:       "word with spaces - should split",
			shortcuts:  map[string]*Shortcut{"search": {ID: 1, Word: "search", Link: "https://google.com/search?q={*}", User: "u"}},
			word:       "search golang",
			searchTerm: "",
			want:       "https://google.com/search?q=golang",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newServiceForTest(tt.shortcuts)
			got, err := svc.GetLink(context.Background(), tt.word, tt.searchTerm)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetLink() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("GetLink() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestService_UpdateLink(t *testing.T) {
	tests := []struct {
		name    string
		request LinkRequest
		wantErr bool
	}{
		{"valid URL", LinkRequest{Word: "docs", Link: "https://docs.example.com"}, false},
		{"empty word", LinkRequest{Word: "", Link: "https://docs.example.com"}, true},
		{"word ending with slash", LinkRequest{Word: "docs/", Link: "https://docs.example.com"}, true},
		{"recursive link", LinkRequest{Word: "test", Link: "test"}, true},
		{"invalid URL format", LinkRequest{Word: "docs", Link: "example.com"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newServiceForTest(nil)
			err := svc.UpdateLink(context.Background(), tt.request, "tester")
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateLink() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_GetRecentQueries(t *testing.T) {
	svc := newServiceForTest(nil)
	queries, err := svc.GetRecentQueries(context.Background())
	if err != nil {
		t.Fatalf("GetRecentQueries() error = %v", err)
	}
	if len(queries) == 0 {
		t.Fatal("GetRecentQueries() returned empty results")
	}
	if queries[0].Word != "docs" || queries[0].Count != 5 {
		t.Errorf("unexpected first result: %+v", queries[0])
	}
}

func TestService_GetAllKeywords(t *testing.T) {
	now := time.Now()
	shortcuts := map[string]*Shortcut{
		"docs":   {ID: 1, Word: "docs", Link: "https://docs.example.com", CreatedAt: now},
		"github": {ID: 2, Word: "github", Link: "https://github.com", CreatedAt: now},
		"alias":  {ID: 3, Word: "alias", Link: "docs", CreatedAt: now},
	}
	svc := newServiceForTest(shortcuts)

	keywords, err := svc.GetAllKeywords(context.Background())
	if err != nil {
		t.Fatalf("GetAllKeywords() error = %v", err)
	}
	// Aliases (non-URL links) should be filtered out: 3 stored, 2 returned.
	if len(keywords) != 2 {
		t.Errorf("expected 2 keywords, got %d", len(keywords))
	}
}

func Test_isURL(t *testing.T) {
	tests := []struct {
		link string
		want bool
	}{
		{"http://example.com", true},
		{"https://example.com", true},
		{"docs", false},
		{"", false},
		{"ftp://example.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.link, func(t *testing.T) {
			if got := isURL(tt.link); got != tt.want {
				t.Errorf("isURL(%q) = %v, want %v", tt.link, got, tt.want)
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
		{"no substitution", "https://example.com", "test", "https://example.com"},
		{"simple substitution", "https://google.com/search?q={*}", "golang", "https://google.com/search?q=golang"},
		{"multiple substitutions", "https://example.com/{*}/docs/{*}", "api", "https://example.com/api/docs/api"},
		{"URL encoding", "https://google.com/search?q={*}", "hello world", "https://google.com/search?q=hello+world"},
		{"empty search term", "https://example.com/{*}", "", "https://example.com/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := processResultLink(tt.link, tt.searchTerm); got != tt.want {
				t.Errorf("processResultLink() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_moveLastWord(t *testing.T) {
	tests := []struct {
		name     string
		from, to string
		wantFrom string
		wantTo   string
	}{
		{"simple move", "search golang", "", "search", "golang"},
		{"move to existing", "search golang", "tutorial", "search", "golang tutorial"},
		{"single word", "golang", "", "", "golang"},
		{"empty from", "", "test", "", "test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFrom, gotTo := moveLastWord(tt.from, tt.to)
			if gotFrom != tt.wantFrom || gotTo != tt.wantTo {
				t.Errorf("moveLastWord() = (%q,%q), want (%q,%q)", gotFrom, gotTo, tt.wantFrom, tt.wantTo)
			}
		})
	}
}

func newServiceForTest(shortcuts map[string]*Shortcut) *Service {
	if shortcuts == nil {
		shortcuts = map[string]*Shortcut{}
	}
	return NewService(
		&mockShortcutRepository{shortcuts: shortcuts},
		&mockQueryRepository{},
		logger.New(logger.Config{Level: "debug", Format: "text"}),
	)
}
