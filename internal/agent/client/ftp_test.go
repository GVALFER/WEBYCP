package client

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GVALFER/WEBYCP/internal/agent/ftp"
	agentserver "github.com/GVALFER/WEBYCP/internal/agent/server"
)

type ftpSync struct {
	account, user string
	entries       []ftp.Entry
	err           error
}

func (d *ftpSync) Sync(_ context.Context, account, user string, entries []ftp.Entry) error {
	d.account, d.user, d.entries = account, user, entries
	return d.err
}

func TestFTPSocket(t *testing.T) {
	driver := &ftpSync{}
	socket, server := testServer(t, agentserver.Options{FTP: driver})
	defer server.Shutdown(context.Background())
	client := New(time.Second)
	entries := []ftp.Entry{{ID: "abcdef0123456789abcdef0123456789", Username: "customer", PasswordHash: "private-hash", Enabled: true}}
	if err := client.SyncFTP(context.Background(), socket, "0123456789abcdef0123456789abcdef", "wcp_0123456789ab", entries); err != nil {
		t.Fatal(err)
	}
	if driver.account != "0123456789abcdef0123456789abcdef" || driver.user != "wcp_0123456789ab" || len(driver.entries) != 1 || driver.entries[0] != entries[0] {
		t.Fatal("FTP request changed across the Unix socket")
	}
	driver.err = errors.New("private-hash private-path")
	if err := client.SyncFTP(context.Background(), socket, driver.account, driver.user, entries); err == nil || strings.Contains(err.Error(), "private") {
		t.Fatal("internal error was hidden or exposed private state")
	}
}
