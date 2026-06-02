package handlers

import (
	"context"
	"net/http"

	"golinks/internal/domain"
	"golinks/internal/logger"
)

// ctxKey is an unexported type for context keys defined in this package, so
// keys can't collide with those from other packages.
type ctxKey int

const userCtxKey ctxKey = iota

// Authenticator resolves a raw session token to its user (or nil if anonymous).
type Authenticator interface {
	Authenticate(ctx context.Context, rawToken string) (*domain.User, error)
}

// AuthMiddleware provides request authentication and authorization guards.
type AuthMiddleware struct {
	auth   Authenticator
	logger *logger.Logger
}

// NewAuthMiddleware creates auth middleware backed by the given authenticator.
func NewAuthMiddleware(auth Authenticator, log *logger.Logger) *AuthMiddleware {
	return &AuthMiddleware{auth: auth, logger: log}
}

// Authenticate loads the optional user from the session cookie into the request
// context. It never rejects: an absent or invalid cookie simply means the
// request is anonymous, so public routes keep working.
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
			user, err := m.auth.Authenticate(r.Context(), cookie.Value)
			if err != nil {
				m.logger.Error("Authenticate middleware lookup failed: %v", err)
			} else if user != nil {
				r = r.WithContext(WithUser(r.Context(), user))
			}
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAuth rejects anonymous requests with 401.
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if UserFromContext(r.Context()) == nil {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin rejects anonymous requests with 401 and non-admins with 403.
func (m *AuthMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		if !user.IsAdmin() {
			http.Error(w, "admin role required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// WithUser returns a copy of ctx carrying the authenticated user.
func WithUser(ctx context.Context, user *domain.User) context.Context {
	return context.WithValue(ctx, userCtxKey, user)
}

// UserFromContext returns the authenticated user, or nil if the request is
// anonymous.
func UserFromContext(ctx context.Context) *domain.User {
	user, _ := ctx.Value(userCtxKey).(*domain.User)
	return user
}
