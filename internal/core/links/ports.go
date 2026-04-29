package links

import "context"

// Repository is the outbound port for shortcut persistence.
// Implementations live under internal/adapters/persistence.
type Repository interface {
	GetByWord(ctx context.Context, word string) (*Shortcut, error)
	Create(ctx context.Context, shortcut *Shortcut) error
	GetAllKeywords(ctx context.Context) ([]KeywordInfo, error)
}

// QueryRepository is the outbound port for usage analytics.
type QueryRepository interface {
	Create(ctx context.Context, wordID int) error
	GetRecentQueries(ctx context.Context, timeWindowDays, numResults int) ([]PopularQuery, error)
}
