package crontab

import (
	"context"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GVALFER/WEBYCP/internal/agent/hostuser"
	"github.com/GVALFER/WEBYCP/internal/agent/scheduler"
)

func TestSyncWritesAndRemovesAccountCrontab(t *testing.T) {
	driver := testDriver(t)
	accountID := "0123456789abcdef0123456789abcdef"
	entryID := "abcdef0123456789abcdef0123456789"
	entries := []scheduler.Entry{{ID: entryID, Kind: "command", Schedule: "0 * * * *", Command: "php web/example.com/task.php"}}
	if err := driver.Sync(context.Background(), accountID, "wcp_0123456789ab", entries); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(driver.dir, "webycp-"+accountID)
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	contents := string(data)
	if !strings.Contains(contents, "MAILTO=\"\"") {
		t.Fatalf("cron output policy is missing:\n%s", contents)
	}
	if !strings.Contains(contents, "0 * * * * wcp_0123456789ab cd -- /home/wcp_0123456789ab && php web/example.com/task.php") {
		t.Fatalf("unexpected crontab:\n%s", contents)
	}
	if err := driver.Sync(context.Background(), accountID, "wcp_0123456789ab", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("crontab still exists: %v", err)
	}
}

func TestSyncRejectsCronPercentExpansion(t *testing.T) {
	driver := testDriver(t)
	err := driver.Sync(context.Background(), "0123456789abcdef0123456789abcdef", "wcp_0123456789ab", []scheduler.Entry{{ID: "abcdef0123456789abcdef0123456789", Kind: "command", Schedule: "* * * * *", Command: "date +%s"}})
	if err == nil {
		t.Fatal("expected unsafe percent to be rejected")
	}
}

func testDriver(t *testing.T) *Driver {
	t.Helper()
	driver := New()
	driver.dir = t.TempDir()
	driver.lookup = func(name string) (*user.User, error) {
		return &user.User{Uid: "1001", Gid: "1001", Username: name, HomeDir: "/home/" + name, Name: hostuser.Marker("0123456789abcdef0123456789abcdef")}, nil
	}
	return driver
}

func TestSyncRejectsOtherAccountsAndRoot(t *testing.T) {
	for _, field := range []string{"username", "uid", "marker", "home"} {
		t.Run(field, func(t *testing.T) {
			driver := testDriver(t)
			lookup := driver.lookup
			driver.lookup = func(name string) (*user.User, error) {
				found, err := lookup(name)
				switch field {
				case "uid":
					found.Uid = "0"
				case "marker":
					found.Name = "not-managed"
				case "home":
					found.HomeDir = "/root"
				}
				return found, err
			}
			name := "wcp_0123456789ab"
			if field == "username" {
				name = "wcp_abcdef012345"
			}
			if err := driver.Sync(context.Background(), "0123456789abcdef0123456789abcdef", name, nil); err == nil {
				t.Fatal("unmanaged account was accepted")
			}
		})
	}
}

func TestInvalidTaskKeepsInstalledCrontab(t *testing.T) {
	driver := testDriver(t)
	account := "0123456789abcdef0123456789abcdef"
	entry := scheduler.Entry{ID: "abcdef0123456789abcdef0123456789", Kind: "command", Schedule: "* * * * *", Command: "/usr/bin/true"}
	if err := driver.Sync(context.Background(), account, "wcp_0123456789ab", []scheduler.Entry{entry}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(driver.dir, "webycp-"+account)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	entry.Kind = "http"
	if err := driver.Sync(context.Background(), account, "wcp_0123456789ab", []scheduler.Entry{entry}); err == nil {
		t.Fatal("unsupported task kind accepted")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatalf("invalid task changed crontab: %v", err)
	}
}
