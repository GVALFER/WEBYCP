package certbot

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GVALFER/WEBYCP/internal/agent/certificate"
	"github.com/GVALFER/WEBYCP/internal/agent/webserver"
)

type tlsDriver struct {
	site      webserver.Site
	panel     string
	challenge string
}

func (d *tlsDriver) EnsureTLS(_ context.Context, site webserver.Site, _, _ string, _ bool) error {
	d.site = site
	return nil
}

func (d *tlsDriver) EnsurePanelChallenge(_ context.Context, name string) error {
	d.challenge = name
	return nil
}

func (d *tlsDriver) EnsurePanelTLS(_ context.Context, name, _, _ string) error {
	d.panel = name
	return nil
}

func TestIssueFiltersAliasesByPrimaryDNSAndInstallsTLS(t *testing.T) {
	web := &tlsDriver{}
	driver := New(web)
	driver.root = filepath.Join(t.TempDir(), "acme")
	driver.live = filepath.Join(t.TempDir(), "live")
	driver.home = "/home"
	driver.lookup = func(_ context.Context, name string) ([]string, error) {
		if name == "bad.example.net" {
			return []string{"203.0.113.20"}, nil
		}
		return []string{"203.0.113.10"}, nil
	}
	driver.local = func() ([]string, error) { return []string{"203.0.113.10"}, nil }
	expires := time.Now().UTC().Add(80 * 24 * time.Hour).Truncate(time.Second)
	driver.run = func(_ context.Context, _ string, args ...string) error {
		directory := filepath.Join(driver.live, "webycp-0123456789abcdef0123456789abcdef")
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return err
		}
		return writeCertificate(filepath.Join(directory, "fullchain.pem"), expires)
	}
	result, err := driver.Issue(context.Background(), certificate.Request{
		ID: "0123456789abcdef0123456789abcdef", Kind: "domain",
		DomainID: "abcdef0123456789abcdef0123456789", AccountID: "fedcba9876543210fedcba9876543210",
		SystemUser: "wcp_fedcba987654", Name: "example.com", Email: "admin@example.com",
		PHPVersion: "8.3", Names: []string{"example.com", "www.example.com", "bad.example.net"},
		RedirectHTTPS: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Names) != 2 || result.Names[1] != "www.example.com" {
		t.Fatalf("eligible names = %#v", result.Names)
	}
	if !result.ExpiresAt.Equal(expires) || web.site.Name != "example.com" || len(web.site.Aliases) != 1 {
		t.Fatalf("result = %+v, site = %+v", result, web.site)
	}
}

func TestIssueBootstrapsPanelChallenge(t *testing.T) {
	web := &tlsDriver{}
	driver := New(web)
	driver.root = filepath.Join(t.TempDir(), "acme")
	driver.live = filepath.Join(t.TempDir(), "live")
	driver.lookup = func(context.Context, string) ([]string, error) { return []string{"203.0.113.10"}, nil }
	driver.local = func() ([]string, error) { return []string{"203.0.113.10"}, nil }
	driver.run = func(context.Context, string, ...string) error {
		directory := filepath.Join(driver.live, "webycp-0123456789abcdef0123456789abcdef")
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return err
		}
		return writeCertificate(filepath.Join(directory, "fullchain.pem"), time.Now().Add(24*time.Hour))
	}
	_, err := driver.Issue(context.Background(), certificate.Request{ID: "0123456789abcdef0123456789abcdef", Kind: "panel", Name: "panel.example.com", Names: []string{"panel.example.com"}, Email: "admin@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if web.challenge != "panel.example.com" || web.panel != "panel.example.com" {
		t.Fatalf("challenge = %q, panel = %q", web.challenge, web.panel)
	}
}

func TestIssueRejectsPrimaryDNSForAnotherNode(t *testing.T) {
	driver := New(&tlsDriver{})
	driver.lookup = func(context.Context, string) ([]string, error) {
		return []string{"203.0.113.20"}, nil
	}
	driver.local = func() ([]string, error) { return []string{"203.0.113.10"}, nil }
	driver.run = func(context.Context, string, ...string) error {
		t.Fatal("certbot must not run after a failed DNS preflight")
		return nil
	}
	_, err := driver.Issue(context.Background(), certificate.Request{
		ID: "0123456789abcdef0123456789abcdef", Kind: "panel",
		Name: "panel.example.com", Names: []string{"panel.example.com"},
		Email: "admin@example.com",
	})
	if err == nil {
		t.Fatal("expected DNS preflight error")
	}
}

func writeCertificate(path string, expires time.Time) error {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Now().Add(-time.Hour), NotAfter: expires}
	data, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return err
	}
	return os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: data}), 0o600)
}
