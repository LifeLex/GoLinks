package domain

import (
	"time"
)

// Shortcut represents a golink shortcut. Tags is populated on the write path
// and is not loaded by the redirect lookup, which has no need for it.
type Shortcut struct {
	ID        int       `json:"id" db:"id"`
	Word      string    `json:"word" db:"word"`
	Link      string    `json:"link" db:"link"`
	User      string    `json:"user" db:"user"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	Tags      []string  `json:"tags,omitempty" db:"-"`
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
	Word string   `json:"word" validate:"required"`
	Link string   `json:"link" validate:"required"`
	Tags []string `json:"tags,omitempty"`
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
	Tags      []string  `json:"tags"`
}

// Role enumerates the authorization levels a user can hold.
type Role string

const (
	// RoleAdmin can manage users and perform every write, including doc uploads.
	RoleAdmin Role = "admin"
	// RoleUser can perform standard writes but not user management or doc uploads.
	RoleUser Role = "user"
)

// User is an authenticated account. PasswordHash is never serialized to JSON.
type User struct {
	ID           int       `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Role         Role      `json:"role" db:"role"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// IsAdmin reports whether the user holds the admin role.
func (u *User) IsAdmin() bool {
	return u != nil && u.Role == RoleAdmin
}

// Session is a server-side login session. The raw token is never stored; only
// its SHA-256 hash (TokenHash) lives in the database, so a DB read cannot mint
// a valid cookie.
type Session struct {
	TokenHash string    `json:"-" db:"token_hash"`
	UserID    int       `json:"user_id" db:"user_id"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// LoginRequest is the body of POST /auth/login and POST /auth/setup.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// CreateUserRequest is the body of POST /api/users (admin only).
type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     Role   `json:"role"`
}

// AuthStatus is returned by GET /auth/status to drive the SPA's bootstrap:
// whether a first-run setup is needed and who (if anyone) is logged in.
type AuthStatus struct {
	NeedsSetup    bool  `json:"needs_setup"`
	Authenticated bool  `json:"authenticated"`
	User          *User `json:"user,omitempty"`
}
