package databases_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/databases"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/packages"
	"github.com/GVALFER/WEBYCP/internal/services"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite"
)

func TestGrantCannotCrossAccountBoundary(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "webycp.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	if created, err := store.InitAdmin(ctx,
		auth.NewUser{ID: "user-1", Username: "admin", Email: "admin@example.com", Name: "Admin", PasswordHash: "hash", Role: "admin", CreatedAt: now},
	); err != nil || !created {
		t.Fatalf("create owner: created=%v, error=%v", created, err)
	}
	node, err := store.EnsureLocal(ctx, "test", "/tmp/test-agent.sock")
	if err != nil {
		t.Fatal(err)
	}
	first := createAccount(t, ctx, store, node.ID, "0123456789abcdef0123456789abcdef", "First", now)
	second := createAccount(t, ctx, store, node.ID, "abcdef0123456789abcdef0123456789", "Second", now)
	accountService := accounts.NewService(store, store, nil, packages.NewService(store), func() {})
	service := databases.NewService(store, accountService, store, nil, func() {})
	if _, _, err := service.CreateDatabase(ctx, first.ID, "unsupported", "sqlite", "user-1", true); err == nil {
		t.Fatal("unsupported database driver was accepted")
	}
	database, _, err := service.CreateDatabase(ctx, first.ID, "app", services.MySQL, "user-1", true)
	if err != nil {
		t.Fatal(err)
	}
	user, userJob, password, err := service.CreateUser(ctx, second.ID, "app", services.MySQL, "user-1", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(password) < 32 {
		t.Fatalf("generated password is too short: %d", len(password))
	}
	for {
		claimed, err := store.ClaimJob(ctx, time.Now().UTC())
		if err != nil {
			t.Fatal(err)
		}
		if err := store.CompleteJob(ctx, claimed.ID, time.Now().UTC()); err != nil {
			t.Fatal(err)
		}
		if claimed.ID == userJob.ID {
			break
		}
	}
	storedJob, err := store.Job(ctx, userJob.ID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(storedJob.Payload, password) || strings.Contains(storedJob.Payload, "password") {
		t.Fatal("completed job retained the generated credential")
	}
	if err := store.SetDatabaseStatus(ctx, database.ID, "active"); err != nil {
		t.Fatal(err)
	}
	if err := store.SetUserStatus(ctx, user.ID, "active"); err != nil {
		t.Fatal(err)
	}
	_, _, err = service.SetGrant(ctx, database.ID, user.ID, "user-1", true, true)
	if !errors.Is(err, databases.ErrCrossAccount) {
		t.Fatalf("expected cross-account error, got %v", err)
	}
}

func createAccount(t *testing.T, ctx context.Context, store *sqlite.Store, nodeID, id, name string, now time.Time) accounts.Account {
	t.Helper()
	value, _, err := store.CreateProvision(ctx, accounts.Account{
		ID: id, NodeID: nodeID, Name: name, SystemUser: "wcp_" + id[:12],
		Status: "pending", Enabled: true, CreatedAt: now, UpdatedAt: now,
	}, "user-1", packages.DefaultID, jobs.Job{
		ID: "job-" + id, NodeID: nodeID, UserID: "user-1", Kind: jobs.KindAccountCreate,
		Payload: "{}", MaxAttempts: 1, CreatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateStatus(ctx, value.ID, "active"); err != nil {
		t.Fatal(err)
	}
	value.Status = "active"
	return value
}
