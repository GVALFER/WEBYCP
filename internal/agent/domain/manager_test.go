package domain

import (
	"context"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/GVALFER/WEBYCP/internal/agent/hostuser"
	agentruntime "github.com/GVALFER/WEBYCP/internal/agent/runtime"
	"github.com/GVALFER/WEBYCP/internal/agent/webserver"
)

const (
	testAccountID = "0123456789abcdef0123456789abcdef"
	testDomainID  = "fedcba9876543210fedcba9876543210"
	testUser      = "wcp_0123456789ab"
)

type driver struct {
	site     webserver.Site
	disabled string
	deleted  string
	err      error
}

func (d *driver) Ensure(_ context.Context, site webserver.Site) error {
	if d.err != nil {
		return d.err
	}
	d.site = site
	return nil
}

func (d *driver) Disable(_ context.Context, domainID string) error {
	d.disabled = domainID
	return nil
}

func (d *driver) Delete(_ context.Context, domainID string) error {
	d.deleted = domainID
	return nil
}

type runtimeDriver struct {
	account agentruntime.Account
	err     error
}

func (d *runtimeDriver) Ensure(
	_ context.Context,
	account agentruntime.Account,
) (agentruntime.Pool, error) {
	d.account = account
	if d.err != nil {
		return agentruntime.Pool{}, d.err
	}
	return agentruntime.Pool{Socket: "/run/php/test.sock"}, nil
}

func TestEnsureCreatesPublicDirectoryAndSite(t *testing.T) {
	manager, runtime, webDriver := testManager(t)
	if err := manager.Ensure(
		context.Background(), testAccountID, testUser, testDomainID,
		"example.com", "8.3", []string{"www.example.com"},
	); err != nil {
		t.Fatal(err)
	}
	public := filepath.Join(manager.home, testUser, "web", "example.com", "public_html")
	info, err := os.Stat(public)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o750 {
		t.Fatalf("public mode = %o, want 750", mode)
	}
	if info.Mode()&os.ModeSetgid == 0 {
		t.Fatal("public directory is missing setgid mode")
	}
	if webDriver.site.ID != testDomainID || webDriver.site.Name != "example.com" || webDriver.site.Root != public {
		t.Fatalf("site = %+v", webDriver.site)
	}
	if webDriver.site.PHPSocket != "/run/php/test.sock" ||
		runtime.account.ID != testAccountID || runtime.account.Version != "8.3" {
		t.Fatalf("runtime = %+v, site = %+v", runtime.account, webDriver.site)
	}
	if len(webDriver.site.Aliases) != 1 || webDriver.site.Aliases[0] != "www.example.com" {
		t.Fatalf("site aliases = %#v", webDriver.site.Aliases)
	}
}

func TestEnsureRejectsSymlinkedDomainDirectory(t *testing.T) {
	manager, _, webDriver := testManager(t)
	web := filepath.Join(manager.home, testUser, "web")
	if err := os.Symlink(t.TempDir(), filepath.Join(web, "example.com")); err != nil {
		t.Fatal(err)
	}
	err := manager.Ensure(
		context.Background(), testAccountID, testUser, testDomainID, "example.com", "8.3", nil,
	)
	if err == nil {
		t.Fatal("expected symlink error")
	}
	if webDriver.site.ID != "" {
		t.Fatal("web server should not be configured")
	}
}

func TestEnsureDoesNotInstallSiteWhenRuntimeFails(t *testing.T) {
	manager, runtime, webDriver := testManager(t)
	runtime.err = errors.New("runtime failed")
	err := manager.Ensure(
		context.Background(), testAccountID, testUser, testDomainID, "example.com", "8.3", nil,
	)
	if err == nil {
		t.Fatal("expected runtime error")
	}
	if webDriver.site.ID != "" {
		t.Fatal("web server should not be configured")
	}
}

func TestDisableRemovesSiteFromService(t *testing.T) {
	manager, _, webDriver := testManager(t)
	if err := manager.Disable(context.Background(), testAccountID, testUser, testDomainID); err != nil {
		t.Fatal(err)
	}
	if webDriver.disabled != testDomainID {
		t.Fatalf("disabled domain = %q", webDriver.disabled)
	}
}

func TestDeleteQuarantinesDomainDirectory(t *testing.T) {
	manager, _, webDriver := testManager(t)
	source := filepath.Join(manager.home, testUser, "web", "example.com")
	if err := os.Mkdir(source, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := manager.Delete(
		context.Background(), testAccountID, testUser, testDomainID, "example.com",
	); err != nil {
		t.Fatal(err)
	}
	if webDriver.deleted != testDomainID {
		t.Fatalf("deleted domain = %q", webDriver.deleted)
	}
	if _, err := os.Stat(filepath.Join(manager.home, testUser, ".webycp-trash", testDomainID)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(source); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("source still exists: %v", err)
	}
	if err := manager.Delete(
		context.Background(), testAccountID, testUser, testDomainID, "example.com",
	); err != nil {
		t.Fatalf("idempotent delete: %v", err)
	}
}

func TestRenameMovesDirectoryAndUpdatesSite(t *testing.T) {
	manager, _, webDriver := testManager(t)
	oldPath := filepath.Join(manager.home, testUser, "web", "example.com")
	if err := os.Mkdir(oldPath, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := manager.Rename(
		context.Background(), testAccountID, testUser, testDomainID,
		"example.com", "renamed.example.com", "8.3", []string{"www.example.com"},
	); err != nil {
		t.Fatal(err)
	}
	newPath := filepath.Join(manager.home, testUser, "web", "renamed.example.com")
	if _, err := os.Stat(newPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(oldPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old path still exists: %v", err)
	}
	if webDriver.site.Name != "renamed.example.com" ||
		webDriver.site.Root != filepath.Join(newPath, "public_html") {
		t.Fatalf("renamed site = %+v", webDriver.site)
	}
	if err := manager.Rename(
		context.Background(), testAccountID, testUser, testDomainID,
		"example.com", "renamed.example.com", "8.3", []string{"www.example.com"},
	); err != nil {
		t.Fatalf("idempotent rename: %v", err)
	}
}

func TestRenameRollsBackDirectoryWhenSiteUpdateFails(t *testing.T) {
	manager, _, webDriver := testManager(t)
	oldPath := filepath.Join(manager.home, testUser, "web", "example.com")
	if err := os.Mkdir(oldPath, 0o750); err != nil {
		t.Fatal(err)
	}
	webDriver.err = errors.New("site update failed")
	err := manager.Rename(
		context.Background(), testAccountID, testUser, testDomainID,
		"example.com", "renamed.example.com", "8.3", nil,
	)
	if err == nil {
		t.Fatal("expected site update error")
	}
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(manager.home, testUser, "web", "renamed.example.com")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("renamed path still exists: %v", err)
	}
}

func testManager(t *testing.T) (*Manager, *runtimeDriver, *driver) {
	t.Helper()
	home := t.TempDir()
	accountHome := filepath.Join(home, testUser)
	if err := os.Mkdir(accountHome, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(accountHome, "web"), 0o750); err != nil {
		t.Fatal(err)
	}
	uid, gid := testIDs()
	webDriver := &driver{}
	runtime := &runtimeDriver{}
	manager := New(runtime, webDriver)
	manager.home = home
	manager.lookup = func(name string) (*user.User, error) {
		return &user.User{
			Username: name, Uid: uid, Gid: gid, HomeDir: accountHome,
			Name: "WEBYCP:" + testAccountID,
		}, nil
	}
	manager.lookupGroup = func(string) (*user.Group, error) {
		return &user.Group{Name: hostuser.WebGroup, Gid: gid}, nil
	}
	return manager, runtime, webDriver
}

func testIDs() (string, string) {
	uid := os.Getuid()
	gid := os.Getgid()
	if uid == 0 {
		uid = 1001
	}
	if gid == 0 {
		gid = 1001
	}
	return strconv.Itoa(uid), strconv.Itoa(gid)
}
