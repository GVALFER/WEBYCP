package website

import (
	"context"
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
	testWebsiteID = "fedcba9876543210fedcba9876543210"
	testUser      = "wcp_0123456789ab"
)

type driver struct {
	site              webserver.Site
	disabled, deleted string
	err               error
}

func (d *driver) Ensure(_ context.Context, site webserver.Site) error {
	if d.err != nil {
		return d.err
	}
	d.site = site
	return nil
}
func (d *driver) Disable(_ context.Context, id string) error { d.disabled = id; return nil }
func (d *driver) Delete(_ context.Context, id string) error  { d.deleted = id; return nil }

type runtimeDriver struct{ account agentruntime.Account }

func (d *runtimeDriver) Ensure(_ context.Context, account agentruntime.Account) (agentruntime.Pool, error) {
	d.account = account
	return agentruntime.Pool{Socket: "/run/php/test.sock"}, nil
}

func TestWebsiteLifecycle(t *testing.T) {
	manager, runtime, web := testManager(t)
	spec := testSpec(manager)
	if err := manager.Ensure(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(spec.DocumentRoot); err != nil {
		t.Fatal(err)
	}
	if web.site.ID != testWebsiteID || web.site.Name != "example.com" || web.site.Root != spec.DocumentRoot || runtime.account.Version != "8.3" {
		t.Fatalf("runtime = %+v, site = %+v", runtime.account, web.site)
	}
	if err := manager.Disable(context.Background(), spec); err != nil || web.disabled != testWebsiteID {
		t.Fatalf("disable: %v, id = %q", err, web.disabled)
	}
	if err := manager.Delete(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	if web.deleted != testWebsiteID {
		t.Fatalf("deleted website = %q", web.deleted)
	}
	if _, err := os.Stat(filepath.Join(manager.home, testUser, ".webycp-trash", testWebsiteID)); err != nil {
		t.Fatal(err)
	}
	if err := manager.Delete(context.Background(), spec); err != nil {
		t.Fatalf("idempotent delete: %v", err)
	}
}

func TestEnsureRejectsDocumentRootOutsideAccount(t *testing.T) {
	manager, _, web := testManager(t)
	spec := testSpec(manager)
	spec.DocumentRoot = filepath.Join(manager.home, "other", "public_html")
	if err := manager.Ensure(context.Background(), spec); err == nil {
		t.Fatal("expected document root validation error")
	}
	if web.site.ID != "" {
		t.Fatal("web server should not be configured")
	}
}

func TestEnsureRejectsSymlinkedWebsiteDirectory(t *testing.T) {
	manager, _, web := testManager(t)
	spec := testSpec(manager)
	if err := os.Symlink(t.TempDir(), filepath.Dir(spec.DocumentRoot)); err != nil {
		t.Fatal(err)
	}
	if err := manager.Ensure(context.Background(), spec); err == nil {
		t.Fatal("expected symlink error")
	}
	if web.site.ID != "" {
		t.Fatal("web server should not be configured")
	}
}

func testSpec(manager *Manager) Spec {
	return Spec{AccountID: testAccountID, SystemUser: testUser, WebsiteID: testWebsiteID, DocumentRoot: filepath.Join(manager.home, testUser, "web", testWebsiteID, "public_html"), Kind: "php", WebDriver: "nginx", RuntimeDriver: "phpfpm", RuntimeVersion: "8.3", PrimaryDomain: "example.com", Aliases: []string{"www.example.com"}}
}

func testManager(t *testing.T) (*Manager, *runtimeDriver, *driver) {
	t.Helper()
	home := t.TempDir()
	accountHome := filepath.Join(home, testUser)
	if err := os.MkdirAll(filepath.Join(accountHome, "web"), 0o750); err != nil {
		t.Fatal(err)
	}
	uid, gid := testIDs()
	web := &driver{}
	runtime := &runtimeDriver{}
	manager := New(runtime, web)
	manager.home = home
	manager.lookup = func(name string) (*user.User, error) {
		return &user.User{Username: name, Uid: uid, Gid: gid, HomeDir: accountHome, Name: hostuser.Marker(testAccountID)}, nil
	}
	manager.lookupGroup = func(string) (*user.Group, error) { return &user.Group{Name: hostuser.WebGroup, Gid: gid}, nil }
	return manager, runtime, web
}

func testIDs() (string, string) {
	return strconv.Itoa(os.Getuid()), strconv.Itoa(os.Getgid())
}
