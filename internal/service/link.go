package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"golinks/internal/domain"
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
}

// NewLinkService creates a new link service
func NewLinkService(shortcutRepo ShortcutRepository, queryRepo QueryRepository) *LinkService {
	return &LinkService{
		shortcutRepo: shortcutRepo,
		queryRepo:    queryRepo,
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

	shortcut, err := s.shortcutRepo.GetByWord(ctx, word)
	if err != nil {
		return "", fmt.Errorf("failed to get shortcut: %w", err)
	}

	if shortcut == nil {
		// Try splitting the word if it contains spaces
		if strings.Contains(word, " ") {
			newWord, newSearchTerm := moveLastWord(word, searchTerm)
			return s.GetLink(ctx, newWord, newSearchTerm)
		}

		return "", InvalidQueryError{
			Message: fmt.Sprintf("Unable to find link for query %s", strings.Join([]string{word, searchTerm}, " ")),
		}
	}

	// Log the query
	if err := s.queryRepo.Create(ctx, shortcut.ID); err != nil {
		// Log error but don't fail the request
		// In a production system, you might want to log this error
		_ = err
	}

	// Handle different types of links
	if !isURL(shortcut.Link) {
		// This is an alias, recurse
		return s.GetLink(ctx, shortcut.Link, searchTerm)
	}

	// Process URL with search term substitution
	resultLink := processResultLink(shortcut.Link, searchTerm)
	return resultLink, nil
}

// UpdateLink creates or updates a golink
func (s *LinkService) UpdateLink(ctx context.Context, req domain.LinkRequest, userID string) error {

	// Validate the request
	if err := s.validateLinkRequest(ctx, req); err != nil {
		return err
	}

	// If the link is not a URL, validate it's a valid alias
	if !isURL(req.Link) {
		_, err := s.GetLink(ctx, req.Link, "")
		if err != nil {
			return InvalidQueryError{
				Message: "The link target appears to neither be a URL, or a valid alias.",
			}
		}
	}

	shortcut := &domain.Shortcut{
		Word:      req.Word,
		Link:      req.Link,
		User:      userID,
		CreatedAt: time.Now(),
	}

	if err := s.shortcutRepo.Create(ctx, shortcut); err != nil {
		return fmt.Errorf("failed to create shortcut: %w", err)
	}

	return nil
}

// GetRecentQueries retrieves popular queries
func (s *LinkService) GetRecentQueries(ctx context.Context) ([]domain.PopularQuery, error) {
	return s.queryRepo.GetRecentQueries(ctx, 3, 20)
}

// GetAllKeywords retrieves all keywords with aliases
func (s *LinkService) GetAllKeywords(ctx context.Context) ([]domain.KeywordInfo, error) {
	keywords, err := s.shortcutRepo.GetAllKeywords(ctx)
	if err != nil {
		return nil, err
	}

	// Process aliases (simplified version - not implementing full recursive alias resolution for now)
	for i := range keywords {
		if !isURL(keywords[i].Link) {
			keywords[i].Aliases = keywords[i].Link
		}
	}

	// Filter to only return URLs (not aliases)
	var result []domain.KeywordInfo
	for _, keyword := range keywords {
		if isURL(keyword.Link) {
			result = append(result, keyword)
		}
	}

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
