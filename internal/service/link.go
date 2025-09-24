package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"golinks/internal/domain"
	"golinks/internal/logger"
)

// ShortcutRepository interface for shortcut operations
type ShortcutRepository interface {
	GetByWord(ctx context.Context, word string) (*domain.Shortcut, error)
	Create(ctx context.Context, shortcut *domain.Shortcut) error
	GetAllKeywords(ctx context.Context) ([]domain.KeywordInfo, error)
}

// QueryRepository interface for query operations
type QueryRepository interface {
	Create(ctx context.Context, wordID int) error
	GetRecentQueries(ctx context.Context, timeWindowDays, numResults int) ([]domain.PopularQuery, error)
}

// LinkService handles business logic for golinks
type LinkService struct {
	shortcutRepo ShortcutRepository
	queryRepo    QueryRepository
	logger       *logger.Logger
}

// NewLinkService creates a new link service
func NewLinkService(shortcutRepo ShortcutRepository, queryRepo QueryRepository, log *logger.Logger) *LinkService {
	log.Info("Link service initialized")
	return &LinkService{
		shortcutRepo: shortcutRepo,
		queryRepo:    queryRepo,
		logger:       log,
	}
}

// InvalidQueryError represents an error when a query cannot be resolved
type InvalidQueryError struct {
	Message string
}

func (e InvalidQueryError) Error() string {
	return e.Message
}

// GetLink resolves a golink query to a URL
func (s *LinkService) GetLink(ctx context.Context, word string, searchTerm string) (string, error) {
	word = strings.TrimSpace(word)
	s.logger.Debug("Processing golink query: '%s' (search: '%s')", word, searchTerm)

	shortcut, err := s.shortcutRepo.GetByWord(ctx, word)
	if err != nil {
		s.logger.Error("Failed to get shortcut from repository: %v", err)
		return "", fmt.Errorf("failed to get shortcut: %w", err)
	}

	if shortcut == nil {
		// Try splitting the word if it contains spaces
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

	// Log the query
	if err := s.queryRepo.Create(ctx, shortcut.ID); err != nil {
		s.logger.Error("Failed to log query usage for shortcut %d: %v", shortcut.ID, err)
		// Don't fail the request for logging errors
	}

	// Handle different types of links
	if !isURL(shortcut.Link) {
		s.logger.Debug("Link is a keyword reference '%s', recursing", shortcut.Link)
		// This is a keyword reference, recurse
		return s.GetLink(ctx, shortcut.Link, searchTerm)
	}

	// Process URL with search term substitution
	resultLink := processResultLink(shortcut.Link, searchTerm)
	s.logger.Info("Link resolution successful: '%s' -> '%s'", word, resultLink)
	return resultLink, nil
}

// UpdateLink creates or updates a golink
func (s *LinkService) UpdateLink(ctx context.Context, req domain.LinkRequest, userID string) error {
	s.logger.Info("Processing link update: word='%s' link='%s' user='%s'", req.Word, req.Link, userID)

	// Validate the request
	if err := s.validateLinkRequest(ctx, req); err != nil {
		s.logger.Warn("Link request validation failed: %v", err)
		return err
	}

	// Validate that the link is a proper URL
	if !isURL(req.Link) {
		s.logger.Warn("Invalid URL format: %s", req.Link)
		return InvalidQueryError{
			Message: "URL must start with http:// or https://",
		}
	}

	shortcut := &domain.Shortcut{
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

// GetRecentQueries retrieves popular queries
func (s *LinkService) GetRecentQueries(ctx context.Context) ([]domain.PopularQuery, error) {
	s.logger.Debug("Fetching recent queries (3 days, max 20 results)")

	queries, err := s.queryRepo.GetRecentQueries(ctx, 3, 20)
	if err != nil {
		s.logger.Error("Failed to get recent queries: %v", err)
		return nil, err
	}

	s.logger.Debug("Recent queries retrieved successfully: %d queries", len(queries))
	return queries, nil
}

// GetAllKeywords retrieves all keywords
func (s *LinkService) GetAllKeywords(ctx context.Context) ([]domain.KeywordInfo, error) {
	s.logger.Debug("Fetching all keywords")

	keywords, err := s.shortcutRepo.GetAllKeywords(ctx)
	if err != nil {
		s.logger.Error("Failed to get all keywords: %v", err)
		return nil, err
	}

	// Filter to only return URLs
	var result []domain.KeywordInfo
	for _, keyword := range keywords {
		if isURL(keyword.Link) {
			result = append(result, keyword)
		}
	}

	s.logger.Debug("Keywords retrieved successfully: %d total, %d URLs", len(keywords), len(result))
	return result, nil
}

// validateLinkRequest validates a link request
func (s *LinkService) validateLinkRequest(ctx context.Context, req domain.LinkRequest) error {
	req.Word = strings.TrimSpace(req.Word)
	req.Link = strings.TrimSpace(req.Link)

	if req.Word == "" {
		return InvalidQueryError{Message: "No word given, cannot setup a golink"}
	}

	if strings.HasSuffix(req.Word, "/") {
		return InvalidQueryError{Message: "Words ending in a '/' are not supported"}
	}

	if req.Link == "" {
		return InvalidQueryError{Message: "No link given, cannot setup a golink"}
	}

	if req.Link == req.Word {
		return InvalidQueryError{Message: "Word points to itself, will cause a recursive lookup"}
	}

	return nil
}

// isURL checks if a string is a URL
func isURL(link string) bool {
	return strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://")
}

// processResultLink processes a URL with search term substitution
func processResultLink(link, searchTerm string) string {
	// Remove wildcard markers and encode spaces
	searchTerm = strings.ReplaceAll(searchTerm, "{*}", "")
	searchTerm = strings.TrimSpace(searchTerm)
	searchTerm = url.QueryEscape(searchTerm)

	// Replace wildcards in the link
	resultLink := strings.ReplaceAll(link, "{*}", searchTerm)
	return strings.TrimSpace(resultLink)
}

// moveLastWord moves the last word from the first string to the beginning of the second string
func moveLastWord(moveFrom, moveTo string) (string, string) {
	moveFromWords := strings.Fields(moveFrom)
	if len(moveFromWords) == 0 {
		return moveFrom, moveTo
	}

	lastWord := moveFromWords[len(moveFromWords)-1]
	moveToWords := strings.Fields(moveTo)

	moveFromOut := strings.Join(moveFromWords[:len(moveFromWords)-1], " ")
	moveToOut := strings.Join(append([]string{lastWord}, moveToWords...), " ")

	return moveFromOut, moveToOut
}
