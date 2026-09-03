package account

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/GVALFER/WEBYCP/internal/agent/hostuser"
	"github.com/GVALFER/WEBYCP/internal/execx"
	"github.com/GVALFER/WEBYCP/internal/fsx"
)

const (
	useraddPath = "/usr/sbin/useradd"
	userdelPath = "/usr/sbin/userdel"
	nologinPath = "/usr/sbin/nologin"
)

type Linux struct {
	lookup      func(string) (*user.User, error)
	lookupGroup func(string) (*user.Group, error)
	run         func(context.Context, string, ...string) error
	home        string
	trash       string
}

func NewLinux() *Linux {
	return &Linux{
		lookup: user.Lookup, lookupGroup: user.LookupGroup,
		run: execx.Run, home: "/home",
		trash: "/var/lib/webycp/account-trash",
	}
}

func (l *Linux) Disable(ctx context.Context, accountID, systemUser string) error {
	_ = ctx
	identity, err := l.identity(accountID, systemUser)
	if err != nil {
		return err
	}
	if err := os.Chmod(identity.Home, 0); err != nil {
		return fmt.Errorf("disable account home: %w", err)
	}
	return nil
}

func (l *Linux) Enable(ctx context.Context, accountID, systemUser string) error {
	_ = ctx
	identity, err := l.identity(accountID, systemUser)
	if err != nil {
		return err
	}
	info, err := os.Lstat(identity.Home)
	if err != nil {
		return fmt.Errorf("inspect account home: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("account home is not a directory")
	}
	if err := os.Chmod(identity.Home, 0o710); err != nil {
		return fmt.Errorf("enable account home: %w", err)
	}
	return l.ensureLayout(identity)
}

func (l *Linux) Delete(ctx context.Context, accountID, systemUser string) error {
	if err := hostuser.ValidateNames(accountID, systemUser); err != nil {
		return err
	}
	found, err := l.lookup(systemUser)
	if err != nil {
		var unknown user.UnknownUserError
		if errors.As(err, &unknown) {
			return l.quarantineHome(accountID, systemUser)
		}
		return fmt.Errorf("lookup system user: %w", err)
	}
	_, err = hostuser.Validate(found, l.home, accountID, systemUser)
	if err != nil {
		return err
	}
	if err := l.run(ctx, userdelPath, "--", systemUser); err != nil {
		return fmt.Errorf("delete system user: %w", err)
	}
	return l.quarantineHome(accountID, systemUser)
}

func (l *Linux) quarantineHome(accountID, systemUser string) error {
	if err := os.MkdirAll(l.trash, 0o700); err != nil {
		return fmt.Errorf("create account trash: %w", err)
	}
	if err := os.Chmod(l.trash, 0o700); err != nil {
		return fmt.Errorf("secure account trash: %w", err)
	}
	target := filepath.Join(l.trash, accountID)
	if _, err := os.Lstat(target); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect account trash target: %w", err)
	}
	if err := os.Rename(filepath.Join(l.home, systemUser), target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("quarantine account home: %w", err)
	}
	return nil
}

func (l *Linux) identity(accountID, systemUser string) (hostuser.Identity, error) {
	if err := hostuser.ValidateNames(accountID, systemUser); err != nil {
		return hostuser.Identity{}, err
	}
	found, err := l.lookup(systemUser)
	if err != nil {
		return hostuser.Identity{}, fmt.Errorf("lookup system user: %w", err)
	}
	return hostuser.Validate(found, l.home, accountID, systemUser)
}

func (l *Linux) Ensure(ctx context.Context, accountID, systemUser string) error {
	if err := hostuser.ValidateNames(accountID, systemUser); err != nil {
		return err
	}
	home := filepath.Join(l.home, systemUser)
	marker := "WEBYCP:" + accountID
	found, err := l.lookup(systemUser)
	if err == nil {
		identity, err := hostuser.Validate(found, l.home, accountID, systemUser)
		if err != nil {
			return err
		}
		return l.ensureLayout(identity)
	}
	var unknown user.UnknownUserError
	if !errors.As(err, &unknown) {
		return fmt.Errorf("lookup system user: %w", err)
	}

	if err := l.run(ctx, useraddPath,
		"--create-home",
		"--home-dir", home,
		"--shell", nologinPath,
		"--comment", marker,
		"--user-group",
		"--", systemUser,
	); err != nil {
		return fmt.Errorf("create system user: %w", err)
	}
	found, err = l.lookup(systemUser)
	if err != nil {
		return fmt.Errorf("verify system user: %w", err)
	}
	identity, err := hostuser.Validate(found, l.home, accountID, systemUser)
	if err != nil {
		return err
	}
	return l.ensureLayout(identity)
}

func (l *Linux) ensureLayout(identity hostuser.Identity) error {
	group, err := l.lookupGroup(hostuser.WebGroup)
	if err != nil {
		return fmt.Errorf("lookup web server group: %w", err)
	}
	webGID, err := strconv.Atoi(group.Gid)
	if err != nil || webGID <= 0 {
		return fmt.Errorf("invalid web server GID")
	}
	root, err := fsx.OpenDir(l.home)
	if err != nil {
		return err
	}
	defer root.Close()
	home, err := root.Open(identity.SystemUser)
	if err != nil {
		return err
	}
	defer home.Close()
	if err := home.Configure(0o710, identity.UID, webGID); err != nil {
		return fmt.Errorf("configure account home: %w", err)
	}
	if err := home.Ensure("web", 0o710, identity.UID, webGID); err != nil {
		return fmt.Errorf("ensure account web directory: %w", err)
	}
	for _, directory := range []struct {
		name string
		mode uint32
	}{
		{name: "logs", mode: 0o750},
		{name: "tmp", mode: 0o700},
	} {
		if err := home.Ensure(directory.name, directory.mode, identity.UID, identity.GID); err != nil {
			return fmt.Errorf("ensure account layout: %w", err)
		}
	}
	return nil
}
