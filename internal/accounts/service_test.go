package accounts_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	agentclient "github.com/GVALFER/WEBYCP/internal/agent/client"
	agentserver "github.com/GVALFER/WEBYCP/internal/agent/server"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/packages"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite"
)

type manager struct {
	user string
}

func (m *manager) Ensure(_ context.Context, _, systemUser string) error {
	m.user = systemUser
	return nil
}

func TestCreateProvisionsAccount(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "webycp.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	createOwner(t, ctx, store)
	socket := t.TempDir() + "/agent.sock"
	listener, cleanup, err := agentserver.Listen(socket)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	manager := &manager{}
	agentServer := &http.Server{Handler: agentserver.New(agentserver.Options{
		Version: "test", Accounts: manager,
	})}
	go func() { _ = agentServer.Serve(listener) }()
	defer agentServer.Shutdown(context.Background())

	node, err := store.EnsureLocal(ctx, "test", socket)
	if err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	worker := jobs.NewWorker(store, store, logger)
	agent := agentclient.New(time.Second)
	service := accounts.NewService(store, store, agent, packages.NewService(store), worker.Notify)
	worker.Handle(jobs.KindAccountCreate, service.Provision)
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()

	account, job, err := service.Create(ctx, "Example Hosting", node.ID, packages.DefaultID, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if account.Status != "pending" || job.Status != "queued" {
		t.Fatalf("account = %+v, job = %+v", account, job)
	}
	waitForAccount(t, ctx, store, account.ID, "active")
	if manager.user != account.SystemUser {
		t.Fatalf("system user = %q, want %q", manager.user, account.SystemUser)
	}

	_, _, err = service.Create(ctx, "example hosting", node.ID, packages.DefaultID, "user-1")
	if !errors.Is(err, accounts.ErrNameExists) {
		t.Fatalf("expected duplicate name error, got %v", err)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
}

func createOwner(t *testing.T, ctx context.Context, store *sqlite.Store) {
	t.Helper()
	now := time.Now().UTC()
	created, err := store.InitAdmin(ctx, auth.NewUser{
		ID: "user-1", Username: "admin", Email: "admin@example.com", Name: "Admin",
		PasswordHash: "test", Role: "admin", CreatedAt: now,
	})
	if err != nil || !created {
		t.Fatalf("create owner: created=%v, error=%v", created, err)
	}
}

func waitForAccount(
	t *testing.T,
	ctx context.Context,
	store *sqlite.Store,
	id, status string,
) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		account, err := store.Account(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if account.Status == status {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("account status = %s, want %s", account.Status, status)
		case <-ticker.C:
		}
	}
}
