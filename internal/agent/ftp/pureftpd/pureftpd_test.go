package pureftpd

import (
	"context"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GVALFER/WEBYCP/internal/agent/ftp"
	"github.com/GVALFER/WEBYCP/internal/agent/hostuser"
)

const accountID = "0123456789abcdef0123456789abcdef"
const systemUser = "wcp_0123456789ab"
const entryID = "abcdef0123456789abcdef0123456789"

func hashFixture() string {
	return "$argon2id$v=19$m=65536,t=3,p=2$" + strings.Repeat("A", 22) + "$" + strings.Repeat("A", 43)
}

func entryFixture() ftp.Entry {
	return ftp.Entry{ID: entryID, Username: "customer", PasswordHash: hashFixture(), Enabled: true}
}

func driverFixture(t *testing.T) *Driver {
	t.Helper()
	d := New()
	d.dir = filepath.Join(t.TempDir(), "ftp")
	d.home = t.TempDir()
	if err := os.Mkdir(filepath.Join(d.home, systemUser), 0o710); err != nil {
		t.Fatal(err)
	}
	d.lookup = func(name string) (*user.User, error) {
		if name != systemUser {
			return nil, user.UnknownUserError(name)
		}
		return &user.User{Uid: "1000", Gid: "1000", Name: hostuser.Marker(accountID), HomeDir: filepath.Join(d.home, name)}, nil
	}
	d.run = func(_ context.Context, name string, args ...string) error {
		if name != "/usr/bin/pure-pw" || len(args) != 4 || args[0] != "mkdb" || args[2] != "-f" {
			t.Fatal("unexpected privileged command")
		}
		data, err := os.ReadFile(args[3])
		if err != nil {
			return err
		}
		// Preserve the candidate in the fake database for assertions. Real PureDB
		// compilation and authentication are covered by the Ubuntu integration test.
		return os.WriteFile(args[1], data, 0o600)
	}
	d.disconnect = func(_ context.Context, uid int) error {
		if uid != 1000 {
			t.Fatal("unexpected session identity")
		}
		return nil
	}
	return d
}

func databaseFixture(t *testing.T, d *Driver) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(d.dir, "pureftpd.pdb"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestLifecycle(t *testing.T) {
	ctx := context.Background()
	d := driverFixture(t)
	entry := entryFixture()
	untouched := filepath.Join(d.home, systemUser, "customer.txt")
	if err := os.WriteFile(untouched, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := d.Sync(ctx, accountID, systemUser, []ftp.Entry{entry}); err != nil {
			t.Fatal(err)
		}
	}
	contents := databaseFixture(t, d)
	if strings.Count(contents, "customer:") != 1 || !strings.Contains(contents, ":1000:1000:WEBYCP-"+entryID+":"+filepath.Join(d.home, systemUser)+"/./:") {
		t.Fatal("the virtual login is not mapped to the account jail")
	}
	for _, path := range []string{d.path(accountID), filepath.Join(d.dir, "pureftpd.pdb")} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("authentication state permissions: %v", err)
		}
	}
	if err := d.Disable(ctx, accountID, systemUser); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(databaseFixture(t, d)) != "" {
		t.Fatal("suspended account is still in the authentication database")
	}
	if err := d.Sync(ctx, accountID, systemUser, []ftp.Entry{entry}); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(databaseFixture(t, d)) != "" {
		t.Fatal("sync bypassed account suspension")
	}
	if err := d.Enable(ctx, accountID, systemUser); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(databaseFixture(t, d), "customer:") {
		t.Fatal("re-enabling the account did not restore its enabled login")
	}
	entry.Enabled = false
	if err := d.Sync(ctx, accountID, systemUser, []ftp.Entry{entry}); err != nil {
		t.Fatal(err)
	}
	if err := d.Disable(ctx, accountID, systemUser); err != nil {
		t.Fatal(err)
	}
	if err := d.Enable(ctx, accountID, systemUser); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(databaseFixture(t, d)) != "" {
		t.Fatal("account enable resurrected an individually disabled login")
	}
	for range 2 {
		if err := d.Delete(ctx, accountID, systemUser); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(d.path(accountID)); !os.IsNotExist(err) {
		t.Fatal("deleted account authentication state remains")
	}
	if data, err := os.ReadFile(untouched); err != nil || string(data) != "preserve" {
		t.Fatal("deleting FTP access changed customer files")
	}
}

func TestInvalidEntries(t *testing.T) {
	for _, name := range []string{"id", "username", "password", "newline hash", "duplicate id", "duplicate username", "missing entries", "too many"} {
		t.Run(name, func(t *testing.T) {
			d := driverFixture(t)
			entries := []ftp.Entry{entryFixture()}
			switch name {
			case "id":
				entries[0].ID = "../../root"
			case "username":
				entries[0].Username = "root:injected"
			case "password":
				entries[0].PasswordHash = "plaintext"
			case "newline hash":
				entries[0].PasswordHash += "\n"
			case "duplicate id":
				entries = append(entries, entries[0])
				entries[1].Username = "other"
			case "duplicate username":
				entries = append(entries, entries[0])
				entries[1].ID = accountID
			case "missing entries":
				entries = nil
			case "too many":
				entries = make([]ftp.Entry, 101)
			}
			if err := d.Sync(context.Background(), accountID, systemUser, entries); err == nil {
				t.Fatal("invalid entries accepted")
			}
			if _, err := os.Stat(d.dir); !os.IsNotExist(err) {
				t.Fatal("invalid request wrote host state")
			}
		})
	}
}

func TestIdentityBoundary(t *testing.T) {
	for _, name := range []string{"root", "owner", "home", "username", "symlink", "inactive"} {
		t.Run(name, func(t *testing.T) {
			d := driverFixture(t)
			lookup := d.lookup
			d.lookup = func(userName string) (*user.User, error) {
				found, err := lookup(userName)
				if err != nil {
					return nil, err
				}
				switch name {
				case "root":
					found.Uid = "0"
				case "owner":
					found.Name = "unmanaged"
				case "home":
					found.HomeDir = "/root"
				}
				return found, nil
			}
			userName := systemUser
			if name == "username" {
				userName = "wcp_ffffffffffff"
			}
			if name == "symlink" {
				home := filepath.Join(d.home, systemUser)
				if err := os.Remove(home); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(t.TempDir(), home); err != nil {
					t.Fatal(err)
				}
			}
			if name == "inactive" {
				if err := os.Chmod(filepath.Join(d.home, systemUser), 0); err != nil {
					t.Fatal(err)
				}
			}
			if err := d.Sync(context.Background(), accountID, userName, []ftp.Entry{entryFixture()}); err == nil {
				t.Fatal("invalid host identity accepted")
			}
		})
	}
}

func TestBuildFailurePreservesAccess(t *testing.T) {
	d := driverFixture(t)
	ctx := context.Background()
	if err := d.Sync(ctx, accountID, systemUser, []ftp.Entry{entryFixture()}); err != nil {
		t.Fatal(err)
	}
	before := databaseFixture(t, d)
	state, err := os.ReadFile(d.path(accountID))
	if err != nil {
		t.Fatal(err)
	}
	d.run = func(context.Context, string, ...string) error { return errors.New("sensitive subprocess output") }
	err = d.Sync(ctx, accountID, systemUser, []ftp.Entry{})
	if err == nil || strings.Contains(err.Error(), "sensitive") {
		t.Fatal("compilation failure was hidden or exposed subprocess output")
	}
	after, err := os.ReadFile(d.path(accountID))
	if err != nil || string(state) != string(after) || databaseFixture(t, d) != before {
		t.Fatal("failed compilation replaced the previous authentication state")
	}
}

func TestDisconnectRetry(t *testing.T) {
	d := driverFixture(t)
	ctx := context.Background()
	if err := d.Sync(ctx, accountID, systemUser, []ftp.Entry{entryFixture()}); err != nil {
		t.Fatal(err)
	}
	d.disconnect = func(context.Context, int) error { return errors.New("disconnect failed") }
	if err := d.Sync(ctx, accountID, systemUser, []ftp.Entry{}); err == nil {
		t.Fatal("session revocation failure was hidden")
	}
	if strings.TrimSpace(databaseFixture(t, d)) != "" {
		t.Fatal("failed disconnection restored removed credentials")
	}
	calls := 0
	d.disconnect = func(_ context.Context, uid int) error {
		calls++
		if uid != 1000 {
			t.Fatal("retry used the wrong UID")
		}
		return nil
	}
	if err := d.Sync(ctx, accountID, systemUser, []ftp.Entry{}); err != nil || calls != 1 {
		t.Fatal("retry did not terminate remaining account sessions")
	}
}

func TestOtherAccountsAndConflicts(t *testing.T) {
	d := driverFixture(t)
	ctx := context.Background()
	if err := d.Sync(ctx, accountID, systemUser, []ftp.Entry{entryFixture()}); err != nil {
		t.Fatal(err)
	}
	const otherID = "fedcba9876543210fedcba9876543210"
	const otherUser = "wcp_fedcba987654"
	if err := os.Mkdir(filepath.Join(d.home, otherUser), 0o710); err != nil {
		t.Fatal(err)
	}
	lookup := d.lookup
	d.lookup = func(name string) (*user.User, error) {
		if name == otherUser {
			return &user.User{Uid: "1001", Gid: "1001", Name: hostuser.Marker(otherID), HomeDir: filepath.Join(d.home, name)}, nil
		}
		return lookup(name)
	}
	d.disconnect = func(context.Context, int) error { return nil }
	other := entryFixture()
	other.ID, other.Username = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "other"
	if err := d.Sync(ctx, otherID, otherUser, []ftp.Entry{other}); err != nil {
		t.Fatal(err)
	}
	other.Username = "customer"
	if err := d.Sync(ctx, otherID, otherUser, []ftp.Entry{other}); err == nil {
		t.Fatal("another account's username was adopted")
	}
	if err := d.Delete(ctx, accountID, systemUser); err != nil {
		t.Fatal(err)
	}
	contents := databaseFixture(t, d)
	if strings.Contains(contents, "customer:") || !strings.Contains(contents, "other:") {
		t.Fatal("account deletion modified another account's access")
	}
}

func TestSessionOwner(t *testing.T) {
	for _, test := range []struct {
		status string
		uid    int
		want   bool
	}{
		{"Name:\tpure-ftpd\nUid:\t1000\t1000\t1000\t1000\n", 1000, true},
		{"Uid:\t0\t0\t0\t0\n", 0, false},
		{"Uid:\t1001\t1001\t1001\t1001\n", 1000, false},
		{"Uid:\t1000\t0\t0\t0\n", 1000, false},
		{"invalid", 1000, false},
	} {
		if sessionOwner(test.status, test.uid) != test.want {
			t.Fatal("incorrect FTP session identity")
		}
	}
}

func TestPrivateState(t *testing.T) {
	for _, mode := range []string{"public directory", "symlink account"} {
		t.Run(mode, func(t *testing.T) {
			d := driverFixture(t)
			ctx := context.Background()
			if err := d.Sync(ctx, accountID, systemUser, []ftp.Entry{entryFixture()}); err != nil {
				t.Fatal(err)
			}
			before := databaseFixture(t, d)
			outside := filepath.Join(t.TempDir(), "outside.json")
			if err := os.WriteFile(outside, []byte("preserve"), 0o600); err != nil {
				t.Fatal(err)
			}
			if mode == "public directory" {
				if err := os.Chmod(d.dir, 0o755); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.Remove(d.path(accountID)); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, d.path(accountID)); err != nil {
					t.Fatal(err)
				}
			}
			if err := d.Sync(ctx, accountID, systemUser, []ftp.Entry{}); err == nil {
				t.Fatal("unsafe FTP state was accepted")
			}
			if databaseFixture(t, d) != before {
				t.Fatal("unsafe state replaced the authentication database")
			}
			if data, err := os.ReadFile(outside); err != nil || string(data) != "preserve" {
				t.Fatal("FTP synchronization wrote through a state symlink")
			}
		})
	}
}
