package nginx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GVALFER/WEBYCP/internal/agent/configfile"
	"github.com/GVALFER/WEBYCP/internal/agent/webserver"
	"github.com/GVALFER/WEBYCP/internal/execx"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

const (
	defaultAvailable = "/etc/nginx/webycp/sites-available"
	defaultEnabled   = "/etc/nginx/webycp/sites-enabled"
	defaultInclude   = "/etc/nginx/conf.d/webycp.conf"
	nginxPath        = "/usr/sbin/nginx"
	systemctlPath    = "/usr/bin/systemctl"
)

type Driver struct {
	available string
	enabled   string
	include   string
	run       func(context.Context, string, ...string) error
}

func New() *Driver {
	return &Driver{
		available: defaultAvailable,
		enabled:   defaultEnabled,
		include:   defaultInclude,
		run:       execx.Run,
	}
}

func (d *Driver) Ensure(ctx context.Context, site webserver.Site) error {
	if err := validate.ID("domainId", site.ID); err != nil {
		return err
	}
	aliases, err := validate.DomainAliases(site.Name, site.Aliases)
	if err != nil {
		return err
	}
	site.Aliases = aliases
	if err := validPath("root", site.Root); err != nil {
		return err
	}
	if err := validPath("phpSocket", site.PHPSocket); err != nil {
		return err
	}
	if site.Certificate != "" {
		if err := validPath("certificate", site.Certificate); err != nil {
			return err
		}
		if err := validPath("key", site.Key); err != nil {
			return err
		}
	}
	return d.install(ctx, site.ID, render(site))
}

func (d *Driver) EnsureTLS(
	ctx context.Context, site webserver.Site, certificate, key string, redirect bool,
) error {
	site.Certificate = certificate
	site.Key = key
	site.RedirectHTTPS = redirect
	return d.Ensure(ctx, site)
}

func (d *Driver) EnsurePanelChallenge(ctx context.Context, name string) error {
	name, err := validate.Domain(name)
	if err != nil {
		return err
	}
	configPath := filepath.Join(d.available, "panel.conf")
	current, err := os.ReadFile(configPath)
	if err == nil {
		if !strings.HasPrefix(string(current), "# Managed by WEBYCP.\n") {
			return fmt.Errorf("panel nginx configuration is not managed by WEBYCP")
		}
		contents := current
		if !strings.Contains(string(current), "server_name "+name+";") {
			contents = append(append([]byte(nil), current...), renderPanelChallenge(name)...)
		}
		return d.install(ctx, "panel", contents)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read panel nginx configuration: %w", err)
	}
	return d.install(ctx, "panel", renderPanel(name, "", ""))
}

func (d *Driver) EnsurePanelTLS(ctx context.Context, name, certificate, key string) error {
	name, err := validate.Domain(name)
	if err != nil {
		return err
	}
	if err := validPath("certificate", certificate); err != nil {
		return err
	}
	if err := validPath("key", key); err != nil {
		return err
	}
	return d.install(ctx, "panel", renderPanel(name, certificate, key))
}

func (d *Driver) install(ctx context.Context, id string, contents []byte) error {
	if err := configfile.EnsureDir(d.available, 0o755); err != nil {
		return err
	}
	if err := configfile.EnsureDir(d.enabled, 0o755); err != nil {
		return err
	}
	if err := configfile.EnsureDir(filepath.Dir(d.include), 0o755); err != nil {
		return err
	}
	includeCreated, err := ensureInclude(d.include, d.enabled)
	if err != nil {
		return err
	}
	removeInclude := func() error {
		if !includeCreated {
			return nil
		}
		if err := os.Remove(d.include); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove nginx include: %w", err)
		}
		return nil
	}

	configPath := filepath.Join(d.available, id+".conf")
	linkPath := filepath.Join(d.enabled, id+".conf")
	previous, err := configfile.Take(configPath)
	if err != nil {
		return errors.Join(err, removeInclude())
	}
	if err := configfile.Write(configPath, contents, 0o644); err != nil {
		return errors.Join(err, removeInclude())
	}
	linkCreated, err := ensureLink(linkPath, filepath.Join("..", "sites-available", id+".conf"))
	if err != nil {
		return errors.Join(err, previous.Restore(), removeInclude())
	}
	rollback := func() error {
		var linkErr error
		if linkCreated {
			linkErr = os.Remove(linkPath)
		}
		return errors.Join(
			previous.Restore(), linkErr, removeInclude(),
		)
	}

	if err := d.run(ctx, nginxPath, "-t"); err != nil {
		return errors.Join(fmt.Errorf("validate nginx configuration: %w", err), rollback())
	}
	if err := d.run(ctx, systemctlPath, "reload", "nginx"); err != nil {
		rollbackErr := rollback()
		validateErr := d.run(ctx, nginxPath, "-t")
		reloadErr := d.run(ctx, systemctlPath, "reload", "nginx")
		return errors.Join(
			fmt.Errorf("reload nginx: %w", err),
			rollbackErr,
			wrapRecoveryError("validate restored nginx configuration", validateErr),
			wrapRecoveryError("reload restored nginx configuration", reloadErr),
		)
	}
	return nil
}

func (d *Driver) Disable(ctx context.Context, domainID string) error {
	return d.remove(ctx, domainID, false)
}

func (d *Driver) Delete(ctx context.Context, domainID string) error {
	return d.remove(ctx, domainID, true)
}

func (d *Driver) remove(ctx context.Context, domainID string, deleteConfig bool) error {
	if err := validate.ID("domainId", domainID); err != nil {
		return err
	}
	if err := configfile.EnsureDir(d.available, 0o755); err != nil {
		return err
	}
	if err := configfile.EnsureDir(d.enabled, 0o755); err != nil {
		return err
	}
	configPath := filepath.Join(d.available, domainID+".conf")
	linkPath := filepath.Join(d.enabled, domainID+".conf")
	target := filepath.Join("..", "sites-available", domainID+".conf")
	previous, err := configfile.Take(configPath)
	if err != nil {
		return err
	}
	linkExists, err := inspectLink(linkPath, target)
	if err != nil {
		return err
	}
	if !linkExists && (!deleteConfig || !previous.Exists) {
		return nil
	}
	if linkExists {
		if err := os.Remove(linkPath); err != nil {
			return fmt.Errorf("disable nginx site: %w", err)
		}
	}
	if deleteConfig && previous.Exists {
		if err := os.Remove(configPath); err != nil {
			return errors.Join(fmt.Errorf("delete nginx site: %w", err), restoreLink(linkPath, target, linkExists))
		}
	}
	if !linkExists {
		return nil
	}
	rollback := func() error {
		var configErr error
		if deleteConfig {
			configErr = previous.Restore()
		}
		return errors.Join(configErr, restoreLink(linkPath, target, true))
	}
	if err := d.run(ctx, nginxPath, "-t"); err != nil {
		return errors.Join(fmt.Errorf("validate nginx configuration: %w", err), rollback())
	}
	if err := d.run(ctx, systemctlPath, "reload", "nginx"); err != nil {
		rollbackErr := rollback()
		validateErr := d.run(ctx, nginxPath, "-t")
		reloadErr := d.run(ctx, systemctlPath, "reload", "nginx")
		return errors.Join(
			fmt.Errorf("reload nginx: %w", err),
			rollbackErr,
			wrapRecoveryError("validate restored nginx configuration", validateErr),
			wrapRecoveryError("reload restored nginx configuration", reloadErr),
		)
	}
	return nil
}

func ensureInclude(path, enabled string) (bool, error) {
	contents := []byte("include " + filepath.Join(enabled, "*.conf") + ";\n")
	snapshot, err := configfile.Take(path)
	if err != nil {
		return false, err
	}
	if !snapshot.Exists {
		if err := configfile.Write(path, contents, 0o644); err != nil {
			return false, err
		}
		return true, nil
	}
	if string(snapshot.Data) != string(contents) {
		return false, fmt.Errorf("nginx include has unexpected contents: %s", path)
	}
	return false, nil
}

func wrapRecoveryError(message string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

func render(site webserver.Site) []byte {
	names := strings.Join(append([]string{site.Name}, site.Aliases...), " ")
	if site.Certificate != "" {
		redirect := ""
		if site.RedirectHTTPS {
			redirect = "\n    location / {\n        return 301 https://$host$request_uri;\n    }\n"
		} else {
			redirect = renderLocations(site.PHPSocket)
		}
		return []byte(fmt.Sprintf(`server {
    listen 80;
    listen [::]:80;
    server_name %s;

    location ^~ /.well-known/acme-challenge/ {
        root /var/lib/webycp/acme;
        try_files $uri =404;
    }
%s}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    server_name %s;
    root %s;
    index index.php index.html;

    ssl_certificate %s;
    ssl_certificate_key %s;
    ssl_protocols TLSv1.2 TLSv1.3;

    access_log /var/log/nginx/webycp-%s.access.log;
    error_log /var/log/nginx/webycp-%s.error.log;
%s}
`, names, redirect, names, site.Root, site.Certificate, site.Key, site.ID, site.ID,
			renderLocations(site.PHPSocket)))
	}
	return []byte(fmt.Sprintf(`server {
    listen 80;
    listen [::]:80;

    server_name %s;
    root %s;
    index index.php index.html;

    access_log /var/log/nginx/webycp-%s.access.log;
    error_log /var/log/nginx/webycp-%s.error.log;

    location ^~ /.well-known/acme-challenge/ {
        root /var/lib/webycp/acme;
        try_files $uri =404;
    }

    location / {
        try_files $uri $uri/ =404;
    }

    location ~ /\. {
        deny all;
    }

    location ~ \.php$ {
        try_files $uri =404;
        include fastcgi_params;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        fastcgi_pass unix:%s;
    }
}
`, names, site.Root, site.ID, site.ID, site.PHPSocket))
}

func renderLocations(socket string) string {
	return fmt.Sprintf(`
    location / {
        try_files $uri $uri/ =404;
    }

    location ~ /\. {
        deny all;
    }

    location ~ \.php$ {
        try_files $uri =404;
        include fastcgi_params;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        fastcgi_pass unix:%s;
    }
`, socket)
}

func renderPanel(name, certificate, key string) []byte {
	tls := ""
	httpHandler := `proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto http;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;`
	if certificate != "" {
		httpHandler = "return 301 https://$host$request_uri;"
		tls = fmt.Sprintf(`
server {
    listen 443 ssl;
    listen [::]:443 ssl;
    server_name %s;
    ssl_certificate %s;
    ssl_certificate_key %s;
    ssl_protocols TLSv1.2 TLSv1.3;
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto https;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
`, name, certificate, key)
	}
	return []byte(fmt.Sprintf(`# Managed by WEBYCP.
server {
    listen 80;
    listen [::]:80;
    server_name %s;
    location ^~ /.well-known/acme-challenge/ {
        root /var/lib/webycp/acme;
        try_files $uri =404;
    }
    location / {
        %s
    }
}
%s`, name, httpHandler, tls))
}

func renderPanelChallenge(name string) []byte {
	return []byte(fmt.Sprintf(`
server {
    listen 80;
    listen [::]:80;
    server_name %s;
    location ^~ /.well-known/acme-challenge/ {
        root /var/lib/webycp/acme;
        try_files $uri =404;
    }
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto http;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
`, name))
}

func validPath(field, path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsAny(path, " \t\r\n;{}") {
		return &validate.Error{Field: field, Message: "Site path is invalid"}
	}
	return nil
}

func ensureLink(path, target string) (bool, error) {
	exists, err := inspectLink(path, target)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	if err := os.Symlink(target, path); err != nil {
		return false, fmt.Errorf("enable nginx site: %w", err)
	}
	return true, nil
}

func inspectLink(path, target string) (bool, error) {
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			return false, fmt.Errorf("nginx enabled path is not a symlink: %s", path)
		}
		current, err := os.Readlink(path)
		if err != nil {
			return false, fmt.Errorf("read nginx symlink: %w", err)
		}
		if current != target {
			return false, fmt.Errorf("nginx symlink has unexpected target")
		}
		return true, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("inspect nginx symlink: %w", err)
	}
	return false, nil
}

func restoreLink(path, target string, restore bool) error {
	if !restore {
		return nil
	}
	if _, err := ensureLink(path, target); err != nil {
		return fmt.Errorf("restore nginx site: %w", err)
	}
	return nil
}
