package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"golinks/internal/domain"
	"golinks/internal/logger"

	"golang.org/x/crypto/bcrypt"
)

// UserRepo is the subset of user persistence the AuthService depends on.
type UserRepo interface {
	CountUsers(ctx context.Context) (int, error)
	CountAdmins(ctx context.Context) (int, error)
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id int) (*domain.User, error)
	List(ctx context.Context) ([]domain.User, error)
	Delete(ctx context.Context, id int) error
}

// SessionRepo is the subset of session persistence the AuthService depends on.
type SessionRepo interface {
	Create(ctx context.Context, s *domain.Session) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error)
	Delete(ctx context.Context, tokenHash string) error
	DeleteByUserID(ctx context.Context, userID int) error
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}

// bcryptMaxPasswordBytes is the hard input limit bcrypt silently truncates at.
const bcryptMaxPasswordBytes = 72

// AuthService handles authentication, sessions, and user management.
type AuthService struct {
	users      UserRepo
	sessions   SessionRepo
	logger     *logger.Logger
	sessionTTL time.Duration
	bcryptCost int
	minPwLen   int
	// dummyHash is compared against during failed logins so an unknown email
	// takes the same time as a wrong password (mitigates user enumeration).
	dummyHash []byte
}

// NewAuthService creates a new auth service. sessionTTL, bcryptCost, and
// minPwLen come from config.
func NewAuthService(
	users UserRepo,
	sessions SessionRepo,
	log *logger.Logger,
	sessionTTL time.Duration,
	bcryptCost, minPwLen int,
) *AuthService {
	dummy, err := bcrypt.GenerateFromPassword([]byte("timing-equalizer"), bcryptCost)
	if err != nil {
		// bcrypt only errors on an out-of-range cost; fall back to the default.
		dummy, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer"), bcrypt.DefaultCost)
	}
	log.Info("Auth service initialized")
	return &AuthService{
		users:      users,
		sessions:   sessions,
		logger:     log,
		sessionTTL: sessionTTL,
		bcryptCost: bcryptCost,
		minPwLen:   minPwLen,
		dummyHash:  dummy,
	}
}

// NeedsSetup reports whether the instance has no users yet (first-run state).
func (s *AuthService) NeedsSetup(ctx context.Context) (bool, error) {
	n, err := s.users.CountUsers(ctx)
	if err != nil {
		return false, err
	}
	return n == 0, nil
}

// Bootstrap creates the very first user as an admin and opens a session. It
// fails with domain.ErrRegistrationClosed if any user already exists — there is
// no open self-registration; subsequent users are created by an admin.
func (s *AuthService) Bootstrap(ctx context.Context, req domain.LoginRequest) (*domain.User, string, error) {
	n, err := s.users.CountUsers(ctx)
	if err != nil {
		return nil, "", err
	}
	if n > 0 {
		return nil, "", domain.ErrRegistrationClosed
	}
	return s.createUserWithSession(ctx, req.Email, req.Password, domain.RoleAdmin)
}

// Login verifies credentials and opens a session, returning the raw session
// token. The same domain.ErrInvalidCredentials is returned for both unknown
// emails and wrong passwords.
func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest) (*domain.User, string, error) {
	email := normalizeEmail(req.Email)
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		// Equalize timing against the wrong-password path.
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(req.Password))
		return nil, "", domain.ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, "", domain.ErrInvalidCredentials
	}

	token, err := s.openSession(ctx, user.ID)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

// Authenticate resolves a raw session token to its user. A missing, unknown, or
// expired token yields (nil, nil) — i.e. anonymous, not an error. Expired
// sessions are deleted opportunistically.
func (s *AuthService) Authenticate(ctx context.Context, rawToken string) (*domain.User, error) {
	if rawToken == "" {
		return nil, nil
	}
	tokenHash := hashToken(rawToken)
	session, err := s.sessions.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}
	if !session.ExpiresAt.After(time.Now()) {
		_ = s.sessions.Delete(ctx, tokenHash)
		return nil, nil
	}
	return s.users.GetByID(ctx, session.UserID)
}

// Logout deletes the session for the given raw token. It is idempotent.
func (s *AuthService) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	return s.sessions.Delete(ctx, hashToken(rawToken))
}

// CreateUser creates a non-bootstrap user (admin action). An empty role
// defaults to RoleUser.
func (s *AuthService) CreateUser(ctx context.Context, req domain.CreateUserRequest) (*domain.User, error) {
	role := req.Role
	if role == "" {
		role = domain.RoleUser
	}
	if role != domain.RoleAdmin && role != domain.RoleUser {
		return nil, fmt.Errorf("invalid role %q", role)
	}
	user, _, err := s.createUser(ctx, req.Email, req.Password, role)
	return user, err
}

// ListUsers returns all users.
func (s *AuthService) ListUsers(ctx context.Context) ([]domain.User, error) {
	return s.users.List(ctx)
}

// DeleteUser removes a user, refusing to remove the last remaining admin.
func (s *AuthService) DeleteUser(ctx context.Context, id int) error {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return domain.ErrUserNotFound
	}
	if user.IsAdmin() {
		admins, err := s.users.CountAdmins(ctx)
		if err != nil {
			return err
		}
		if admins <= 1 {
			return domain.ErrLastAdmin
		}
	}
	return s.users.Delete(ctx, id)
}

// PurgeExpiredSessions deletes expired sessions; called periodically.
func (s *AuthService) PurgeExpiredSessions(ctx context.Context) (int64, error) {
	return s.sessions.DeleteExpired(ctx, time.Now())
}

// createUserWithSession creates a user and immediately opens a session for them.
func (s *AuthService) createUserWithSession(ctx context.Context, email, password string, role domain.Role) (*domain.User, string, error) {
	user, _, err := s.createUser(ctx, email, password, role)
	if err != nil {
		return nil, "", err
	}
	token, err := s.openSession(ctx, user.ID)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

// createUser validates input, hashes the password, and persists the user.
func (s *AuthService) createUser(ctx context.Context, email, password string, role domain.Role) (*domain.User, string, error) {
	email = normalizeEmail(email)
	if email == "" {
		return nil, "", domain.ErrInvalidCredentials
	}
	if err := s.validatePassword(password); err != nil {
		return nil, "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	user := &domain.User{Email: email, PasswordHash: string(hash), Role: role}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, "", err
	}
	return user, "", nil
}

// openSession mints a fresh random token, stores its hash, and returns the raw
// token for the cookie. A fresh token on every login prevents session fixation.
func (s *AuthService) openSession(ctx context.Context, userID int) (string, error) {
	token, tokenHash, err := generateToken()
	if err != nil {
		return "", err
	}
	session := &domain.Session{
		TokenHash: tokenHash,
		UserID:    userID,
		ExpiresAt: time.Now().Add(s.sessionTTL),
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return "", err
	}
	return token, nil
}

// validatePassword enforces the length policy. Passwords are never trimmed.
func (s *AuthService) validatePassword(password string) error {
	if len(password) < s.minPwLen || len(password) > bcryptMaxPasswordBytes {
		return domain.ErrWeakPassword
	}
	return nil
}

// normalizeEmail lower-cases and trims an email for storage and comparison.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// generateToken returns a random URL-safe token and its SHA-256 hex hash.
func generateToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("failed to generate token: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, hashToken(raw), nil
}

// hashToken returns the hex-encoded SHA-256 of a raw token.
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
