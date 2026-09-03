package domains_test

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	agentclient "github.com/GVALFER/WEBYCP/internal/agent/client"
	agentserver "github.com/GVALFER/WEBYCP/internal/agent/server"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/domains"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite"
)

type accountManager struct{}

func (accountManager) Ensure(context.Context, string, string) error { return nil }

type domainManager struct {
	mu        sync.Mutex
	action    string
	accountID string
	user      string
	domainID  string
	name      string
	version   string
	aliases   []string
	renameErr error
	ensureErr error
}

func (m *domainManager) Ensure(
	_ context.Context,
	accountID, systemUser, domainID, name, version string,
	aliases []string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.action = "ensure"
	m.accountID = accountID
	m.user = systemUser
	m.domainID = domainID
	m.name = name
	m.version = version
	m.aliases = aliases
	return m.ensureErr
}

func (m *domainManager) Disable(_ context.Context, _, _, domainID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.action = "disable"
	m.domainID = domainID
	return nil
}

func (m *domainManager) Delete(_ context.Context, _, _, domainID, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.action = "delete"
	m.domainID = domainID
	m.name = name
	return nil
}

func (m *domainManager) Rename(
	_ context.Context,
	accountID, systemUser, domainID, currentName, name, version string,
	aliases []string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.action = "rename"
	m.accountID = accountID
	m.user = systemUser
	m.domainID = domainID
	m.name = currentName + ":" + name
	m.version = version
	m.aliases = aliases
	return m.renameErr
}

func (m *domainManager) state() (string, string, string, string, string, string, []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.action, m.accountID, m.user, m.domainID, m.name, m.version, append([]string(nil), m.aliases...)
}

func (m *domainManager) setRenameError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.renameErr = err
}

func (m *domainManager) setEnsureError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureErr = err
}

func TestCreateProvisionsDomain(t *testing.T) {
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
	domainHost := &domainManager{}
	agentServer := &http.Server{Handler: agentserver.New(agentserver.Options{
		Version: "test", Accounts: accountManager{}, Domains: domainHost,
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
	accountService := accounts.NewService(store, store, agent, worker.Notify)
	domainService := domains.NewService(store, accountService, store, agent, worker.Notify)
	worker.Handle(jobs.KindAccountCreate, accountService.Provision)
	worker.Handle(jobs.KindAliasCreate, domainService.ProvisionAlias)
	worker.Handle(jobs.KindAliasDelete, domainService.ProvisionAliasAction)
	worker.Handle(jobs.KindAliasDisable, domainService.ProvisionAliasAction)
	worker.Handle(jobs.KindAliasEnable, domainService.ProvisionAliasAction)
	worker.Handle(jobs.KindAliasUpdate, domainService.ProvisionAliasRename)
	worker.Handle(jobs.KindDomainCreate, domainService.Provision)
	worker.Handle(jobs.KindDomainDelete, domainService.ProvisionDomainAction)
	worker.Handle(jobs.KindDomainDisable, domainService.ProvisionDomainAction)
	worker.Handle(jobs.KindDomainEnable, domainService.ProvisionDomainAction)
	worker.Handle(jobs.KindDomainUpdate, domainService.ProvisionDomainRename)
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()

	account, _, err := accountService.Create(ctx, "Example Hosting", node.ID, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() (string, error) {
		current, err := store.Account(ctx, account.ID)
		return current.Status, err
	}, "active")

	domain, job, err := domainService.Create(ctx, account.ID, "Example.COM.", "user-1", false)
	if err != nil {
		t.Fatal(err)
	}
	if domain.Name != "example.com" || domain.Status != "pending" || job.Status != "queued" {
		t.Fatalf("domain = %+v, job = %+v", domain, job)
	}
	waitFor(t, func() (string, error) {
		current, err := store.Domain(ctx, domain.ID)
		return current.Status, err
	}, "active")
	_, hostAccount, hostUser, hostDomain, hostName, hostVersion, hostAliases := domainHost.state()
	if hostAccount != account.ID || hostUser != account.SystemUser ||
		hostDomain != domain.ID || hostName != "example.com" ||
		hostVersion != "8.3" || len(hostAliases) != 0 {
		t.Fatalf("agent domain request = %+v", domainHost)
	}

	alias, aliasJob, err := domainService.CreateAlias(
		ctx, domain.ID, "WWW.Example.COM.", "user-1", false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if alias.Name != "www.example.com" || alias.Status != "pending" || aliasJob.Status != "queued" {
		t.Fatalf("alias = %+v, job = %+v", alias, aliasJob)
	}
	waitFor(t, func() (string, error) {
		current, err := store.Alias(ctx, alias.ID)
		return current.Status, err
	}, "active")
	_, _, _, _, _, _, hostAliases = domainHost.state()
	if len(hostAliases) != 1 || hostAliases[0] != "www.example.com" {
		t.Fatalf("agent aliases = %#v", hostAliases)
	}
	listed, err := domainService.Aliases(ctx, domain.ID, "user-1", false)
	if err != nil || len(listed) != 1 || listed[0].ID != alias.ID {
		t.Fatalf("aliases = %+v, error = %v", listed, err)
	}
	if _, err := domainService.Aliases(ctx, domain.ID, "user-2", false); !errors.Is(err, accounts.ErrForbidden) {
		t.Fatalf("expected alias access error, got %v", err)
	}
	_, _, err = domainService.CreateAlias(ctx, domain.ID, "example.com", "user-1", false)
	if !errors.Is(err, domains.ErrNameExists) {
		t.Fatalf("expected primary name collision, got %v", err)
	}

	_, _, err = domainService.Create(ctx, account.ID, "EXAMPLE.com", "user-1", false)
	if !errors.Is(err, domains.ErrNameExists) {
		t.Fatalf("expected duplicate domain error, got %v", err)
	}
	_, _, err = domainService.Create(ctx, account.ID, "other.example", "user-2", false)
	if !errors.Is(err, accounts.ErrForbidden) {
		t.Fatalf("expected account access error, got %v", err)
	}

	renamedAlias, _, err := domainService.RenameAlias(
		ctx, domain.ID, alias.ID, "Static.Example.COM.", "user-1", false,
	)
	if err != nil || renamedAlias.Name != "static.example.com" || renamedAlias.Status != "pending" {
		t.Fatalf("rename alias = %+v, error = %v", renamedAlias, err)
	}
	waitFor(t, func() (string, error) {
		current, err := store.Alias(ctx, alias.ID)
		if err == nil && current.Status == "active" && current.Name != "static.example.com" {
			return current.Name, nil
		}
		return current.Status, err
	}, "active")
	_, _, err = domainService.RenameAlias(
		ctx, domain.ID, alias.ID, "example.com", "user-1", false,
	)
	if !errors.Is(err, domains.ErrNameExists) {
		t.Fatalf("expected alias rename collision, got %v", err)
	}

	renamedDomain, _, err := domainService.RenameDomain(
		ctx, domain.ID, "Renamed.Example.COM.", "user-1", false,
	)
	if err != nil || renamedDomain.Name != "renamed.example.com" || renamedDomain.Status != "pending" {
		t.Fatalf("rename domain = %+v, error = %v", renamedDomain, err)
	}
	waitFor(t, func() (string, error) {
		current, err := store.Domain(ctx, domain.ID)
		if err == nil && current.Status == "active" && current.Name != "renamed.example.com" {
			return current.Name, nil
		}
		return current.Status, err
	}, "active")
	action, _, _, _, renamedNames, _, _ := domainHost.state()
	if action != "rename" || renamedNames != "example.com:renamed.example.com" {
		t.Fatalf("domain rename action = %q, names = %q", action, renamedNames)
	}
	domainHost.setRenameError(errors.New("rename failed"))
	if _, _, err := domainService.RenameDomain(
		ctx, domain.ID, "broken.example.com", "user-1", false,
	); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() (string, error) {
		current, err := store.Domain(ctx, domain.ID)
		return current.Status, err
	}, "error")
	currentDomain, err := store.Domain(ctx, domain.ID)
	if err != nil {
		t.Fatal(err)
	}
	if currentDomain.Name != "renamed.example.com" || currentDomain.PreviousName != "" {
		t.Fatalf("failed rename state = %+v", currentDomain)
	}
	domainHost.setRenameError(nil)
	if _, _, err := domainService.SetDomain(ctx, domain.ID, "user-1", false, true); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() (string, error) {
		current, err := store.Domain(ctx, domain.ID)
		return current.Status, err
	}, "active")
	domainHost.setEnsureError(errors.New("alias update failed"))
	if _, _, err := domainService.RenameAlias(
		ctx, domain.ID, alias.ID, "broken.example.com", "user-1", false,
	); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() (string, error) {
		current, err := store.Alias(ctx, alias.ID)
		return current.Status, err
	}, "error")
	currentAlias, err := store.Alias(ctx, alias.ID)
	if err != nil {
		t.Fatal(err)
	}
	if currentAlias.Name != "static.example.com" || currentAlias.PreviousName != "" {
		t.Fatalf("failed alias rename state = %+v", currentAlias)
	}
	domainHost.setEnsureError(nil)

	disabledAlias, _, err := domainService.SetAlias(
		ctx, domain.ID, alias.ID, "user-1", false, false,
	)
	if err != nil || disabledAlias.Enabled || disabledAlias.Status != "pending" {
		t.Fatalf("disable alias = %+v, error = %v", disabledAlias, err)
	}
	waitFor(t, func() (string, error) {
		current, err := store.Alias(ctx, alias.ID)
		return current.Status, err
	}, "disabled")
	_, _, _, _, _, _, hostAliases = domainHost.state()
	if len(hostAliases) != 0 {
		t.Fatalf("disabled aliases = %#v", hostAliases)
	}

	if _, _, err := domainService.SetAlias(
		ctx, domain.ID, alias.ID, "user-1", false, true,
	); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() (string, error) {
		current, err := store.Alias(ctx, alias.ID)
		return current.Status, err
	}, "active")
	_, aliasDeleteJob, err := domainService.DeleteAlias(
		ctx, domain.ID, alias.ID, "user-1", false,
	)
	if err != nil {
		t.Fatal(err)
	}
	waitDeleted(t, func() error {
		_, err := store.Alias(ctx, alias.ID)
		return err
	})
	if err := domainService.ProvisionAliasAction(ctx, aliasDeleteJob); err != nil {
		t.Fatalf("retry deleted alias: %v", err)
	}

	if _, _, err := domainService.SetDomain(ctx, domain.ID, "user-1", false, false); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() (string, error) {
		current, err := store.Domain(ctx, domain.ID)
		return current.Status, err
	}, "disabled")
	action, _, _, _, _, _, _ = domainHost.state()
	if action != "disable" {
		t.Fatalf("domain action = %q", action)
	}
	if _, _, err := domainService.SetDomain(ctx, domain.ID, "user-1", false, true); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() (string, error) {
		current, err := store.Domain(ctx, domain.ID)
		return current.Status, err
	}, "active")
	_, domainDeleteJob, err := domainService.DeleteDomain(ctx, domain.ID, "user-1", false)
	if err != nil {
		t.Fatal(err)
	}
	waitDeleted(t, func() error {
		_, err := store.Domain(ctx, domain.ID)
		return err
	})
	action, _, _, _, deletedName, _, _ := domainHost.state()
	if action != "delete" || deletedName != "renamed.example.com" {
		t.Fatalf("domain delete action = %q, name = %q", action, deletedName)
	}
	if err := domainService.ProvisionDomainAction(ctx, domainDeleteJob); err != nil {
		t.Fatalf("retry deleted domain: %v", err)
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

func waitDeleted(t *testing.T, current func() error) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		err := current()
		if errors.Is(err, sql.ErrNoRows) {
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		select {
		case <-deadline:
			t.Fatal("resource was not deleted")
		case <-ticker.C:
		}
	}
}

func createOwner(t *testing.T, ctx context.Context, store *sqlite.Store) {
	t.Helper()
	now := time.Now().UTC()
	err := store.Bootstrap(ctx, auth.NewUser{
		ID: "user-1", Email: "admin@example.com", Name: "Admin",
		PasswordHash: "test", Role: "admin", CreatedAt: now,
	}, auth.NewSession{
		ID: "session-1", UserID: "user-1", TokenHash: "hash", CSRFToken: "csrf",
		ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func waitFor(t *testing.T, current func() (string, error), want string) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		status, err := current()
		if err != nil {
			t.Fatal(err)
		}
		if status == want {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("status = %s, want %s", status, want)
		case <-ticker.C:
		}
	}
}
