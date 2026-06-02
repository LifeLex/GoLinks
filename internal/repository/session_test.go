package repository

import (
	"context"
	"testing"
	"time"

	"golinks/internal/domain"
	"golinks/internal/logger"
)

func newSessionRepo(t *testing.T) (*SessionRepository, *UserRepository, func()) {
	t.Helper()
	db := setupTestDB(t)
	log := logger.New(logger.Config{Level: "error", Format: "text"})
	return NewSessionRepository(db, log), NewUserRepository(db, log), func() { db.Close() }
}

func TestSessionRepository_CreateGetDelete(t *testing.T) {
	sessions, users, cleanup := newSessionRepo(t)
	defer cleanup()
	ctx := context.Background()

	u := &domain.User{Email: "s@example.com", PasswordHash: "h", Role: domain.RoleUser}
	if err := users.Create(ctx, u); err != nil {
		t.Fatalf("user Create() error = %v", err)
	}

	s := &domain.Session{TokenHash: "hash-abc", UserID: u.ID, ExpiresAt: time.Now().Add(time.Hour)}
	if err := sessions.Create(ctx, s); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := sessions.GetByTokenHash(ctx, "hash-abc")
	if err != nil || got == nil || got.UserID != u.ID {
		t.Fatalf("GetByTokenHash() = %+v, err = %v", got, err)
	}

	if err := sessions.Delete(ctx, "hash-abc"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	got, _ = sessions.GetByTokenHash(ctx, "hash-abc")
	if got != nil {
		t.Errorf("GetByTokenHash() after delete = %+v, want nil", got)
	}

	// Idempotent delete.
	if err := sessions.Delete(ctx, "hash-abc"); err != nil {
		t.Errorf("second Delete() error = %v, want nil", err)
	}
}

func TestSessionRepository_DeleteExpired(t *testing.T) {
	sessions, users, cleanup := newSessionRepo(t)
	defer cleanup()
	ctx := context.Background()

	u := &domain.User{Email: "exp@example.com", PasswordHash: "h", Role: domain.RoleUser}
	if err := users.Create(ctx, u); err != nil {
		t.Fatalf("user Create() error = %v", err)
	}

	now := time.Now()
	expired := &domain.Session{TokenHash: "old", UserID: u.ID, ExpiresAt: now.Add(-time.Hour)}
	live := &domain.Session{TokenHash: "new", UserID: u.ID, ExpiresAt: now.Add(time.Hour)}
	for _, s := range []*domain.Session{expired, live} {
		if err := sessions.Create(ctx, s); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	n, err := sessions.DeleteExpired(ctx, now)
	if err != nil {
		t.Fatalf("DeleteExpired() error = %v", err)
	}
	if n != 1 {
		t.Errorf("DeleteExpired() removed %d, want 1", n)
	}
	if got, _ := sessions.GetByTokenHash(ctx, "new"); got == nil {
		t.Error("DeleteExpired() removed the live session")
	}
}

func TestSessionRepository_CascadeOnUserDelete(t *testing.T) {
	sessions, users, cleanup := newSessionRepo(t)
	defer cleanup()
	ctx := context.Background()

	u := &domain.User{Email: "casc@example.com", PasswordHash: "h", Role: domain.RoleUser}
	if err := users.Create(ctx, u); err != nil {
		t.Fatalf("user Create() error = %v", err)
	}
	if err := sessions.Create(ctx, &domain.Session{TokenHash: "c", UserID: u.ID, ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatalf("session Create() error = %v", err)
	}

	if err := users.Delete(ctx, u.ID); err != nil {
		t.Fatalf("user Delete() error = %v", err)
	}

	// The FK ON DELETE CASCADE should have removed the session.
	if got, _ := sessions.GetByTokenHash(ctx, "c"); got != nil {
		t.Errorf("session survived user delete: %+v", got)
	}
}
