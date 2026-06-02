package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golinks/internal/domain"
	"golinks/internal/logger"

	"github.com/gorilla/mux"
)

// mockAuthenticator resolves session tokens to users for middleware tests.
type mockAuthenticator struct {
	tokens map[string]*domain.User
}

func (m *mockAuthenticator) Authenticate(_ context.Context, raw string) (*domain.User, error) {
	return m.tokens[raw], nil
}

func testMiddleware(tokens map[string]*domain.User) *AuthMiddleware {
	return NewAuthMiddleware(&mockAuthenticator{tokens: tokens}, logger.New(logger.Config{Level: "error", Format: "text"}))
}

// okHandler is a trivial next-handler that records whether it ran.
func okHandler(ran *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*ran = true
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequireAuth(t *testing.T) {
	mw := testMiddleware(nil)
	tests := []struct {
		name       string
		user       *domain.User
		wantStatus int
		wantRan    bool
	}{
		{"anonymous", nil, http.StatusUnauthorized, false},
		{"authenticated", &domain.User{Email: "u@x.com", Role: domain.RoleUser}, http.StatusOK, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ran := false
			req := httptest.NewRequest("POST", "/api/links", nil)
			if tt.user != nil {
				req = req.WithContext(WithUser(req.Context(), tt.user))
			}
			w := httptest.NewRecorder()
			mw.RequireAuth(okHandler(&ran)).ServeHTTP(w, req)
			if w.Code != tt.wantStatus || ran != tt.wantRan {
				t.Errorf("status=%d ran=%v, want status=%d ran=%v", w.Code, ran, tt.wantStatus, tt.wantRan)
			}
		})
	}
}

func TestRequireAdmin(t *testing.T) {
	mw := testMiddleware(nil)
	tests := []struct {
		name       string
		user       *domain.User
		wantStatus int
	}{
		{"anonymous", nil, http.StatusUnauthorized},
		{"non-admin", &domain.User{Email: "u@x.com", Role: domain.RoleUser}, http.StatusForbidden},
		{"admin", &domain.User{Email: "a@x.com", Role: domain.RoleAdmin}, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ran := false
			req := httptest.NewRequest("POST", "/api/users", nil)
			if tt.user != nil {
				req = req.WithContext(WithUser(req.Context(), tt.user))
			}
			w := httptest.NewRecorder()
			mw.RequireAdmin(okHandler(&ran)).ServeHTTP(w, req)
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestAuthenticate_LoadsUserFromCookie(t *testing.T) {
	user := &domain.User{ID: 1, Email: "u@x.com", Role: domain.RoleUser}
	mw := testMiddleware(map[string]*domain.User{"good-token": user})

	var seen *domain.User
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = UserFromContext(r.Context())
	})

	t.Run("valid cookie", func(t *testing.T) {
		seen = nil
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "good-token"})
		mw.Authenticate(next).ServeHTTP(httptest.NewRecorder(), req)
		if seen == nil || seen.Email != "u@x.com" {
			t.Errorf("user not loaded into context: %+v", seen)
		}
	})

	t.Run("unknown cookie stays anonymous", func(t *testing.T) {
		seen = nil
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "bogus"})
		mw.Authenticate(next).ServeHTTP(httptest.NewRecorder(), req)
		if seen != nil {
			t.Errorf("expected anonymous, got %+v", seen)
		}
	})

	t.Run("no cookie stays anonymous", func(t *testing.T) {
		seen = nil
		req := httptest.NewRequest("GET", "/", nil)
		mw.Authenticate(next).ServeHTTP(httptest.NewRecorder(), req)
		if seen != nil {
			t.Errorf("expected anonymous, got %+v", seen)
		}
	})
}

// TestLinkGating wires the real links Handler the way main.go does — global
// Authenticate + a RequireAuth subrouter — and verifies the public/authed split
// plus author propagation.
func TestLinkGating(t *testing.T) {
	admin := &domain.User{ID: 1, Email: "admin@example.com", Role: domain.RoleAdmin}
	mw := testMiddleware(map[string]*domain.User{"good": admin})
	handler := setupTestHandler()

	router := mux.NewRouter()
	router.Use(mw.Authenticate)
	authed := router.NewRoute().Subrouter()
	authed.Use(mw.RequireAuth)
	handler.RegisterRoutes(router, authed)

	do := func(method, path, body, token string) *httptest.ResponseRecorder {
		var r *http.Request
		if body != "" {
			r = httptest.NewRequest(method, path, strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
		} else {
			r = httptest.NewRequest(method, path, nil)
		}
		if token != "" {
			r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}

	// Public reads work anonymously.
	if w := do("GET", "/api/links", "", ""); w.Code != http.StatusOK {
		t.Errorf("GET /api/links anonymous = %d, want 200", w.Code)
	}
	if w := do("GET", "/api/search?q=x", "", ""); w.Code != http.StatusOK {
		t.Errorf("GET /api/search anonymous = %d, want 200", w.Code)
	}

	// Write is rejected when anonymous.
	if w := do("POST", "/api/links", `{"word":"k","link":"https://k.com"}`, ""); w.Code != http.StatusUnauthorized {
		t.Errorf("POST /api/links anonymous = %d, want 401", w.Code)
	}

	// Write succeeds with a session, and the author is the logged-in user.
	if w := do("POST", "/api/links", `{"word":"k","link":"https://k.com"}`, "good"); w.Code != http.StatusOK {
		t.Fatalf("POST /api/links authed = %d, want 200, body=%q", w.Code, w.Body.String())
	}
	if got := handler.linkService.(*mockLinkService).lastUpdateUser; got != "admin@example.com" {
		t.Errorf("link author = %q, want admin@example.com", got)
	}
}
