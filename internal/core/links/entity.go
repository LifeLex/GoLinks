package links

import "time"

// Shortcut is a stored golink: a keyword that resolves to a URL (or to another keyword).
type Shortcut struct {
	ID        int       `json:"id"`
	Word      string    `json:"word"`
	Link      string    `json:"link"`
	User      string    `json:"user"`
	CreatedAt time.Time `json:"created_at"`
}

// LinkRequest is the inbound payload for creating or updating a golink.
type LinkRequest struct {
	Word string `json:"word" validate:"required"`
	Link string `json:"link" validate:"required"`
}

// PopularQuery is an aggregate of how many times a keyword has been resolved over a recent window.
type PopularQuery struct {
	Count int    `json:"count"`
	Word  string `json:"word"`
	Link  string `json:"link"`
}

// KeywordInfo is a thin projection of Shortcut used by the homepage listing.
type KeywordInfo struct {
	Word      string    `json:"word"`
	Link      string    `json:"link"`
	CreatedAt time.Time `json:"created_at"`
}
