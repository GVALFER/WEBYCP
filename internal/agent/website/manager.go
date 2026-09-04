package website

import (
	"context"
	"errors"
	"fmt"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/GVALFER/WEBYCP/internal/agent/hostuser"
	agentruntime "github.com/GVALFER/WEBYCP/internal/agent/runtime"
	"github.com/GVALFER/WEBYCP/internal/agent/webserver"
	"github.com/GVALFER/WEBYCP/internal/fsx"
	"github.com/GVALFER/WEBYCP/internal/validate"
	"golang.org/x/sys/unix"
)

type Spec struct {
	AccountID, SystemUser, WebsiteID, DocumentRoot string
	Kind, WebDriver, RuntimeDriver, RuntimeVersion string
	PrimaryDomain                                  string
	Aliases                                        []string
}

type Manager struct {
	lookup      func(string) (*user.User, error)
	lookupGroup func(string) (*user.Group, error)
	home        string
	runtime     agentruntime.Driver
	webserver   webserver.Driver
}

func New(runtimeDriver agentruntime.Driver, webDriver webserver.Driver) *Manager {
	return &Manager{lookup: user.Lookup, lookupGroup: user.LookupGroup, home: "/home", runtime: runtimeDriver, webserver: webDriver}
}

func (m *Manager) Ensure(ctx context.Context, spec Spec) error {
	identity, dir, aliases, err := m.validate(spec)
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
	home, err := root.Open(spec.SystemUser)
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
	if err := web.Ensure(dir, 0o750, identity.UID, webGID); err != nil {
		return fmt.Errorf("ensure website directory: %w", err)
	}
	websiteDir, err := web.Open(dir)
	if err != nil {
		return err
	}
	defer websiteDir.Close()
	if err := websiteDir.Ensure("public_html", 0o2750, identity.UID, webGID); err != nil {
		return fmt.Errorf("ensure document root: %w", err)
	}
	pool, err := m.runtime.Ensure(ctx, agentruntime.Account{ID: spec.AccountID, SystemUser: spec.SystemUser, Home: identity.Home, Version: spec.RuntimeVersion})
	if err != nil {
		return fmt.Errorf("ensure website runtime: %w", err)
	}
	return m.webserver.Ensure(ctx, webserver.Site{ID: spec.WebsiteID, Name: spec.PrimaryDomain, Aliases: aliases, Root: spec.DocumentRoot, PHPSocket: pool.Socket})
}

func (m *Manager) Disable(ctx context.Context, spec Spec) error {
	if _, _, _, err := m.validate(spec); err != nil {
		return err
	}
	return m.webserver.Disable(ctx, spec.WebsiteID)
}

func (m *Manager) Delete(ctx context.Context, spec Spec) error {
	identity, dir, _, err := m.validate(spec)
	if err != nil {
		return err
	}
	if err := m.webserver.Delete(ctx, spec.WebsiteID); err != nil {
		return fmt.Errorf("delete web server site: %w", err)
	}
	root, err := fsx.OpenDir(m.home)
	if err != nil {
		return err
	}
	defer root.Close()
	home, err := root.Open(spec.SystemUser)
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
	source, err := web.Open(dir)
	if err == nil {
		source.Close()
		if err := web.Rename(dir, trash, spec.WebsiteID); err != nil {
			return fmt.Errorf("quarantine website directory: %w", err)
		}
		return nil
	}
	if !errors.Is(err, unix.ENOENT) {
		return err
	}
	quarantined, trashErr := trash.Open(spec.WebsiteID)
	if trashErr == nil {
		quarantined.Close()
		return nil
	}
	if errors.Is(trashErr, unix.ENOENT) {
		return fmt.Errorf("website directory does not exist: %w", err)
	}
	return trashErr
}

func (m *Manager) validate(spec Spec) (hostuser.Identity, string, []string, error) {
	if validate.ID("websiteId", spec.WebsiteID) != nil || spec.Kind != "php" || spec.WebDriver != "nginx" || spec.RuntimeDriver != "phpfpm" || spec.RuntimeVersion != "8.3" {
		return hostuser.Identity{}, "", nil, &validate.Error{Field: "website", Message: "Website stack is invalid"}
	}
	aliases, err := validate.DomainAliases(spec.PrimaryDomain, spec.Aliases)
	if err != nil {
		return hostuser.Identity{}, "", nil, err
	}
	identity, err := m.identity(spec.AccountID, spec.SystemUser)
	if err != nil {
		return hostuser.Identity{}, "", nil, err
	}
	base := filepath.Join(identity.Home, "web")
	root := filepath.Clean(spec.DocumentRoot)
	rel, err := filepath.Rel(base, root)
	if err != nil || rel == "." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return hostuser.Identity{}, "", nil, &validate.Error{Field: "documentRoot", Message: "Document root is outside the account web directory"}
	}
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) != 2 || parts[0] == "" || parts[0] == "." || parts[1] != "public_html" {
		return hostuser.Identity{}, "", nil, &validate.Error{Field: "documentRoot", Message: "Document root is invalid"}
	}
	return identity, parts[0], aliases, nil
}

func (m *Manager) identity(accountID, systemUser string) (hostuser.Identity, error) {
	found, err := m.lookup(systemUser)
	if err != nil {
		return hostuser.Identity{}, fmt.Errorf("lookup system user: %w", err)
	}
	return hostuser.Validate(found, m.home, accountID, systemUser)
}
