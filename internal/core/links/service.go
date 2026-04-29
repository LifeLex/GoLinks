package links

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"golinks/internal/platform/logger"
)

// Service implements the golinks use cases (resolve, create/update, list, recent queries).
// It depends only on the Repository and QueryRepository ports — no DB types leak in here.
type Service struct {
	shortcutRepo Repository
	queryRepo    QueryRepository
	logger       *logger.Logger
}

// NewService wires the use cases against persistence ports.
func NewService(shortcutRepo Repository, queryRepo QueryRepository, log *logger.Logger) *Service {
	log.Info("Link service initialized")
	return &Service{
		shortcutRepo: shortcutRepo,
		queryRepo:    queryRepo,
		logger:       log,
	}
}

// GetLink resolves a golink query to a URL. Handles {*} substitution, keyword
// aliasing (recursion when a stored "link" is itself a keyword), and the
// space-splitting fallback so `go search golang` resolves through the
// `search` keyword with `golang` as the search term.
func (s *Service) GetLink(ctx context.Context, word, searchTerm string) (string, error) {
	word = strings.TrimSpace(word)
	s.logger.Debug("Processing golink query: '%s' (search: '%s')", word, searchTerm)

	shortcut, err := s.shortcutRepo.GetByWord(ctx, word)
	if err != nil {
		s.logger.Error("Failed to get shortcut from repository: %v", err)
		return "", fmt.Errorf("failed to get shortcut: %w", err)
	}

	if shortcut == nil {
		if strings.Contains(word, " ") {
			newWord, newSearchTerm := moveLastWord(word, searchTerm)
			s.logger.Debug("Splitting word '%s' -> '%s' and retrying", word, newWord)
			return s.GetLink(ctx, newWord, newSearchTerm)
		}

		query := strings.Join([]string{word, searchTerm}, " ")
		s.logger.Warn("No shortcut found for query: %s", query)
		return "", InvalidQueryError{
			Message: fmt.Sprintf("Unable to find link for query %s", query),
		}
	}

	s.logger.Info("Found shortcut: id=%d link='%s' user='%s'", shortcut.ID, shortcut.Link, shortcut.User)

	if err := s.queryRepo.Create(ctx, shortcut.ID); err != nil {
		s.logger.Error("Failed to log query usage for shortcut %d: %v", shortcut.ID, err)
		// Don't fail the request for logging errors.
	}

	if !isURL(shortcut.Link) {
		s.logger.Debug("Link is a keyword reference '%s', recursing", shortcut.Link)
		return s.GetLink(ctx, shortcut.Link, searchTerm)
	}

	resultLink := processResultLink(shortcut.Link, searchTerm)
	s.logger.Info("Link resolution successful: '%s' -> '%s'", word, resultLink)
	return resultLink, nil
}

// UpdateLink creates a new shortcut row (multiple rows per word are allowed —
// GetByWord returns the most recent).
func (s *Service) UpdateLink(ctx context.Context, req LinkRequest, userID string) error {
	s.logger.Info("Processing link update: word='%s' link='%s' user='%s'", req.Word, req.Link, userID)

	if err := s.validateLinkRequest(req); err != nil {
		s.logger.Warn("Link request validation failed: %v", err)
		return err
	}

	if !isURL(req.Link) {
		s.logger.Warn("Invalid URL format: %s", req.Link)
		return InvalidQueryError{Message: "URL must start with http:// or https://"}
	}

	shortcut := &Shortcut{
		Word:      req.Word,
		Link:      req.Link,
		User:      userID,
		CreatedAt: time.Now(),
	}

	if err := s.shortcutRepo.Create(ctx, shortcut); err != nil {
		s.logger.Error("Failed to create shortcut in repository: %v", err)
		return fmt.Errorf("failed to create shortcut: %w", err)
	}

	s.logger.Info("Link update completed successfully: id=%d", shortcut.ID)
	return nil
}

// GetRecentQueries returns the most-used keywords over a fixed window (3 days, max 20).
func (s *Service) GetRecentQueries(ctx context.Context) ([]PopularQuery, error) {
	s.logger.Debug("Fetching recent queries (3 days, max 20 results)")

	queries, err := s.queryRepo.GetRecentQueries(ctx, 3, 20)
	if err != nil {
		s.logger.Error("Failed to get recent queries: %v", err)
		return nil, err
	}

	s.logger.Debug("Recent queries retrieved successfully: %d queries", len(queries))
	return queries, nil
}

// GetAllKeywords returns every keyword whose link is a URL (aliases are filtered out
// because the homepage table only shows directly-resolvable rows).
func (s *Service) GetAllKeywords(ctx context.Context) ([]KeywordInfo, error) {
	s.logger.Debug("Fetching all keywords")

	keywords, err := s.shortcutRepo.GetAllKeywords(ctx)
	if err != nil {
		s.logger.Error("Failed to get all keywords: %v", err)
		return nil, err
	}

	result := make([]KeywordInfo, 0, len(keywords))
	for _, keyword := range keywords {
		if isURL(keyword.Link) {
			result = append(result, keyword)
		}
	}

	s.logger.Debug("Keywords retrieved successfully: %d total, %d URLs", len(keywords), len(result))
	return result, nil
}

func (s *Service) validateLinkRequest(req LinkRequest) error {
	req.Word = strings.TrimSpace(req.Word)
	req.Link = strings.TrimSpace(req.Link)

	switch {
	case req.Word == "":
		return InvalidQueryError{Message: "No word given, cannot setup a golink"}
	case strings.HasSuffix(req.Word, "/"):
		return InvalidQueryError{Message: "Words ending in a '/' are not supported"}
	case req.Link == "":
		return InvalidQueryError{Message: "No link given, cannot setup a golink"}
	case req.Link == req.Word:
		return InvalidQueryError{Message: "Word points to itself, will cause a recursive lookup"}
	}
	return nil
}

func isURL(link string) bool {
	return strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://")
}

func processResultLink(link, searchTerm string) string {
	searchTerm = strings.ReplaceAll(searchTerm, "{*}", "")
	searchTerm = strings.TrimSpace(searchTerm)
	searchTerm = url.QueryEscape(searchTerm)

	resultLink := strings.ReplaceAll(link, "{*}", searchTerm)
	return strings.TrimSpace(resultLink)
}

// moveLastWord pops the trailing token from `from` and prepends it to `to`,
// supporting the "go search golang" → resolve("search", "golang") flow.
func moveLastWord(from, to string) (string, string) {
	fromWords := strings.Fields(from)
	if len(fromWords) == 0 {
		return from, to
	}

	lastWord := fromWords[len(fromWords)-1]
	toWords := strings.Fields(to)

	fromOut := strings.Join(fromWords[:len(fromWords)-1], " ")
	toOut := strings.Join(append([]string{lastWord}, toWords...), " ")

	return fromOut, toOut
}
