package crontab

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GVALFER/WEBYCP/internal/agent/scheduler"
)

func TestSyncWritesAndRemovesAccountCrontab(t *testing.T) {
	driver := New()
	driver.dir = t.TempDir()
	accountID := "0123456789abcdef0123456789abcdef"
	entryID := "abcdef0123456789abcdef0123456789"
	entries := []scheduler.Entry{{ID: entryID, Schedule: "0 * * * *", Command: "php web/example.com/task.php"}}
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
	driver := New()
	driver.dir = t.TempDir()
	err := driver.Sync(context.Background(), "0123456789abcdef0123456789abcdef", "wcp_0123456789ab", []scheduler.Entry{{ID: "abcdef0123456789abcdef0123456789", Schedule: "* * * * *", Command: "date +%s"}})
	if err == nil {
		t.Fatal("expected unsafe percent to be rejected")
	}
}
