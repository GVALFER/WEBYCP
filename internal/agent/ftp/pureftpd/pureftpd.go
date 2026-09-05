package pureftpd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/GVALFER/WEBYCP/internal/agent/configfile"
	"github.com/GVALFER/WEBYCP/internal/agent/ftp"
	"github.com/GVALFER/WEBYCP/internal/agent/hostuser"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/execx"
	"github.com/GVALFER/WEBYCP/internal/fsx"
	"github.com/GVALFER/WEBYCP/internal/validate"
	"golang.org/x/sys/unix"
)

const Dir = "/etc/webycp/ftp"

type Driver struct {
	mu         sync.Mutex
	dir, home  string
	lookup     func(string) (*user.User, error)
	run        func(context.Context, string, ...string) error
	disconnect func(context.Context, int) error
}

// An account file retains disabled logins so account re-enablement does not
// resurrect credentials that were individually disabled.
type account struct {
	ID         string      `json:"id"`
	SystemUser string      `json:"systemUser"`
	Suspended  bool        `json:"suspended"`
	Entries    []ftp.Entry `json:"entries"`
}

func New() *Driver {
	return &Driver{dir: Dir, home: "/home", lookup: user.Lookup, run: execx.Run, disconnect: disconnect}
}

func (d *Driver) Sync(ctx context.Context, accountID, systemUser string, entries []ftp.Entry) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, err := d.identity(accountID, systemUser); err != nil {
		return err
	}
	if err := checkEntries(entries); err != nil {
		return err
	}
	value, err := d.read(accountID, systemUser)
	if err != nil {
		return err
	}
	info, err := os.Lstat(filepath.Join(d.home, systemUser))
	if err != nil {
		return err
	}
	if info.Mode().Perm()&0o100 == 0 && !value.Suspended {
		return &validate.Error{Field: "accountId", Message: "The account must be active before changing FTP logins"}
	}
	value.Entries = entries
	return d.save(ctx, value)
}

func (d *Driver) Disable(ctx context.Context, accountID, systemUser string) error {
	return d.setEnabled(ctx, accountID, systemUser, false)
}

func (d *Driver) Enable(ctx context.Context, accountID, systemUser string) error {
	return d.setEnabled(ctx, accountID, systemUser, true)
}

func (d *Driver) setEnabled(ctx context.Context, accountID, systemUser string, enabled bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if hostuser.ValidateNames(accountID, systemUser) != nil {
		return &validate.Error{Field: "accountId", Message: "The account identity is invalid"}
	}
	value, err := d.read(accountID, systemUser)
	if err != nil {
		return err
	}
	// Accounts without FTP state do not require an installed FTP service.
	if value.Entries == nil {
		return nil
	}
	value.Suspended = !enabled
	return d.save(ctx, value)
}

func (d *Driver) Delete(ctx context.Context, accountID, systemUser string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if hostuser.ValidateNames(accountID, systemUser) != nil {
		return &validate.Error{Field: "accountId", Message: "The account identity is invalid"}
	}
	value, err := d.read(accountID, systemUser)
	if err != nil || value.Entries == nil {
		return err
	}
	value.Suspended = true
	// Keep the suspended state until all active sessions have been terminated.
	if err := d.save(ctx, value); err != nil {
		return err
	}
	return os.Remove(d.path(accountID))
}

func checkEntries(entries []ftp.Entry) error {
	if entries == nil || len(entries) > 100 {
		return &validate.Error{Field: "entries", Message: "Provide at most 100 FTP logins"}
	}
	ids, names := make(map[string]bool), make(map[string]bool)
	for _, entry := range entries {
		if validate.ID("ftpId", entry.ID) != nil || ids[entry.ID] {
			return &validate.Error{Field: "id", Message: "FTP login IDs must be valid and unique"}
		}
		name, err := validate.Username(entry.Username)
		if err != nil {
			return err
		}
		if name != entry.Username {
			return &validate.Error{Field: "username", Message: "Use a lowercase FTP username without surrounding whitespace"}
		}
		if names[entry.Username] {
			return &validate.Error{Field: "username", Message: "FTP usernames must be unique"}
		}
		if !auth.ValidPasswordHash(entry.PasswordHash) {
			return &validate.Error{Field: "passwordHash", Message: "The password hash is invalid"}
		}
		ids[entry.ID], names[entry.Username] = true, true
	}
	return nil
}

func (d *Driver) path(accountID string) string {
	return filepath.Join(d.dir, "accounts", accountID+".json")
}

func (d *Driver) read(accountID, systemUser string) (account, error) {
	for _, path := range []string{d.dir, filepath.Join(d.dir, "accounts")} {
		if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
			return account{ID: accountID, SystemUser: systemUser}, nil
		} else if err != nil {
			return account{}, err
		}
		if err := secureDir(path); err != nil {
			return account{}, err
		}
	}
	snapshot, err := configfile.Take(d.path(accountID))
	if err != nil {
		return account{}, err
	}
	if !snapshot.Exists {
		return account{ID: accountID, SystemUser: systemUser}, nil
	}
	var value account
	if err := json.Unmarshal(snapshot.Data, &value); err != nil {
		return account{}, fmt.Errorf("decode FTP account state")
	}
	if value.ID != accountID || value.SystemUser != systemUser || checkEntries(value.Entries) != nil {
		return account{}, fmt.Errorf("FTP account state has an invalid ownership marker or entries")
	}
	return value, nil
}

func (d *Driver) identity(accountID, systemUser string) (hostuser.Identity, error) {
	if err := hostuser.ValidateNames(accountID, systemUser); err != nil {
		return hostuser.Identity{}, err
	}
	found, err := d.lookup(systemUser)
	if err != nil {
		return hostuser.Identity{}, fmt.Errorf("lookup FTP account identity: %w", err)
	}
	identity, err := hostuser.Validate(found, d.home, accountID, systemUser)
	if err != nil {
		return hostuser.Identity{}, err
	}
	// The jail is always a direct child of the trusted /home directory. Never
	// follow a customer-controlled symlink or accept a path from an API client.
	home, err := fsx.OpenDir(identity.Home)
	if err != nil {
		return hostuser.Identity{}, err
	}
	if err := home.Close(); err != nil {
		return hostuser.Identity{}, err
	}
	return identity, nil
}

func (d *Driver) save(ctx context.Context, value account) error {
	if err := secureDir(d.dir); err != nil {
		return err
	}
	if err := secureDir(filepath.Join(d.dir, "accounts")); err != nil {
		return err
	}
	previous, err := configfile.Take(d.path(value.ID))
	if err != nil {
		return err
	}
	contents, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err := configfile.Write(d.path(value.ID), contents, 0o600); err != nil {
		return err
	}
	// Revoke authentication before terminating sessions. A disconnection failure
	// keeps the new database in place, and a retry disconnects by Account UID.
	if err := d.rebuild(ctx); err != nil {
		return errors.Join(err, previous.Restore())
	}
	identity, err := d.identity(value.ID, value.SystemUser)
	if err != nil {
		return err
	}
	return d.disconnect(ctx, identity.UID)
}

func (d *Driver) rebuild(ctx context.Context) error {
	files, err := os.ReadDir(filepath.Join(d.dir, "accounts"))
	if err != nil {
		return err
	}
	names := make(map[string]bool)
	var lines []string
	for _, file := range files {
		id := strings.TrimSuffix(file.Name(), ".json")
		if file.Name() != id+".json" || validate.ID("accountId", id) != nil {
			return fmt.Errorf("unexpected file in FTP account state")
		}
		value, err := d.read(id, "wcp_"+id[:12])
		if err != nil {
			return err
		}
		for _, entry := range value.Entries {
			if names[entry.Username] {
				return &validate.Error{Field: "username", Message: "The FTP username is already assigned"}
			}
			names[entry.Username] = true
		}
		if value.Suspended || len(value.Entries) == 0 {
			continue
		}
		identity, err := d.identity(value.ID, value.SystemUser)
		if err != nil {
			return err
		}
		for _, entry := range value.Entries {
			if entry.Enabled {
				lines = append(lines, fmt.Sprintf("%s:%s:%d:%d:WEBYCP-%s:%s/./:::::::::::::", entry.Username, entry.PasswordHash, identity.UID, identity.GID, entry.ID, identity.Home))
			}
		}
	}
	sort.Strings(lines)
	stage, err := os.MkdirTemp(d.dir, ".build-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	passwd, database := filepath.Join(stage, "passwd"), filepath.Join(stage, "pureftpd.pdb")
	if err := configfile.Write(passwd, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		return err
	}
	if err := d.run(ctx, "/usr/bin/pure-pw", "mkdb", database, "-f", passwd); err != nil {
		// Do not let subprocess output expose password hashes to jobs or logs.
		return fmt.Errorf("compile FTP authentication database")
	}
	if err := os.Chmod(database, 0o600); err != nil {
		return err
	}
	if err := os.Rename(database, filepath.Join(d.dir, "pureftpd.pdb")); err != nil {
		return fmt.Errorf("activate FTP authentication database: %w", err)
	}
	return nil
}

func secureDir(path string) error {
	if err := configfile.EnsureDir(path, 0o700); err != nil {
		return err
	}
	var info unix.Stat_t
	if err := unix.Lstat(path, &info); err != nil {
		return err
	}
	if info.Mode&0o777 != 0o700 || info.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("FTP configuration directory must be privately owned by the Agent with mode 0700")
	}
	return nil
}
