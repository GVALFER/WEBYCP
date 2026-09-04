package nginx

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GVALFER/WEBYCP/internal/agent/webserver"
)

const testDomainID = "0123456789abcdef0123456789abcdef"

func TestEnsureInstallsValidatedSite(t *testing.T) {
	driver := testDriver(t)
	var commands []string
	driver.run = func(_ context.Context, name string, args ...string) error {
		commands = append(commands, name+" "+strings.Join(args, " "))
		return nil
	}
	site := testSite()
	if err := driver.Ensure(context.Background(), site); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(driver.available, testDomainID+".conf")
	contents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	config := string(contents)
	if !strings.Contains(config, "server_name example.com cdn.example.com www.example.com;") ||
		!strings.Contains(config, "root "+site.Root+";") {
		t.Fatalf("unexpected config:\n%s", config)
	}
	if !strings.Contains(config, "fastcgi_pass unix:"+site.PHPSocket+";") ||
		!strings.Contains(config, "fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;") {
		t.Fatalf("PHP handler is missing:\n%s", config)
	}
	include, err := os.ReadFile(driver.include)
	if err != nil {
		t.Fatal(err)
	}
	if string(include) != "include "+filepath.Join(driver.enabled, "*.conf")+";\n" {
		t.Fatalf("unexpected include: %q", include)
	}
	target, err := os.Readlink(filepath.Join(driver.enabled, testDomainID+".conf"))
	if err != nil {
		t.Fatal(err)
	}
	if target != filepath.Join("..", "sites-available", testDomainID+".conf") {
		t.Fatalf("symlink target = %q", target)
	}
	want := []string{nginxPath + " -t", systemctlPath + " reload nginx"}
	if strings.Join(commands, "|") != strings.Join(want, "|") {
		t.Fatalf("commands = %#v, want %#v", commands, want)
	}
}

func TestEnsureRestoresKnownGoodConfigWhenValidationFails(t *testing.T) {
	driver := testDriver(t)
	configPath := filepath.Join(driver.available, testDomainID+".conf")
	if err := os.WriteFile(configPath, []byte("known-good\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(driver.enabled, testDomainID+".conf")
	if err := os.Symlink(filepath.Join("..", "sites-available", testDomainID+".conf"), linkPath); err != nil {
		t.Fatal(err)
	}
	driver.run = func(_ context.Context, name string, _ ...string) error {
		if name == nginxPath {
			return errors.New("invalid config")
		}
		return nil
	}

	err := driver.Ensure(context.Background(), testSite())
	if err == nil {
		t.Fatal("expected validation error")
	}
	contents, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(contents) != "known-good\n" {
		t.Fatalf("config was not restored: %q", contents)
	}
}

func TestEnsureRemovesNewSiteWhenReloadFails(t *testing.T) {
	driver := testDriver(t)
	reloads := 0
	driver.run = func(_ context.Context, name string, _ ...string) error {
		if name == systemctlPath {
			reloads++
			if reloads == 1 {
				return errors.New("reload failed")
			}
		}
		return nil
	}

	err := driver.Ensure(context.Background(), testSite())
	if err == nil {
		t.Fatal("expected reload error")
	}
	for _, path := range []string{
		filepath.Join(driver.available, testDomainID+".conf"),
		filepath.Join(driver.enabled, testDomainID+".conf"),
		driver.include,
	} {
		if _, statErr := os.Lstat(path); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("rollback left %s: %v", path, statErr)
		}
	}
	if reloads != 2 {
		t.Fatalf("reloads = %d, want recovery reload", reloads)
	}
}

func TestEnsureIsIdempotent(t *testing.T) {
	driver := testDriver(t)
	driver.run = func(context.Context, string, ...string) error { return nil }
	for range 2 {
		if err := driver.Ensure(context.Background(), testSite()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEnsureTLSAddsHTTPSAndRedirect(t *testing.T) {
	driver := testDriver(t)
	driver.run = func(context.Context, string, ...string) error { return nil }
	site := testSite()
	if err := driver.EnsureTLS(
		context.Background(), site,
		"/etc/letsencrypt/live/example/fullchain.pem",
		"/etc/letsencrypt/live/example/privkey.pem", true,
	); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(driver.available, testDomainID+".conf"))
	if err != nil {
		t.Fatal(err)
	}
	config := string(data)
	for _, expected := range []string{
		"listen 443 ssl;", "ssl_certificate /etc/letsencrypt/live/example/fullchain.pem;",
		"return 301 https://$host$request_uri;", "location ^~ /.well-known/acme-challenge/",
	} {
		if !strings.Contains(config, expected) {
			t.Fatalf("TLS config is missing %q:\n%s", expected, config)
		}
	}
}

func TestPanelTLSRedirectsHTTPButKeepsChallenge(t *testing.T) {
	driver := testDriver(t)
	driver.run = func(context.Context, string, ...string) error { return nil }
	if err := driver.EnsurePanelTLS(
		context.Background(), "panel.example.com",
		"/etc/letsencrypt/live/panel/fullchain.pem",
		"/etc/letsencrypt/live/panel/privkey.pem",
	); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(driver.available, "panel.conf"))
	if err != nil {
		t.Fatal(err)
	}
	config := string(data)
	for _, expected := range []string{
		"location ^~ /.well-known/acme-challenge/",
		"return 301 https://$host$request_uri;",
		"listen 443 ssl;",
	} {
		if !strings.Contains(config, expected) {
			t.Fatalf("panel TLS config is missing %q:\n%s", expected, config)
		}
	}
}

func TestPanelChallengeKeepsExistingTLSDuringRenewal(t *testing.T) {
	driver := testDriver(t)
	driver.run = func(context.Context, string, ...string) error { return nil }
	if err := driver.EnsurePanelTLS(
		context.Background(), "panel.example.com",
		"/etc/letsencrypt/live/panel/fullchain.pem",
		"/etc/letsencrypt/live/panel/privkey.pem",
	); err != nil {
		t.Fatal(err)
	}
	if err := driver.EnsurePanelChallenge(context.Background(), "panel.example.com"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(driver.available, "panel.conf"))
	if err != nil {
		t.Fatal(err)
	}
	config := string(data)
	if !strings.Contains(config, "listen 443 ssl;") ||
		!strings.Contains(config, "ssl_certificate /etc/letsencrypt/live/panel/fullchain.pem;") {
		t.Fatalf("panel TLS was removed during renewal:\n%s", config)
	}
}

func TestPanelChallengeKeepsHTTPPanelAvailable(t *testing.T) {
	driver := testDriver(t)
	driver.run = func(context.Context, string, ...string) error { return nil }
	bootstrap := []byte("# Managed by WEBYCP.\nserver { listen 8443 ssl; server_name _; }\n")
	if err := os.WriteFile(filepath.Join(driver.available, "panel.conf"), bootstrap, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := driver.EnsurePanelChallenge(context.Background(), "panel.example.com"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(driver.available, "panel.conf"))
	if err != nil {
		t.Fatal(err)
	}
	config := string(data)
	for _, expected := range []string{
		string(bootstrap),
		"server_name panel.example.com;",
		"location ^~ /.well-known/acme-challenge/",
		"proxy_pass http://127.0.0.1:3000;",
	} {
		if !strings.Contains(config, expected) {
			t.Fatalf("panel challenge config is missing %q:\n%s", expected, config)
		}
	}
}

func TestDisableKeepsConfigAndRemovesLink(t *testing.T) {
	driver := installedDriver(t)
	if err := driver.Disable(context.Background(), testDomainID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(driver.available, testDomainID+".conf")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(driver.enabled, testDomainID+".conf")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("enabled site still exists: %v", err)
	}
}

func TestDisableRestoresLinkWhenReloadFails(t *testing.T) {
	driver := installedDriver(t)
	reloads := 0
	driver.run = func(_ context.Context, name string, _ ...string) error {
		if name == systemctlPath {
			reloads++
			if reloads == 1 {
				return errors.New("reload failed")
			}
		}
		return nil
	}
	if err := driver.Disable(context.Background(), testDomainID); err == nil {
		t.Fatal("expected reload error")
	}
	if _, err := os.Lstat(filepath.Join(driver.enabled, testDomainID+".conf")); err != nil {
		t.Fatal(err)
	}
	if reloads != 2 {
		t.Fatalf("reloads = %d, want recovery reload", reloads)
	}
}

func TestDeleteRemovesConfigAndLink(t *testing.T) {
	driver := installedDriver(t)
	if err := driver.Delete(context.Background(), testDomainID); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(driver.available, testDomainID+".conf"),
		filepath.Join(driver.enabled, testDomainID+".conf"),
	} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("deleted site left %s: %v", path, err)
		}
	}
	if err := driver.Delete(context.Background(), testDomainID); err != nil {
		t.Fatalf("idempotent delete: %v", err)
	}
}

func TestDeleteRestoresSiteWhenReloadFails(t *testing.T) {
	driver := installedDriver(t)
	reloads := 0
	driver.run = func(_ context.Context, name string, _ ...string) error {
		if name == systemctlPath {
			reloads++
			if reloads == 1 {
				return errors.New("reload failed")
			}
		}
		return nil
	}
	if err := driver.Delete(context.Background(), testDomainID); err == nil {
		t.Fatal("expected reload error")
	}
	if _, err := os.Stat(filepath.Join(driver.available, testDomainID+".conf")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(driver.enabled, testDomainID+".conf")); err != nil {
		t.Fatal(err)
	}
}

func testDriver(t *testing.T) *Driver {
	t.Helper()
	root := t.TempDir()
	driver := New()
	driver.available = filepath.Join(root, "sites-available")
	driver.enabled = filepath.Join(root, "sites-enabled")
	driver.include = filepath.Join(root, "conf.d", "webycp.conf")
	if err := os.MkdirAll(driver.available, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(driver.enabled, 0o755); err != nil {
		t.Fatal(err)
	}
	return driver
}

func installedDriver(t *testing.T) *Driver {
	t.Helper()
	driver := testDriver(t)
	driver.run = func(context.Context, string, ...string) error { return nil }
	if err := driver.Ensure(context.Background(), testSite()); err != nil {
		t.Fatal(err)
	}
	return driver
}

func testSite() webserver.Site {
	return webserver.Site{
		ID: testDomainID, Name: "example.com",
		Aliases:   []string{"www.example.com", "cdn.example.com"},
		Root:      "/home/wcp_0123456789ab/web/example.com/public_html",
		PHPSocket: "/run/php/webycp-8.3-0123456789abcdef0123456789abcdef.sock",
	}
}
