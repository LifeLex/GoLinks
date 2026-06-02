package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"golinks/internal/config"
	"golinks/internal/domain"
	"golinks/internal/logger"

	"github.com/gorilla/mux"
)

// sessionCookieName is the name of the HttpOnly session cookie. Shared with the
// auth middleware in the same package.
const sessionCookieName = "golinks_session"

// AuthService is the subset of the auth service the HTTP layer depends on.
type AuthService interface {
	NeedsSetup(ctx context.Context) (bool, error)
	Bootstrap(ctx context.Context, req domain.LoginRequest) (*domain.User, string, error)
	Login(ctx context.Context, req domain.LoginRequest) (*domain.User, string, error)
	Logout(ctx context.Context, rawToken string) error
	CreateUser(ctx context.Context, req domain.CreateUserRequest) (*domain.User, error)
	ListUsers(ctx context.Context) ([]domain.User, error)
	DeleteUser(ctx context.Context, id int) error
}

// AuthHandler owns the /auth/* and /api/users endpoints.
type AuthHandler struct {
	auth   AuthService
	config *config.Config
	logger *logger.Logger
}

// NewAuthHandler builds a new auth handler.
func NewAuthHandler(auth AuthService, cfg *config.Config, log *logger.Logger) *AuthHandler {
	log.Info("Auth handler initialized")
	return &AuthHandler{auth: auth, config: cfg, logger: log}
}

// RegisterRoutes wires auth endpoints. Public auth endpoints go on public; user
// management goes on the admin-gated router.
func (h *AuthHandler) RegisterRoutes(public, admin *mux.Router) {
	public.HandleFunc("/auth/status", h.Status).Methods("GET")
	public.HandleFunc("/auth/me", h.Me).Methods("GET")
	public.HandleFunc("/auth/setup", h.Setup).Methods("POST")
	public.HandleFunc("/auth/login", h.Login).Methods("POST")
	public.HandleFunc("/auth/logout", h.Logout).Methods("POST")

	admin.HandleFunc("/api/users", h.ListUsers).Methods("GET")
	admin.HandleFunc("/api/users", h.CreateUser).Methods("POST")
	admin.HandleFunc("/api/users/{id}", h.DeleteUser).Methods("DELETE")
}

// Status reports first-run setup state and the current user (bootstrap probe).
func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	needsSetup, err := h.auth.NeedsSetup(r.Context())
	if err != nil {
		h.logger.Error("NeedsSetup failed: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	user := UserFromContext(r.Context())
	writeJSON(w, http.StatusOK, domain.AuthStatus{
		NeedsSetup:    needsSetup,
		Authenticated: user != nil,
		User:          user,
	})
}

// Me returns the current user, or {authenticated:false} when anonymous.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"authenticated": user != nil,
		"user":          user,
	})
}

// Setup creates the first (admin) user on a fresh instance and logs them in.
func (h *AuthHandler) Setup(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	user, token, err := h.auth.Bootstrap(r.Context(), req)
	if err != nil {
		h.writeAuthError(w, "setup", err)
		return
	}
	h.setSessionCookie(w, token)
	writeJSON(w, http.StatusOK, map[string]interface{}{"user": user})
}

// Login verifies credentials and opens a session.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	user, token, err := h.auth.Login(r.Context(), req)
	if err != nil {
		h.writeAuthError(w, "login", err)
		return
	}
	h.setSessionCookie(w, token)
	writeJSON(w, http.StatusOK, map[string]interface{}{"user": user})
}

// Logout clears the session.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		if err := h.auth.Logout(r.Context(), cookie.Value); err != nil {
			h.logger.Error("Logout failed: %v", err)
		}
	}
	h.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

// ListUsers returns all users (admin only).
func (h *AuthHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.auth.ListUsers(r.Context())
	if err != nil {
		h.logger.Error("ListUsers failed: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"users": users})
}

// CreateUser creates a new user (admin only).
func (h *AuthHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateUserRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	user, err := h.auth.CreateUser(r.Context(), req)
	if err != nil {
		h.writeAuthError(w, "create user", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"user": user})
}

// DeleteUser removes a user by id (admin only).
func (h *AuthHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}
	if err := h.auth.DeleteUser(r.Context(), id); err != nil {
		h.writeAuthError(w, "delete user", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

// writeAuthError maps domain sentinel errors to HTTP status codes without
// leaking detail; anything unrecognized is a 500.
func (h *AuthHandler) writeAuthError(w http.ResponseWriter, op string, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		http.Error(w, err.Error(), http.StatusUnauthorized)
	case errors.Is(err, domain.ErrRegistrationClosed):
		http.Error(w, err.Error(), http.StatusForbidden)
	case errors.Is(err, domain.ErrEmailTaken), errors.Is(err, domain.ErrLastAdmin):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, domain.ErrWeakPassword):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, domain.ErrUserNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		h.logger.Error("%s failed: %v", op, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.config.Auth.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   h.config.Auth.SessionTTLHours * 3600,
	})
}

func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.config.Auth.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// decodeJSON decodes a JSON request body, writing a 400 and returning false on
// failure.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return false
	}
	return true
}
