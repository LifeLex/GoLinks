package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golinks/internal/config"
	"golinks/internal/domain"
	"golinks/internal/logger"

	"github.com/gorilla/mux"
)

// mockAuthService implements handlers.AuthService for endpoint tests.
type mockAuthService struct {
	needsSetup     bool
	bootstrapUser  *domain.User
	bootstrapToken string
	bootstrapErr   error
	loginUser      *domain.User
	loginToken     string
	loginErr       error
	users          []domain.User
	createUser     *domain.User
	createErr      error
	deleteErr      error
	loggedOutToken string
}

func (m *mockAuthService) NeedsSetup(context.Context) (bool, error) { return m.needsSetup, nil }

func (m *mockAuthService) Bootstrap(context.Context, domain.LoginRequest) (*domain.User, string, error) {
	return m.bootstrapUser, m.bootstrapToken, m.bootstrapErr
}

func (m *mockAuthService) Login(context.Context, domain.LoginRequest) (*domain.User, string, error) {
	return m.loginUser, m.loginToken, m.loginErr
}

func (m *mockAuthService) Logout(_ context.Context, token string) error {
	m.loggedOutToken = token
	return nil
}

func (m *mockAuthService) CreateUser(context.Context, domain.CreateUserRequest) (*domain.User, error) {
	return m.createUser, m.createErr
}

func (m *mockAuthService) ListUsers(context.Context) ([]domain.User, error) { return m.users, nil }

func (m *mockAuthService) DeleteUser(context.Context, int) error { return m.deleteErr }

func testAuthHandler(svc *mockAuthService) *AuthHandler {
	cfg := &config.Config{Auth: config.AuthConfig{SessionTTLHours: 1, CookieSecure: false}}
	return NewAuthHandler(svc, cfg, logger.New(logger.Config{Level: "error", Format: "text"}))
}

func TestAuthHandler_Status(t *testing.T) {
	h := testAuthHandler(&mockAuthService{needsSetup: true})
	req := httptest.NewRequest("GET", "/auth/status", nil)
	w := httptest.NewRecorder()
	h.Status(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp domain.AuthStatus
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.NeedsSetup || resp.Authenticated {
		t.Errorf("status = %+v, want needs_setup=true authenticated=false", resp)
	}
}

func TestAuthHandler_Setup_SetsCookie(t *testing.T) {
	h := testAuthHandler(&mockAuthService{
		bootstrapUser:  &domain.User{Email: "admin@example.com", Role: domain.RoleAdmin},
		bootstrapToken: "tok123",
	})
	req := httptest.NewRequest("POST", "/auth/setup", strings.NewReader(`{"email":"admin@example.com","password":"password123"}`))
	w := httptest.NewRecorder()
	h.Setup(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%q", w.Code, w.Body.String())
	}
	cookie := findCookie(w.Result().Cookies(), sessionCookieName)
	if cookie == nil || cookie.Value != "tok123" {
		t.Fatalf("session cookie = %+v, want value tok123", cookie)
	}
	if !cookie.HttpOnly {
		t.Error("session cookie must be HttpOnly")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Error("session cookie must be SameSite=Lax")
	}
}

func TestAuthHandler_Setup_RegistrationClosed(t *testing.T) {
	h := testAuthHandler(&mockAuthService{bootstrapErr: domain.ErrRegistrationClosed})
	req := httptest.NewRequest("POST", "/auth/setup", strings.NewReader(`{"email":"x@x.com","password":"password123"}`))
	w := httptest.NewRecorder()
	h.Setup(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

type denyLimiter struct{}

func (denyLimiter) Allow(string) bool { return false }

func TestAuthHandler_Login_RateLimited(t *testing.T) {
	h := testAuthHandler(&mockAuthService{loginUser: &domain.User{Email: "u@x.com"}, loginToken: "abc"})
	h.loginLimiter = denyLimiter{} // simulate exhausted limit

	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"email":"u@x.com","password":"password123"}`))
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", w.Code)
	}
}

func TestAuthHandler_Login(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := testAuthHandler(&mockAuthService{loginUser: &domain.User{Email: "u@x.com"}, loginToken: "abc"})
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"email":"u@x.com","password":"password123"}`))
		w := httptest.NewRecorder()
		h.Login(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if c := findCookie(w.Result().Cookies(), sessionCookieName); c == nil || c.Value != "abc" {
			t.Error("login did not set the session cookie")
		}
	})

	t.Run("invalid credentials", func(t *testing.T) {
		h := testAuthHandler(&mockAuthService{loginErr: domain.ErrInvalidCredentials})
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"email":"u@x.com","password":"nope"}`))
		w := httptest.NewRecorder()
		h.Login(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", w.Code)
		}
	})
}

func TestAuthHandler_Logout_ClearsCookie(t *testing.T) {
	svc := &mockAuthService{}
	h := testAuthHandler(svc)
	req := httptest.NewRequest("POST", "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "live-token"})
	w := httptest.NewRecorder()
	h.Logout(w, req)

	if svc.loggedOutToken != "live-token" {
		t.Errorf("service logout token = %q, want live-token", svc.loggedOutToken)
	}
	c := findCookie(w.Result().Cookies(), sessionCookieName)
	if c == nil || c.MaxAge >= 0 {
		t.Errorf("logout must expire the cookie, got %+v", c)
	}
}

// TestUserManagementGating wires /api/users through the real Authenticate +
// RequireAdmin middleware and verifies the access matrix.
func TestUserManagementGating(t *testing.T) {
	adminUser := &domain.User{ID: 1, Email: "admin@x.com", Role: domain.RoleAdmin}
	normalUser := &domain.User{ID: 2, Email: "user@x.com", Role: domain.RoleUser}
	mw := testMiddleware(map[string]*domain.User{"admintok": adminUser, "usertok": normalUser})
	h := testAuthHandler(&mockAuthService{users: []domain.User{*adminUser}})

	router := mux.NewRouter()
	router.Use(mw.Authenticate)
	admin := router.NewRoute().Subrouter()
	admin.Use(mw.RequireAdmin)
	h.RegisterRoutes(router, admin)

	tests := []struct {
		name  string
		token string
		want  int
	}{
		{"anonymous", "", http.StatusUnauthorized},
		{"non-admin", "usertok", http.StatusForbidden},
		{"admin", "admintok", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/users", nil)
			if tt.token != "" {
				req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: tt.token})
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tt.want {
				t.Errorf("GET /api/users (%s) = %d, want %d", tt.name, w.Code, tt.want)
			}
		})
	}
}

func TestAuthHandler_DeleteUser_LastAdmin(t *testing.T) {
	h := testAuthHandler(&mockAuthService{deleteErr: domain.ErrLastAdmin})
	router := mux.NewRouter()
	// Register on a plain router (gating tested separately); exercise the handler.
	router.HandleFunc("/api/users/{id}", h.DeleteUser).Methods("DELETE")

	req := httptest.NewRequest("DELETE", "/api/users/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", w.Code)
	}
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}
