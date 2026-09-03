package domain

import (
	"context"
	"errors"
	"fmt"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/GVALFER/WEBYCP/internal/agent/hostuser"
	agentruntime "github.com/GVALFER/WEBYCP/internal/agent/runtime"
	"github.com/GVALFER/WEBYCP/internal/agent/webserver"
	"github.com/GVALFER/WEBYCP/internal/fsx"
	"github.com/GVALFER/WEBYCP/internal/validate"
	"golang.org/x/sys/unix"
)

type Manager struct {
	lookup      func(string) (*user.User, error)
	lookupGroup func(string) (*user.Group, error)
	home        string
	runtime     agentruntime.Driver
	webserver   webserver.Driver
}

func New(runtimeDriver agentruntime.Driver, webDriver webserver.Driver) *Manager {
	return &Manager{
		lookup: user.Lookup, lookupGroup: user.LookupGroup,
		home: "/home", runtime: runtimeDriver, webserver: webDriver,
	}
}

func (m *Manager) Ensure(
	ctx context.Context,
	accountID, systemUser, domainID, name, phpVersion string,
	aliases []string,
) error {
	if err := validate.ID("domainId", domainID); err != nil {
		return err
	}
	aliases, err := validate.DomainAliases(name, aliases)
	if err != nil {
		return err
	}
	identity, err := m.identity(accountID, systemUser)
	if err != nil {
		return err
	}
	group, err := m.lookupGroup(hostuser.WebGroup)
	if err != nil {
		return fmt.Errorf("lookup web server group: %w", err)
	}
	webGID, err := strconv.Atoi(group.Gid)
	if err != nil || webGID <= 0 {
		return fmt.Errorf("invalid web server GID")
	}

	root, err := fsx.OpenDir(m.home)
	if err != nil {
		return err
	}
	defer root.Close()
	home, err := root.Open(systemUser)
	if err != nil {
		return err
	}
	defer home.Close()
	if err := home.Configure(0o710, identity.UID, webGID); err != nil {
		return fmt.Errorf("configure account home for web server: %w", err)
	}
	web, err := home.Open("web")
	if err != nil {
		return err
	}
	defer web.Close()
	if err := web.Configure(0o710, identity.UID, webGID); err != nil {
		return fmt.Errorf("configure account web directory: %w", err)
	}
	if err := web.Ensure(name, 0o750, identity.UID, webGID); err != nil {
		return fmt.Errorf("ensure domain directory: %w", err)
	}
	domainDir, err := web.Open(name)
	if err != nil {
		return err
	}
	defer domainDir.Close()
	if err := domainDir.Ensure("public_html", 0o2750, identity.UID, webGID); err != nil {
		return fmt.Errorf("ensure public directory: %w", err)
	}

	pool, err := m.runtime.Ensure(ctx, agentruntime.Account{
		ID: accountID, SystemUser: systemUser, Home: identity.Home, Version: phpVersion,
	})
	if err != nil {
		return fmt.Errorf("ensure account runtime: %w", err)
	}

	return m.webserver.Ensure(ctx, webserver.Site{
		ID: domainID, Name: name, Aliases: aliases,
		Root: filepath.Join(identity.Home, "web", name, "public_html"), PHPSocket: pool.Socket,
	})
}

func (m *Manager) Disable(
	ctx context.Context,
	accountID, systemUser, domainID string,
) error {
	if err := validate.ID("domainId", domainID); err != nil {
		return err
	}
	if _, err := m.identity(accountID, systemUser); err != nil {
		return err
	}
	return m.webserver.Disable(ctx, domainID)
}

func (m *Manager) Delete(
	ctx context.Context,
	accountID, systemUser, domainID, name string,
) error {
	if err := validate.ID("domainId", domainID); err != nil {
		return err
	}
	normalized, err := validate.Domain(name)
	if err != nil || normalized != name {
		return &validate.Error{Field: "name", Message: "Domain name is not normalized"}
	}
	identity, err := m.identity(accountID, systemUser)
	if err != nil {
		return err
	}
	if err := m.webserver.Delete(ctx, domainID); err != nil {
		return fmt.Errorf("delete web server site: %w", err)
	}

	root, err := fsx.OpenDir(m.home)
	if err != nil {
		return err
	}
	defer root.Close()
	home, err := root.Open(systemUser)
	if err != nil {
		return err
	}
	defer home.Close()
	web, err := home.Open("web")
	if err != nil {
		return err
	}
	defer web.Close()
	if err := home.Ensure(".webycp-trash", 0o700, identity.UID, identity.GID); err != nil {
		return fmt.Errorf("ensure account trash: %w", err)
	}
	trash, err := home.Open(".webycp-trash")
	if err != nil {
		return err
	}
	defer trash.Close()

	source, err := web.Open(name)
	if err == nil {
		source.Close()
		if err := web.Rename(name, trash, domainID); err != nil {
			return fmt.Errorf("quarantine domain directory: %w", err)
		}
		return nil
	}
	if !errors.Is(err, unix.ENOENT) {
		return err
	}
	quarantined, trashErr := trash.Open(domainID)
	if trashErr == nil {
		quarantined.Close()
		return nil
	}
	if errors.Is(trashErr, unix.ENOENT) {
		return fmt.Errorf("domain directory does not exist: %w", err)
	}
	return trashErr
}

func (m *Manager) Rename(
	ctx context.Context,
	accountID, systemUser, domainID, currentName, name, phpVersion string,
	aliases []string,
) error {
	if err := validate.ID("domainId", domainID); err != nil {
		return err
	}
	current, err := validate.Domain(currentName)
	if err != nil || current != currentName || currentName == name {
		return &validate.Error{Field: "currentName", Message: "Current domain name is invalid"}
	}
	aliases, err = validate.DomainAliases(name, aliases)
	if err != nil {
		return err
	}
	identity, err := m.identity(accountID, systemUser)
	if err != nil {
		return err
	}
	root, err := fsx.OpenDir(m.home)
	if err != nil {
		return err
	}
	defer root.Close()
	home, err := root.Open(systemUser)
	if err != nil {
		return err
	}
	defer home.Close()
	web, err := home.Open("web")
	if err != nil {
		return err
	}
	defer web.Close()
	moved, err := moveDomain(web, currentName, name)
	if err != nil {
		return err
	}
	rollback := func(actionErr error) error {
		if !moved {
			return actionErr
		}
		return errors.Join(actionErr, web.Rename(name, web, currentName))
	}

	pool, err := m.runtime.Ensure(ctx, agentruntime.Account{
		ID: accountID, SystemUser: systemUser, Home: identity.Home, Version: phpVersion,
	})
	if err != nil {
		return rollback(fmt.Errorf("ensure account runtime: %w", err))
	}
	if err := m.webserver.Ensure(ctx, webserver.Site{
		ID: domainID, Name: name, Aliases: aliases,
		Root: filepath.Join(identity.Home, "web", name, "public_html"), PHPSocket: pool.Socket,
	}); err != nil {
		return rollback(err)
	}
	return nil
}

func moveDomain(web *fsx.Dir, currentName, name string) (bool, error) {
	source, err := web.Open(currentName)
	if err == nil {
		source.Close()
		if err := web.Rename(currentName, web, name); err != nil {
			return false, fmt.Errorf("rename domain directory: %w", err)
		}
		return true, nil
	}
	if !errors.Is(err, unix.ENOENT) {
		return false, err
	}
	target, targetErr := web.Open(name)
	if targetErr == nil {
		target.Close()
		return false, nil
	}
	if errors.Is(targetErr, unix.ENOENT) {
		return false, fmt.Errorf("current domain directory does not exist: %w", err)
	}
	return false, targetErr
}

func (m *Manager) identity(accountID, systemUser string) (hostuser.Identity, error) {
	found, err := m.lookup(systemUser)
	if err != nil {
		return hostuser.Identity{}, fmt.Errorf("lookup system user: %w", err)
	}
	return hostuser.Validate(found, m.home, accountID, systemUser)
}
