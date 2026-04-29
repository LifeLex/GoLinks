// External test package — required to break the import cycle that would
// otherwise form: persistence_test → migrations → persistence.
package persistence_test

import (
	"context"
	"testing"
	"time"

	"golinks/internal/adapters/persistence"
	// Side-effect import: registers Goose migrations via init() so
	// setupTestDB runs the same schema production does.
	_ "golinks/internal/adapters/persistence/migrations"
	"golinks/internal/core/links"
	"golinks/internal/platform/database"
	"golinks/internal/platform/logger"

	"gorm.io/gorm"
)

// setupTestDB opens an in-memory SQLite database via GORM and runs the
// production migrations against it. Used by every repository test — keeps
// test schema and prod schema strictly in lockstep.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.OpenGorm(database.DriverSQLite, ":memory:")
	if err != nil {
		t.Fatalf("OpenGorm: %v", err)
	}
	if err := database.Migrate(db, database.DriverSQLite); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return db
}

func newTestLogger() *logger.Logger {
	return logger.New(logger.Config{Level: "debug", Format: "text"})
}

// --- LinksRepo --------------------------------------------------------------

func TestLinksRepo_GetByWord(t *testing.T) {
	db := setupTestDB(t)
	repo := persistence.NewLinksRepo(db, newTestLogger())

	if err := repo.Create(context.Background(), &links.Shortcut{
		Word: "docs", Link: "https://docs.example.com", User: "u",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	tests := []struct {
		name string
		word string
		want *links.Shortcut
	}{
		{"existing word", "docs", &links.Shortcut{Word: "docs", Link: "https://docs.example.com", User: "u"}},
		{"non-existing word", "missing", nil},
		{"empty word", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetByWord(context.Background(), tt.word)
			if err != nil {
				t.Fatalf("GetByWord: %v", err)
			}
			if (tt.want == nil) != (got == nil) {
				t.Fatalf("got = %v, want %v", got, tt.want)
			}
			if got != nil && (got.Word != tt.want.Word || got.Link != tt.want.Link) {
				t.Errorf("got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLinksRepo_Create_AssignsID(t *testing.T) {
	db := setupTestDB(t)
	repo := persistence.NewLinksRepo(db, newTestLogger())

	s := &links.Shortcut{Word: "github", Link: "https://github.com", User: "u"}
	if err := repo.Create(context.Background(), s); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if s.ID == 0 {
		t.Error("expected ID to be set after Create")
	}
	if s.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set after Create")
	}
}

func TestLinksRepo_GetByWord_MostRecent(t *testing.T) {
	db := setupTestDB(t)
	repo := persistence.NewLinksRepo(db, newTestLogger())

	for i, link := range []string{"https://test1.com", "https://test2.com", "https://test3.com"} {
		if err := repo.Create(context.Background(), &links.Shortcut{
			Word: "test", Link: link, User: "user",
		}); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
		time.Sleep(time.Millisecond)
	}

	got, err := repo.GetByWord(context.Background(), "test")
	if err != nil {
		t.Fatalf("GetByWord: %v", err)
	}
	if got == nil || got.Link != "https://test3.com" {
		t.Errorf("got = %+v, want most recent", got)
	}
}

func TestLinksRepo_GetAllKeywords(t *testing.T) {
	db := setupTestDB(t)
	repo := persistence.NewLinksRepo(db, newTestLogger())

	for _, s := range []*links.Shortcut{
		{Word: "docs", Link: "https://docs.example.com", User: "u1"},
		{Word: "github", Link: "https://github.com", User: "u2"},
		{Word: "docs", Link: "https://docs.example.com/v2", User: "u1"},
	} {
		if err := repo.Create(context.Background(), s); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	keywords, err := repo.GetAllKeywords(context.Background())
	if err != nil {
		t.Fatalf("GetAllKeywords: %v", err)
	}
	if len(keywords) != 2 {
		t.Errorf("got %d keywords, want 2", len(keywords))
	}
	for _, k := range keywords {
		if k.Word == "docs" && k.Link != "https://docs.example.com/v2" {
			t.Errorf("docs link = %q, want most recent", k.Link)
		}
	}
}

// --- QueriesRepo ------------------------------------------------------------

func TestQueriesRepo_Create_RespectsFK(t *testing.T) {
	db := setupTestDB(t)
	linksRepo := persistence.NewLinksRepo(db, newTestLogger())
	queriesRepo := persistence.NewQueriesRepo(db, newTestLogger())

	s := &links.Shortcut{Word: "test", Link: "https://test.com", User: "u"}
	if err := linksRepo.Create(context.Background(), s); err != nil {
		t.Fatalf("Create shortcut: %v", err)
	}

	if err := queriesRepo.Create(context.Background(), s.ID); err != nil {
		t.Errorf("valid FK: %v", err)
	}
	if err := queriesRepo.Create(context.Background(), 999); err == nil {
		t.Error("expected FK violation for missing word_id")
	}
}

func TestQueriesRepo_GetRecentQueries(t *testing.T) {
	db := setupTestDB(t)
	linksRepo := persistence.NewLinksRepo(db, newTestLogger())
	queriesRepo := persistence.NewQueriesRepo(db, newTestLogger())

	type seed struct {
		word, link string
		count      int
	}
	for _, s := range []seed{
		{"docs", "https://docs.example.com", 5},
		{"github", "https://github.com", 3},
		{"search", "https://google.com/search?q={*}", 1},
	} {
		shortcut := &links.Shortcut{Word: s.word, Link: s.link, User: "u"}
		if err := linksRepo.Create(context.Background(), shortcut); err != nil {
			t.Fatalf("seed shortcut: %v", err)
		}
		for i := 0; i < s.count; i++ {
			if err := queriesRepo.Create(context.Background(), shortcut.ID); err != nil {
				t.Fatalf("seed query: %v", err)
			}
		}
	}

	tests := []struct {
		name          string
		days, limit   int
		expectedCount int
		expectedFirst string
	}{
		{"top 2", 1, 2, 2, "docs"},
		{"all", 1, 10, 3, "docs"},
		{"limit 1", 1, 1, 1, "docs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := queriesRepo.GetRecentQueries(context.Background(), tt.days, tt.limit)
			if err != nil {
				t.Fatalf("GetRecentQueries: %v", err)
			}
			if len(got) != tt.expectedCount {
				t.Fatalf("len = %d, want %d", len(got), tt.expectedCount)
			}
			if got[0].Word != tt.expectedFirst {
				t.Errorf("first = %q, want %q", got[0].Word, tt.expectedFirst)
			}
		})
	}
}

func TestQueriesRepo_TimeWindow(t *testing.T) {
	db := setupTestDB(t)
	linksRepo := persistence.NewLinksRepo(db, newTestLogger())
	queriesRepo := persistence.NewQueriesRepo(db, newTestLogger())

	s := &links.Shortcut{Word: "test", Link: "https://test.com", User: "u"}
	if err := linksRepo.Create(context.Background(), s); err != nil {
		t.Fatalf("Create shortcut: %v", err)
	}
	if err := queriesRepo.Create(context.Background(), s.ID); err != nil {
		t.Fatalf("Create query: %v", err)
	}

	tests := []struct {
		name          string
		days          int
		expectedCount int
	}{
		{"1 day window", 1, 1},
		{"0 day window", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := queriesRepo.GetRecentQueries(context.Background(), tt.days, 10)
			if err != nil {
				t.Fatalf("GetRecentQueries: %v", err)
			}
			if len(got) != tt.expectedCount {
				t.Errorf("len = %d, want %d", len(got), tt.expectedCount)
			}
		})
	}
}

func TestQueriesRepo_EmptyResults(t *testing.T) {
	db := setupTestDB(t)
	repo := persistence.NewQueriesRepo(db, newTestLogger())

	got, err := repo.GetRecentQueries(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("GetRecentQueries: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d results, want 0", len(got))
	}
}
