package domain

import "errors"

// Auth-related sentinel errors. Handlers map these to HTTP status codes without
// leaking which part of a credential was wrong (see internal/handlers/auth.go).
var (
	// ErrEmailTaken is returned when creating a user whose email already exists.
	ErrEmailTaken = errors.New("email already registered")
	// ErrInvalidCredentials is returned for any failed login — unknown email or
	// wrong password are deliberately indistinguishable to the caller.
	ErrInvalidCredentials = errors.New("invalid email or password")
	// ErrRegistrationClosed is returned when self-registration is attempted after
	// the first (bootstrap) user already exists.
	ErrRegistrationClosed = errors.New("registration is closed")
	// ErrLastAdmin is returned when deleting a user would leave no admins.
	ErrLastAdmin = errors.New("cannot remove the last admin")
	// ErrWeakPassword is returned when a password fails the length policy.
	ErrWeakPassword = errors.New("password does not meet requirements")
	// ErrUserNotFound is returned when a user lookup by id finds nothing.
	ErrUserNotFound = errors.New("user not found")
)
