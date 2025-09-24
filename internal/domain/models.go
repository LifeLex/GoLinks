package domain

import (
	"time"
)

// Shortcut represents a golink shortcut
type Shortcut struct {
	ID        int       `json:"id" db:"id"`
	Word      string    `json:"word" db:"word"`
	Link      string    `json:"link" db:"link"`
	User      string    `json:"user" db:"user"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Query represents a query log entry
type Query struct {
	ID        int       `json:"id" db:"query_id"`
	WordID    int       `json:"word_id" db:"word_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Tag represents a tag associated with a shortcut
type Tag struct {
	ID     int    `json:"id" db:"id"`
	WordID int    `json:"word_id" db:"word_id"`
	Tag    string `json:"tag" db:"tag"`
}

// LinkRequest represents a request to create or update a link
type LinkRequest struct {
	Word string `json:"word" validate:"required"`
	Link string `json:"link" validate:"required"`
}

// PopularQuery represents a popular query with count
type PopularQuery struct {
	Count int    `json:"count"`
	Word  string `json:"word"`
	Link  string `json:"link"`
}

// KeywordInfo represents keyword information
type KeywordInfo struct {
	Word      string    `json:"word"`
	Link      string    `json:"link"`
	CreatedAt time.Time `json:"created_at"`
}
