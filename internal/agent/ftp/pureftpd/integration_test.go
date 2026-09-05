//go:build linux && integration

package pureftpd

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"io"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/GVALFER/WEBYCP/internal/agent/ftp"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/execx"
	"github.com/GVALFER/WEBYCP/internal/secret"
	"golang.org/x/sys/unix"
)

//go:embed testdata/ftps.py
var wireChecks string

// TestUbuntuFTPS must only run in an explicitly opted-in disposable container.
// It creates Unix identities and runs the real packaged Pure-FTPd executable.
func TestUbuntuFTPS(t *testing.T) {
	if os.Getenv("WEBYCP_FTP_INTEGRATION") != "1" || os.Geteuid() != 0 {
		t.Skip("requires a disposable Ubuntu container with WEBYCP_FTP_INTEGRATION=1")
	}
	if _, err := os.Stat("/.dockerenv"); err != nil {
		t.Fatal("refusing host-level FTP fixtures outside a disposable container")
	}
	ctx := context.Background()
	addAccount(t, accountID, systemUser)
	const otherID = "fedcba9876543210fedcba9876543210"
	const otherUser = "wcp_fedcba987654"
	addAccount(t, otherID, otherUser)
	if err := os.MkdirAll("/run/pure-ftpd", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc", filepath.Join("/home", systemUser, "escape")); err != nil {
		t.Fatal(err)
	}
	d := New()
	d.dir = filepath.Join(t.TempDir(), "ftp")
	password, err := secret.Generate(24)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	entry := ftp.Entry{ID: entryID, Username: "customer", PasswordHash: hash, Enabled: true}
	if err := d.Sync(ctx, accountID, systemUser, []ftp.Entry{entry}); err != nil {
		t.Fatal(err)
	}
	otherEntry := ftp.Entry{ID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Username: "other", PasswordHash: hash, Enabled: true}
	if err := d.Sync(ctx, otherID, otherUser, []ftp.Entry{otherEntry}); err != nil {
		t.Fatal(err)
	}

	cert, key := certificateFixture(t, time.Now().Add(time.Hour))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	unit, err := os.ReadFile(os.Getenv("WEBYCP_FTP_UNIT"))
	if err != nil {
		t.Fatal("WEBYCP_FTP_UNIT must point to the proposed service unit")
	}
	var args []string
	for _, line := range strings.Split(string(unit), "\n") {
		if strings.HasPrefix(line, "ExecStart=") {
			args = strings.Fields(strings.ReplaceAll(strings.TrimPrefix(line, "ExecStart="), Dir, d.dir))
		}
	}
	if len(args) == 0 {
		t.Fatal("FTP unit is missing ExecStart")
	}
	args = append(args, "--bind=127.0.0.1,"+strconv.Itoa(port))
	var daemon *exec.Cmd
	stop := func() {
		if daemon != nil {
			_ = daemon.Process.Kill()
			_ = daemon.Wait()
			daemon = nil
		}
	}
	t.Cleanup(stop)
	d.run = func(ctx context.Context, name string, values ...string) error {
		if name != "/usr/bin/systemctl" {
			return execx.Run(ctx, name, values...)
		}
		stop()
		daemon = exec.Command(args[0], args[1:]...)
		if err := daemon.Start(); err != nil {
			return err
		}
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			connection, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 100*time.Millisecond)
			if err == nil {
				connection.Close()
				return nil
			}
			time.Sleep(50 * time.Millisecond)
		}
		return context.DeadlineExceeded
	}
	if err := d.InstallTLS(ctx, cert, key); err != nil {
		t.Fatal(err)
	}

	check := func(mode, username, password string) func() {
		t.Helper()
		command := exec.Command("/usr/bin/python3", "-u", "-c", wireChecks)
		command.Stderr = os.Stderr
		input, err := command.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}
		output, err := command.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		if err := command.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			input.Close()
			_ = command.Process.Kill()
			_ = command.Wait()
		})
		if err := json.NewEncoder(input).Encode(map[string]any{"mode": mode, "username": username, "password": password, "port": port, "certificate": cert}); err != nil {
			t.Fatal(err)
		}
		if mode == "revoked" || mode == "retained" {
			ready, err := bufio.NewReader(output).ReadString('\n')
			if err != nil || ready != "ready\n" {
				t.Fatalf("FTP session did not become ready: %v", err)
			}
			return func() {
				_, _ = io.WriteString(input, "check\n")
				input.Close()
				if err := command.Wait(); err != nil {
					t.Fatalf("revocation check failed: %v", err)
				}
			}
		}
		input.Close()
		_, _ = io.Copy(io.Discard, output)
		if err := command.Wait(); err != nil {
			t.Fatalf("%s check failed: %v", mode, err)
		}
		return nil
	}
	check("plaintext", entry.Username, password)
	check("denied", entry.Username, "wrong")
	check("transfer", entry.Username, password)
	t.Log("TLS control/data, upload/download and jail checks passed")
	verifyClosed := check("revoked", entry.Username, password)
	verifyRetained := check("retained", otherEntry.Username, password)
	oldPassword := password
	password, err = secret.Generate(24)
	if err != nil {
		t.Fatal(err)
	}
	entry.PasswordHash, err = auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Sync(ctx, accountID, systemUser, []ftp.Entry{entry}); err != nil {
		t.Fatal(err)
	}
	verifyClosed()
	verifyRetained()
	check("denied", entry.Username, oldPassword)
	check("transfer", entry.Username, password)
	cert, key = certificateFixture(t, time.Now().Add(2*time.Hour))
	if err := d.InstallTLS(ctx, cert, key); err != nil {
		t.Fatal(err)
	}
	check("transfer", entry.Username, password)
	t.Log("the renewed certificate is served and verified by the FTPS client")
	verifyClosed = check("revoked", entry.Username, password)
	if err := d.Disable(ctx, accountID, systemUser); err != nil {
		t.Fatal(err)
	}
	verifyClosed()
	check("denied", entry.Username, password)
	if err := d.Enable(ctx, accountID, systemUser); err != nil {
		t.Fatal(err)
	}
	check("transfer", entry.Username, password)
	if err := d.Delete(ctx, accountID, systemUser); err != nil {
		t.Fatal(err)
	}
	check("denied", entry.Username, password)
	contents, err := os.ReadFile(filepath.Join("/home", systemUser, "upload.txt"))
	if err != nil || string(contents) != "WEBYCP encrypted transfer\n" {
		t.Fatal("FTP access deletion modified uploaded files")
	}
	identity, err := d.identity(accountID, systemUser)
	if err != nil {
		t.Fatal(err)
	}
	var info unix.Stat_t
	if err := unix.Lstat(filepath.Join("/home", systemUser, "upload.txt"), &info); err != nil || int(info.Uid) != identity.UID || int(info.Gid) != identity.GID || info.Mode&0o777 != 0o640 {
		t.Fatalf("uploaded file owner %d:%d, mode %04o; expected %d:%d, 0640; error: %v", info.Uid, info.Gid, info.Mode&0o777, identity.UID, identity.GID, err)
	}
	if err := d.Delete(ctx, otherID, otherUser); err != nil {
		t.Fatal(err)
	}
	t.Log("password rotation, session revocation, suspension, re-enable and deletion passed")
}

func addAccount(t *testing.T, id, name string) {
	t.Helper()
	ctx := context.Background()
	if _, err := user.Lookup(name); err == nil {
		t.Fatal("refusing to adopt an existing integration account")
	}
	if err := execx.Run(ctx, "/usr/sbin/useradd", "--create-home", "--user-group", "--shell", "/usr/sbin/nologin", "--comment", "WEBYCP-"+id, "--", name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := execx.Run(ctx, "/usr/sbin/userdel", "--remove", "--", name); err != nil {
			t.Error(err)
		}
	})
}
