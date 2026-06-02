package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"golinks/internal/domain"
	"golinks/internal/logger"
)

// --- mocks ---

type mockUserRepo struct {
	users  []*domain.User
	nextID int
}

func (m *mockUserRepo) CountUsers(_ context.Context) (int, error) { return len(m.users), nil }

func (m *mockUserRepo) CountAdmins(_ context.Context) (int, error) {
	n := 0
	for _, u := range m.users {
		if u.Role == domain.RoleAdmin {
			n++
		}
	}
	return n, nil
}

func (m *mockUserRepo) Create(_ context.Context, u *domain.User) error {
	for _, e := range m.users {
		if strings.EqualFold(e.Email, u.Email) {
			return domain.ErrEmailTaken
		}
	}
	m.nextID++
	u.ID = m.nextID
	clone := *u
	m.users = append(m.users, &clone)
	return nil
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, u := range m.users {
		if strings.EqualFold(u.Email, email) {
			clone := *u
			return &clone, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id int) (*domain.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			clone := *u
			return &clone, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) List(_ context.Context) ([]domain.User, error) {
	out := make([]domain.User, 0, len(m.users))
	for _, u := range m.users {
		out = append(out, *u)
	}
	return out, nil
}

func (m *mockUserRepo) Delete(_ context.Context, id int) error {
	for i, u := range m.users {
		if u.ID == id {
			m.users = append(m.users[:i], m.users[i+1:]...)
			return nil
		}
	}
	return nil
}

type mockSessionRepo struct {
	sessions map[string]*domain.Session
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{sessions: map[string]*domain.Session{}}
}

func (m *mockSessionRepo) Create(_ context.Context, s *domain.Session) error {
	clone := *s
	m.sessions[s.TokenHash] = &clone
	return nil
}

func (m *mockSessionRepo) GetByTokenHash(_ context.Context, h string) (*domain.Session, error) {
	if s, ok := m.sessions[h]; ok {
		clone := *s
		return &clone, nil
	}
	return nil, nil
}

func (m *mockSessionRepo) Delete(_ context.Context, h string) error {
	delete(m.sessions, h)
	return nil
}

func (m *mockSessionRepo) DeleteByUserID(_ context.Context, userID int) error {
	for h, s := range m.sessions {
		if s.UserID == userID {
			delete(m.sessions, h)
		}
	}
	return nil
}

func (m *mockSessionRepo) DeleteExpired(_ context.Context, now time.Time) (int64, error) {
	var n int64
	for h, s := range m.sessions {
		if !s.ExpiresAt.After(now) {
			delete(m.sessions, h)
			n++
		}
	}
	return n, nil
}

func newAuthService() (*AuthService, *mockUserRepo, *mockSessionRepo) {
	users := &mockUserRepo{}
	sessions := newMockSessionRepo()
	log := logger.New(logger.Config{Level: "error", Format: "text"})
	// bcrypt cost 4 keeps tests fast.
	return NewAuthService(users, sessions, log, time.Hour, 4, 8), users, sessions
}

// --- tests ---

func TestAuthService_Bootstrap(t *testing.T) {
	svc, users, sessions := newAuthService()
	ctx := context.Background()

	user, token, err := svc.Bootstrap(ctx, domain.LoginRequest{Email: "Admin@Example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if user.Role != domain.RoleAdmin {
		t.Errorf("first user role = %q, want admin", user.Role)
	}
	if user.Email != "admin@example.com" {
		t.Errorf("email not normalized: %q", user.Email)
	}
	// The session is keyed by the token HASH, never the raw token.
	if _, ok := sessions.sessions[token]; ok {
		t.Error("raw token stored as session key — must store the hash")
	}
	if _, ok := sessions.sessions[hashToken(token)]; !ok {
		t.Error("session not stored under token hash")
	}

	// Second bootstrap is refused.
	_, _, err = svc.Bootstrap(ctx, domain.LoginRequest{Email: "two@example.com", Password: "password123"})
	if !errors.Is(err, domain.ErrRegistrationClosed) {
		t.Errorf("second Bootstrap() error = %v, want ErrRegistrationClosed", err)
	}
	if len(users.users) != 1 {
		t.Errorf("user count = %d, want 1", len(users.users))
	}
}

func TestAuthService_Login(t *testing.T) {
	svc, _, _ := newAuthService()
	ctx := context.Background()
	if _, _, err := svc.Bootstrap(ctx, domain.LoginRequest{Email: "a@example.com", Password: "password123"}); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	tests := []struct {
		name    string
		email   string
		pass    string
		wantErr bool
	}{
		{"correct", "a@example.com", "password123", false},
		{"correct case-insensitive email", "A@EXAMPLE.COM", "password123", false},
		{"wrong password", "a@example.com", "wrongpass1", true},
		{"unknown email", "nobody@example.com", "password123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, token, err := svc.Login(ctx, domain.LoginRequest{Email: tt.email, Password: tt.pass})
			if tt.wantErr {
				if !errors.Is(err, domain.ErrInvalidCredentials) {
					t.Errorf("Login() error = %v, want ErrInvalidCredentials", err)
				}
				return
			}
			if err != nil || user == nil || token == "" {
				t.Errorf("Login() = (%v, %q, %v), want a user + token", user, token, err)
			}
		})
	}
}

func TestAuthService_Authenticate(t *testing.T) {
	svc, _, sessions := newAuthService()
	ctx := context.Background()
	user, token, err := svc.Bootstrap(ctx, domain.LoginRequest{Email: "a@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Valid token resolves to the user.
	got, err := svc.Authenticate(ctx, token)
	if err != nil || got == nil || got.ID != user.ID {
		t.Fatalf("Authenticate(valid) = (%v, %v), want the user", got, err)
	}

	// Unknown token → anonymous, no error.
	if got, err := svc.Authenticate(ctx, "garbage"); err != nil || got != nil {
		t.Errorf("Authenticate(garbage) = (%v, %v), want (nil, nil)", got, err)
	}

	// Empty token → anonymous.
	if got, _ := svc.Authenticate(ctx, ""); got != nil {
		t.Errorf("Authenticate(empty) = %v, want nil", got)
	}

	// Expired session → anonymous, and the row is purged.
	rawExpired := "expired-raw-token"
	sessions.sessions[hashToken(rawExpired)] = &domain.Session{
		TokenHash: hashToken(rawExpired), UserID: user.ID, ExpiresAt: time.Now().Add(-time.Minute),
	}
	if got, _ := svc.Authenticate(ctx, rawExpired); got != nil {
		t.Errorf("Authenticate(expired) = %v, want nil", got)
	}
	if _, ok := sessions.sessions[hashToken(rawExpired)]; ok {
		t.Error("expired session was not purged on lookup")
	}
}

func TestAuthService_Logout(t *testing.T) {
	svc, _, sessions := newAuthService()
	ctx := context.Background()
	_, token, _ := svc.Bootstrap(ctx, domain.LoginRequest{Email: "a@example.com", Password: "password123"})

	if err := svc.Logout(ctx, token); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if len(sessions.sessions) != 0 {
		t.Error("Logout() did not delete the session")
	}
	// Idempotent.
	if err := svc.Logout(ctx, token); err != nil {
		t.Errorf("second Logout() error = %v", err)
	}
}

func TestAuthService_CreateUser(t *testing.T) {
	svc, _, _ := newAuthService()
	ctx := context.Background()

	tests := []struct {
		name     string
		req      domain.CreateUserRequest
		wantErr  error
		wantRole domain.Role
	}{
		{
			"valid default role",
			domain.CreateUserRequest{Email: "u1@example.com", Password: "password123"},
			nil, domain.RoleUser,
		},
		{
			"valid admin role",
			domain.CreateUserRequest{Email: "u2@example.com", Password: "password123", Role: domain.RoleAdmin},
			nil, domain.RoleAdmin,
		},
		{
			"weak password",
			domain.CreateUserRequest{Email: "u3@example.com", Password: "short"},
			domain.ErrWeakPassword, "",
		},
		{
			"duplicate",
			domain.CreateUserRequest{Email: "u1@example.com", Password: "password123"},
			domain.ErrEmailTaken, "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := svc.CreateUser(ctx, tt.req)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("CreateUser() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateUser() error = %v", err)
			}
			if user.Role != tt.wantRole {
				t.Errorf("role = %q, want %q", user.Role, tt.wantRole)
			}
		})
	}
}

func TestAuthService_DeleteUser_LastAdmin(t *testing.T) {
	svc, _, _ := newAuthService()
	ctx := context.Background()
	admin, _, _ := svc.Bootstrap(ctx, domain.LoginRequest{Email: "admin@example.com", Password: "password123"})

	// Deleting the sole admin is refused.
	if err := svc.DeleteUser(ctx, admin.ID); !errors.Is(err, domain.ErrLastAdmin) {
		t.Errorf("DeleteUser(last admin) error = %v, want ErrLastAdmin", err)
	}

	// Add a second admin; now the first can be removed.
	second, err := svc.CreateUser(ctx, domain.CreateUserRequest{Email: "admin2@example.com", Password: "password123", Role: domain.RoleAdmin})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if err := svc.DeleteUser(ctx, admin.ID); err != nil {
		t.Errorf("DeleteUser(one of two admins) error = %v", err)
	}

	// Deleting a non-existent user → ErrUserNotFound.
	if err := svc.DeleteUser(ctx, 9999); !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("DeleteUser(missing) error = %v, want ErrUserNotFound", err)
	}
	_ = second
}
