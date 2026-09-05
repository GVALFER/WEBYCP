package ftp_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	agentclient "github.com/GVALFER/WEBYCP/internal/agent/client"
	agentftp "github.com/GVALFER/WEBYCP/internal/agent/ftp"
	agentserver "github.com/GVALFER/WEBYCP/internal/agent/server"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/ftp"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/packages"
	"github.com/GVALFER/WEBYCP/internal/pagination"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite"
)

type driver struct {
	account, user string
	entries       []agentftp.Entry
	err           error
}

func (d *driver) Sync(_ context.Context, id, user string, entries []agentftp.Entry) error {
	d.account, d.user, d.entries = id, user, entries
	return d.err
}

type fixture struct {
	ctx      context.Context
	store    *sqlite.Store
	accounts *accounts.Service
	service  *ftp.Service
	account  accounts.Account
	driver   *driver
}

func setup(t *testing.T) fixture {
	t.Helper()
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "ftp.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	if _, err := store.InitAdmin(ctx, auth.NewUser{ID: "owner", Username: "admin", Name: "Admin",
		PasswordHash: "test-only", Role: "admin", CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp("/tmp", "wcp-ftp-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "agent.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	d := &driver{}
	server := &http.Server{Handler: agentserver.New(agentserver.Options{FTP: d}), ReadHeaderTimeout: time.Second}
	go server.Serve(listener)
	t.Cleanup(func() { server.Close() })
	node, err := store.EnsureLocal(ctx, "Test", socket)
	if err != nil {
		t.Fatal(err)
	}
	a := accounts.NewService(store, store, nil, packages.NewService(store), func() {})
	account, job, err := a.Create(ctx, "Hosting", node.ID, packages.DefaultID, "owner")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateStatus(ctx, account.ID, "active"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ClaimJob(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteJob(ctx, job.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	return fixture{ctx: ctx, store: store, accounts: a, account: account, driver: d,
		service: ftp.NewService(store, a, store, agentclient.New(time.Second), func() {})}
}

func (f fixture) sync(t *testing.T, job jobs.Job) {
	t.Helper()
	claimed, err := f.store.ClaimJob(f.ctx, time.Now().UTC())
	if err != nil || claimed.ID != job.ID {
		t.Fatalf("claim FTP job = %s, error = %v", claimed.ID, err)
	}
	if err := f.service.Sync(f.ctx, job); err != nil {
		t.Fatal(err)
	}
	if err := f.store.CompleteJob(f.ctx, job.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
}

func TestLifecycleAcrossAgentSocket(t *testing.T) {
	f := setup(t)
	password := "a long test-only FTP password"
	value, job, err := f.service.Create(f.ctx, f.account.ID, " FTP.Owner ", password, "owner", false, true)
	if err != nil {
		t.Fatal(err)
	}
	if value.Username != "ftp.owner" || job.Kind != jobs.KindFTPSync || strings.Contains(job.Payload, password) {
		t.Fatal("invalid FTP creation or secret in Job payload")
	}
	stored, err := f.store.FTP(f.ctx, value.ID)
	if err != nil || !auth.VerifyPassword(password, stored.PasswordHash) {
		t.Fatal("password hash was not persisted")
	}
	encoded, _ := json.Marshal(stored)
	if strings.Contains(string(encoded), stored.PasswordHash) {
		t.Fatal("credential hash is JSON-visible")
	}
	if _, _, err := f.service.Update(f.ctx, value.ID, "owner", false, nil, nil, new(false)); !errors.Is(err, ftp.ErrBusy) {
		t.Fatalf("parallel mutation = %v", err)
	}
	f.sync(t, job)
	if f.driver.account != f.account.ID || f.driver.user != f.account.SystemUser || len(f.driver.entries) != 1 ||
		!auth.VerifyPassword(password, f.driver.entries[0].PasswordHash) {
		t.Fatal("Agent credential mismatch")
	}
	if _, _, err := f.accounts.Delete(f.ctx, f.account.ID, "owner", false); !errors.Is(err, accounts.ErrNotEmpty) {
		t.Fatalf("account deletion with FTP credentials = %v", err)
	}
	rotated := "a different test-only FTP password"
	_, job, err = f.service.Update(f.ctx, value.ID, "owner", false, new("renamed.ftp"), &rotated, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.sync(t, job)
	entry := f.driver.entries[0]
	if entry.Username != "renamed.ftp" || !entry.Enabled || auth.VerifyPassword(password, entry.PasswordHash) ||
		!auth.VerifyPassword(rotated, entry.PasswordHash) {
		t.Fatal("rotation did not replace the credential")
	}
	for _, enabled := range []bool{false, true} {
		_, job, err = f.service.Update(f.ctx, value.ID, "owner", false, nil, nil, &enabled)
		if err != nil {
			t.Fatal(err)
		}
		f.sync(t, job)
		if f.driver.entries[0].Enabled != enabled {
			t.Fatal("enabled state was not synchronized")
		}
	}
	// A suspended Account can revoke credentials without changing the desired login flags.
	if err := f.store.UpdateStatus(f.ctx, f.account.ID, "disabled"); err != nil {
		t.Fatal(err)
	}
	page, err := f.service.Page(f.ctx, "owner", false, pagination.Query{Page: 1, Size: 10})
	if err != nil || page.Items[0].AccountStatus != "disabled" {
		t.Fatal("missing Account suspension state")
	}
	job, err = f.service.Delete(f.ctx, value.ID, "owner", false)
	if err != nil {
		t.Fatal(err)
	}
	f.driver.err = errors.New("private driver output " + entry.PasswordHash)
	if _, err := f.store.ClaimJob(f.ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := f.service.Sync(f.ctx, job); err == nil || strings.Contains(err.Error(), entry.PasswordHash) {
		t.Fatal("failed revocation was ignored or exposed private output")
	}
	stored, err = f.store.FTP(f.ctx, value.ID)
	if err != nil || !stored.Deleting || stored.Status != "error" {
		t.Fatal("failed deletion lost recoverable metadata")
	}
	if err := f.store.FailJob(f.ctx, job.ID, "test failure", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.service.Update(f.ctx, value.ID, "owner", false, new("resurrected"), nil, nil); !errors.Is(err, ftp.ErrDeleting) {
		t.Fatalf("deleting credential changed = %v", err)
	}
	job, err = f.service.Delete(f.ctx, value.ID, "owner", false)
	if err != nil {
		t.Fatal(err)
	}
	f.driver.err = nil
	f.sync(t, job)
	if len(f.driver.entries) != 0 {
		t.Fatal("deleting login reached Agent")
	}
	if _, err := f.store.FTP(f.ctx, value.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("metadata remains after successful revocation")
	}
	if err := f.service.Sync(f.ctx, job); err != nil {
		t.Fatalf("deletion replay = %v", err)
	}
}

func TestAuthorizationValidationAndNames(t *testing.T) {
	f := setup(t)
	password := "test-only FTP credential"
	for _, input := range []struct{ name, password string }{{"../escape", password}, {"valid", "short"}, {"valid", strings.Repeat("a", 129)}} {
		if _, _, err := f.service.Create(f.ctx, f.account.ID, input.name, input.password, "owner", false, true); err == nil {
			t.Fatal("invalid credential accepted")
		}
	}
	value, job, err := f.service.Create(f.ctx, f.account.ID, "same.name", password, "owner", false, false)
	if err != nil {
		t.Fatal(err)
	}
	f.sync(t, job)
	if _, _, err := f.service.Create(f.ctx, f.account.ID, "Same.Name", password, "owner", false, true); !errors.Is(err, ftp.ErrNameExists) {
		t.Fatalf("disabled name collision = %v", err)
	}
	other, _, err := f.accounts.Create(f.ctx, "Other hosting", f.account.NodeID, packages.DefaultID, "owner")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.store.UpdateStatus(f.ctx, other.ID, "active"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.service.Create(f.ctx, other.ID, "same.name", password, "owner", false, true); !errors.Is(err, ftp.ErrNameExists) {
		t.Fatalf("cross-Account name collision = %v", err)
	}
	if _, _, err := f.service.Create(f.ctx, f.account.ID, "denied", password, "outsider", false, true); !errors.Is(err, accounts.ErrForbidden) {
		t.Fatal("cross-account create allowed")
	}
	if _, _, err := f.service.Update(f.ctx, value.ID, "outsider", false, nil, nil, new(true)); !errors.Is(err, accounts.ErrForbidden) {
		t.Fatal("cross-account update allowed")
	}
	if _, err := f.service.Delete(f.ctx, value.ID, "outsider", false); !errors.Is(err, accounts.ErrForbidden) {
		t.Fatal("cross-account delete allowed")
	}
	page, err := f.service.Page(f.ctx, "outsider", false, pagination.Query{Page: 1, Size: 10})
	if err != nil || page.Total != 0 {
		t.Fatal("cross-account list allowed")
	}
	if _, _, err := f.service.Update(f.ctx, value.ID, "owner", false, nil, nil, nil); err == nil {
		t.Fatal("empty change accepted")
	}
	if err := f.service.Sync(f.ctx, jobs.Job{Payload: `{"accountId":"invalid"}`}); err == nil {
		t.Fatal("invalid job accepted")
	}
	if err := f.store.UpdateStatus(f.ctx, f.account.ID, "disabled"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.service.Create(f.ctx, f.account.ID, "another", password, "owner", true, true); !errors.Is(err, accounts.ErrBusy) {
		t.Fatal("create on suspended Account allowed")
	}
}

func TestAtomicLimitAndLoweredPackage(t *testing.T) {
	f := setup(t)
	value, err := f.store.Package(f.ctx, packages.DefaultID)
	if err != nil {
		t.Fatal(err)
	}
	value.Limits.FTPAccounts = 1
	if _, err := f.store.UpdatePackage(f.ctx, value); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, name := range []string{"first", "second"} {
		wg.Go(func() {
			_, _, err := f.service.Create(f.ctx, f.account.ID, name, "test-only FTP credential", "owner", false, true)
			results <- err
		})
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else if !errors.Is(err, ftp.ErrBusy) {
			t.Fatalf("concurrent create = %v", err)
		}
	}
	if success != 1 {
		t.Fatalf("successful creates = %d", success)
	}
	all, err := f.store.Jobs(f.ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, job := range all {
		if job.Kind == jobs.KindFTPSync {
			f.sync(t, job)
		}
	}
	_, _, err = f.service.Create(f.ctx, f.account.ID, "overlimit", "test-only FTP credential", "owner", false, true)
	var limit *packages.LimitError
	if !errors.As(err, &limit) || limit.Resource != packages.FTPAccounts || limit.Limit != 1 {
		t.Fatalf("capacity error = %v", err)
	}
	value.Limits.FTPAccounts = 0
	if _, err := f.store.UpdatePackage(f.ctx, value); err != nil {
		t.Fatal(err)
	}
	page, err := f.service.Page(f.ctx, "owner", false, pagination.Query{Page: 99, Size: 10})
	if err != nil || page.Query.Page != 1 || page.Total != 1 {
		t.Fatal("pagination did not clamp")
	}
	_, job, err := f.service.Update(f.ctx, page.Items[0].ID, "owner", false, nil, nil, new(false))
	if err != nil {
		t.Fatalf("existing over-limit login cannot be disabled: %v", err)
	}
	f.sync(t, job)
	overview, err := f.store.AccountOverview(f.ctx, f.account.ID)
	if err != nil || overview.Usage.FTPAccounts != 1 || overview.Package.Limits.FTPAccounts != 0 {
		t.Fatal("FTP usage/limit missing")
	}
	value.Limits.FTPAccounts = 101
	if _, err := packages.NewService(f.store).Update(f.ctx, value.ID, value); err == nil {
		t.Fatal("Package exceeds Agent FTP capacity")
	}
}
