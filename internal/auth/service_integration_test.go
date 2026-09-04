package auth_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite"
)

func TestTemporaryAdminProfileAndReset(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "webycp.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := auth.NewService(store)

	credentials, created, err := service.InitAdmin(ctx)
	if err != nil || !created {
		t.Fatalf("initialize admin: created=%v, error=%v", created, err)
	}
	if credentials.Username != "admin" || len(credentials.Password) < 32 {
		t.Fatalf("unexpected generated credentials: username=%q, passwordLength=%d", credentials.Username, len(credentials.Password))
	}
	if _, created, err := service.InitAdmin(ctx); err != nil || created {
		t.Fatalf("second initialization: created=%v, error=%v", created, err)
	}

	temporary, err := service.Login(ctx, credentials.Username, credentials.Password)
	if err != nil || !temporary.User.MustChangePassword {
		t.Fatalf("temporary login: mustChangePassword=%v, error=%v", temporary.User.MustChangePassword, err)
	}
	configured, err := service.UpdateProfile(
		ctx, temporary, "owner", "Test Owner", "owner@example.com", "Europe/Lisbon", "",
		"correct horse battery staple",
	)
	if err != nil {
		t.Fatal(err)
	}
	if configured.User.Username != "owner" || configured.User.Timezone != "Europe/Lisbon" || configured.User.MustChangePassword {
		t.Fatalf("configured user = %+v", configured.User)
	}
	if _, err := service.Login(ctx, "admin", credentials.Password); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("temporary credentials still work: %v", err)
	}

	session, err := service.Login(ctx, "owner", "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateProfile(
		ctx, session, "owner", "Test Owner", "owner@example.com", "Europe/Lisbon", "wrong password",
		"another correct password",
	); !errors.Is(err, auth.ErrCurrentPassword) {
		t.Fatalf("wrong current password error = %v", err)
	}

	reset, err := service.ResetPassword(ctx, "owner")
	if err != nil || len(reset.Password) < 32 {
		t.Fatalf("reset password length=%d, error=%v", len(reset.Password), err)
	}
	if _, err := service.Authenticate(ctx, session.Token); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("session survived password reset: %v", err)
	}
	profile, err := store.UpdateProfile(ctx, auth.UserUpdate{
		ID: configured.User.ID, Username: "owner", Name: "Test Owner",
		Email: "owner@example.com", Timezone: "Europe/Lisbon", UpdatedAt: time.Now().UTC(),
	})
	if err != nil || !profile.MustChangePassword {
		t.Fatalf("profile update cleared password rotation: mustChangePassword=%v, error=%v", profile.MustChangePassword, err)
	}
	resetSession, err := service.Login(ctx, reset.Username, reset.Password)
	if err != nil || !resetSession.User.MustChangePassword {
		t.Fatalf("reset login: mustChangePassword=%v, error=%v", resetSession.User.MustChangePassword, err)
	}
}
