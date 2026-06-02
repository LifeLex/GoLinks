package repository

import (
	"context"
	"errors"
	"testing"

	"golinks/internal/domain"
	"golinks/internal/logger"
)

func newUserRepo(t *testing.T) (*UserRepository, func()) {
	t.Helper()
	db := setupTestDB(t)
	repo := NewUserRepository(db, logger.New(logger.Config{Level: "error", Format: "text"}))
	return repo, func() { db.Close() }
}

func TestUserRepository_CreateAndGet(t *testing.T) {
	repo, cleanup := newUserRepo(t)
	defer cleanup()
	ctx := context.Background()

	u := &domain.User{Email: "Admin@Example.com", PasswordHash: "hash", Role: domain.RoleAdmin}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if u.ID == 0 {
		t.Error("Create() did not set ID")
	}

	// Lookup is case-insensitive.
	got, err := repo.GetByEmail(ctx, "admin@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}
	if got == nil || got.Email != "Admin@Example.com" || got.Role != domain.RoleAdmin {
		t.Errorf("GetByEmail() = %+v, want the admin user", got)
	}

	byID, err := repo.GetByID(ctx, u.ID)
	if err != nil || byID == nil || byID.ID != u.ID {
		t.Errorf("GetByID() = %+v, err = %v", byID, err)
	}
}

func TestUserRepository_GetByEmail_NotFound(t *testing.T) {
	repo, cleanup := newUserRepo(t)
	defer cleanup()

	got, err := repo.GetByEmail(context.Background(), "nobody@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}
	if got != nil {
		t.Errorf("GetByEmail() = %+v, want nil", got)
	}
}

func TestUserRepository_DuplicateEmail(t *testing.T) {
	repo, cleanup := newUserRepo(t)
	defer cleanup()
	ctx := context.Background()

	first := &domain.User{Email: "dup@example.com", PasswordHash: "h", Role: domain.RoleUser}
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	// Same email in different case must still collide.
	dup := &domain.User{Email: "DUP@example.com", PasswordHash: "h", Role: domain.RoleUser}
	err := repo.Create(ctx, dup)
	if !errors.Is(err, domain.ErrEmailTaken) {
		t.Errorf("Create() error = %v, want ErrEmailTaken", err)
	}
}

func TestUserRepository_CountsAndDelete(t *testing.T) {
	repo, cleanup := newUserRepo(t)
	defer cleanup()
	ctx := context.Background()

	admin := &domain.User{Email: "a@example.com", PasswordHash: "h", Role: domain.RoleAdmin}
	user := &domain.User{Email: "u@example.com", PasswordHash: "h", Role: domain.RoleUser}
	for _, u := range []*domain.User{admin, user} {
		if err := repo.Create(ctx, u); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	if n, _ := repo.CountUsers(ctx); n != 2 {
		t.Errorf("CountUsers() = %d, want 2", n)
	}
	if n, _ := repo.CountAdmins(ctx); n != 1 {
		t.Errorf("CountAdmins() = %d, want 1", n)
	}

	list, err := repo.List(ctx)
	if err != nil || len(list) != 2 {
		t.Fatalf("List() = %d users, err = %v", len(list), err)
	}

	if err := repo.Delete(ctx, user.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if n, _ := repo.CountUsers(ctx); n != 1 {
		t.Errorf("CountUsers() after delete = %d, want 1", n)
	}
}
