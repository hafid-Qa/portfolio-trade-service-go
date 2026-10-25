package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"app/internal/domain"
)

// seedTestUser creates a user via the repo under test (not raw SQL), with a
// username/email derived from the test name so parallel tests in this file
// never collide on the users table's UNIQUE constraints.
func seedTestUser(t *testing.T) domain.UserResult {
	t.Helper()
	repo := NewUserRepo(testStore)
	handle := strings.ToLower(strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()))
	user, err := repo.Create(context.Background(), "Test User", handle, handle+"@example.com", "hashed-password")
	if err != nil {
		t.Fatalf("seeding user: %v", err)
	}
	t.Cleanup(func() {
		testStore.DeleteUser(context.Background(), int32(user.ID()))
	})
	return user
}

func TestUserRepo_Create(t *testing.T) {
	user := seedTestUser(t)

	if user.ID() == 0 {
		t.Error("Create() ID() = 0, want a real assigned ID")
	}
	if user.FullName() != "Test User" {
		t.Errorf("FullName() = %q, want %q", user.FullName(), "Test User")
	}
	if user.PasswordHash() != "hashed-password" {
		t.Errorf("PasswordHash() = %q, want %q", user.PasswordHash(), "hashed-password")
	}
	if user.LastLogin() != nil {
		t.Errorf("LastLogin() = %v, want nil (never logged in)", user.LastLogin())
	}
	if user.CreatedAt().IsZero() {
		t.Error("CreatedAt() is zero, want a real timestamp")
	}
}

func TestUserRepo_Create_LowercasesUsernameAndEmail(t *testing.T) {
	repo := NewUserRepo(testStore)
	handle := strings.ToLower(strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()))

	user, err := repo.Create(context.Background(), "Test User", strings.ToUpper(handle), strings.ToUpper(handle)+"@EXAMPLE.COM", "x")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() {
		testStore.DeleteUser(context.Background(), int32(user.ID()))
	})

	if user.UserName() != handle {
		t.Errorf("UserName() = %q, want lowercased %q", user.UserName(), handle)
	}
	if user.Email() != handle+"@example.com" {
		t.Errorf("Email() = %q, want lowercased %q", user.Email(), handle+"@example.com")
	}
}

func TestUserRepo_Get(t *testing.T) {
	seeded := seedTestUser(t)

	repo := NewUserRepo(testStore)
	got, err := repo.Get(context.Background(), seeded.ID())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID() != seeded.ID() {
		t.Errorf("Get().ID() = %d, want %d", got.ID(), seeded.ID())
	}
}

func TestUserRepo_Get_NotFound(t *testing.T) {
	repo := NewUserRepo(testStore)
	_, err := repo.Get(context.Background(), 999999)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("Get() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserRepo_GetByUsername(t *testing.T) {
	seeded := seedTestUser(t)

	repo := NewUserRepo(testStore)
	got, err := repo.GetByUsername(context.Background(), strings.ToUpper(seeded.UserName()))
	if err != nil {
		t.Fatalf("GetByUsername() error = %v", err)
	}
	if got.ID() != seeded.ID() {
		t.Errorf("GetByUsername().ID() = %d, want %d", got.ID(), seeded.ID())
	}
}

func TestUserRepo_GetByUsername_NotFound(t *testing.T) {
	repo := NewUserRepo(testStore)
	_, err := repo.GetByUsername(context.Background(), "no-such-user")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("GetByUsername() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserRepo_GetByEmail(t *testing.T) {
	seeded := seedTestUser(t)

	repo := NewUserRepo(testStore)
	got, err := repo.GetByEmail(context.Background(), strings.ToUpper(seeded.Email()))
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}
	if got.ID() != seeded.ID() {
		t.Errorf("GetByEmail().ID() = %d, want %d", got.ID(), seeded.ID())
	}
}

func TestUserRepo_GetByEmail_NotFound(t *testing.T) {
	repo := NewUserRepo(testStore)
	_, err := repo.GetByEmail(context.Background(), "no-such-user@example.com")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("GetByEmail() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserRepo_UpdateProfile_Full(t *testing.T) {
	seeded := seedTestUser(t)
	newName := "New Name"
	newUsername := seeded.UserName() + "_updated"
	newEmail := "updated_" + seeded.Email()

	repo := NewUserRepo(testStore)
	got, err := repo.UpdateProfile(context.Background(), seeded.ID(), &newName, &newUsername, &newEmail)
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if got.FullName() != newName {
		t.Errorf("FullName() = %q, want %q", got.FullName(), newName)
	}
	if got.UserName() != newUsername {
		t.Errorf("UserName() = %q, want %q", got.UserName(), newUsername)
	}
	if got.Email() != newEmail {
		t.Errorf("Email() = %q, want %q", got.Email(), newEmail)
	}
}

// TestUserRepo_UpdateProfile_PartialUpdate is the important one: passing nil
// for userName/email must leave those columns untouched (the COALESCE/narg
// behavior), not blank them out.
func TestUserRepo_UpdateProfile_PartialUpdate(t *testing.T) {
	seeded := seedTestUser(t)
	newName := "Only The Name Changed"

	repo := NewUserRepo(testStore)
	got, err := repo.UpdateProfile(context.Background(), seeded.ID(), &newName, nil, nil)
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if got.FullName() != newName {
		t.Errorf("FullName() = %q, want %q", got.FullName(), newName)
	}
	if got.UserName() != seeded.UserName() {
		t.Errorf("UserName() = %q, want unchanged %q", got.UserName(), seeded.UserName())
	}
	if got.Email() != seeded.Email() {
		t.Errorf("Email() = %q, want unchanged %q", got.Email(), seeded.Email())
	}
}

func TestUserRepo_UpdateProfile_NotFound(t *testing.T) {
	newName := "Doesn't Matter"
	repo := NewUserRepo(testStore)
	_, err := repo.UpdateProfile(context.Background(), 999999, &newName, nil, nil)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("UpdateProfile() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserRepo_UpdatePassword(t *testing.T) {
	seeded := seedTestUser(t)

	repo := NewUserRepo(testStore)
	got, err := repo.UpdatePassword(context.Background(), seeded.ID(), "new-hashed-password")
	if err != nil {
		t.Fatalf("UpdatePassword() error = %v", err)
	}
	if got.PasswordHash() != "new-hashed-password" {
		t.Errorf("PasswordHash() = %q, want %q", got.PasswordHash(), "new-hashed-password")
	}
}

func TestUserRepo_UpdatePassword_NotFound(t *testing.T) {
	repo := NewUserRepo(testStore)
	_, err := repo.UpdatePassword(context.Background(), 999999, "x")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("UpdatePassword() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserRepo_UpdateLastLogin(t *testing.T) {
	seeded := seedTestUser(t)
	if seeded.LastLogin() != nil {
		t.Fatalf("seeded user LastLogin() = %v, want nil before first login", seeded.LastLogin())
	}

	repo := NewUserRepo(testStore)
	before := time.Now().Add(-time.Second)
	got, err := repo.UpdateLastLogin(context.Background(), seeded.ID())
	if err != nil {
		t.Fatalf("UpdateLastLogin() error = %v", err)
	}
	if got.LastLogin() == nil {
		t.Fatal("LastLogin() = nil, want a timestamp after logging in")
	}
	if got.LastLogin().Before(before) {
		t.Errorf("LastLogin() = %v, want a time at/after %v", got.LastLogin(), before)
	}
}

func TestUserRepo_UpdateLastLogin_NotFound(t *testing.T) {
	repo := NewUserRepo(testStore)
	_, err := repo.UpdateLastLogin(context.Background(), 999999)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("UpdateLastLogin() error = %v, want ErrUserNotFound", err)
	}
}
