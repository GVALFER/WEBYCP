package certbot

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/GVALFER/WEBYCP/internal/agent/certificate"
	"github.com/GVALFER/WEBYCP/internal/agent/webserver"
	"github.com/GVALFER/WEBYCP/internal/execx"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

const (
	certbotPath = "/usr/bin/certbot"
	webrootPath = "/var/lib/webycp/acme"
	livePath    = "/etc/letsencrypt/live"
)

type TLSDriver interface {
	EnsureTLS(context.Context, webserver.Site, string, string, bool) error
	EnsurePanelChallenge(context.Context, string) error
	EnsurePanelTLS(context.Context, string, string, string) error
}

type Driver struct {
	lookup func(context.Context, string) ([]string, error)
	local  func() ([]string, error)
	run    func(context.Context, string, ...string) error
	web    TLSDriver
	home   string
	live   string
	root   string
}

func New(web TLSDriver) *Driver {
	resolver := net.DefaultResolver
	return &Driver{
		lookup: resolver.LookupHost, local: localAddresses, run: execx.Run, web: web,
		home: "/home", live: livePath, root: webrootPath,
	}
}

func (d *Driver) Issue(ctx context.Context, request certificate.Request) (certificate.Result, error) {
	if validate.ID("certificateId", request.ID) != nil || (request.Kind != "website" && request.Kind != "panel") {
		return certificate.Result{}, &validate.Error{Field: "certificateId", Message: "The certificate identity is invalid"}
	}
	name, err := validate.Domain(request.Name)
	if err != nil || name != request.Name {
		return certificate.Result{}, &validate.Error{Field: "name", Message: "The certificate name is invalid"}
	}
	if _, err := validate.Email(request.Email); err != nil {
		return certificate.Result{}, err
	}
	eligible, err := d.preflight(ctx, request.Name, request.Names)
	if err != nil {
		return certificate.Result{}, err
	}
	if err := os.MkdirAll(d.root, 0o755); err != nil {
		return certificate.Result{}, fmt.Errorf("create ACME webroot: %w", err)
	}
	if request.Kind == "panel" {
		if d.web == nil {
			return certificate.Result{}, fmt.Errorf("panel TLS driver is not configured")
		}
		if err := d.web.EnsurePanelChallenge(ctx, request.Name); err != nil {
			return certificate.Result{}, fmt.Errorf("install panel ACME challenge route: %w", err)
		}
	}
	certName := "webycp-" + request.ID
	args := []string{
		"certonly", "--non-interactive", "--agree-tos", "--preferred-challenges", "http",
		"--webroot", "--webroot-path", d.root, "--cert-name", certName,
		"--email", request.Email, "--keep-until-expiring",
	}
	for _, item := range eligible {
		args = append(args, "--domain", item)
	}
	if err := d.run(ctx, certbotPath, args...); err != nil {
		return certificate.Result{}, fmt.Errorf("issue ACME certificate: %w", err)
	}
	directory := filepath.Join(d.live, certName)
	fullchain := filepath.Join(directory, "fullchain.pem")
	privateKey := filepath.Join(directory, "privkey.pem")
	expires, err := readExpiry(fullchain)
	if err != nil {
		return certificate.Result{}, err
	}
	if request.Kind == "panel" {
		err = d.web.EnsurePanelTLS(ctx, request.Name, fullchain, privateKey)
	} else {
		if validate.ID("websiteId", request.WebsiteID) != nil || validate.ID("accountId", request.AccountID) != nil || validate.SystemUser(request.SystemUser) != nil || request.SystemUser != "wcp_"+request.AccountID[:12] || request.RuntimeVersion != "8.3" {
			return certificate.Result{}, &validate.Error{Field: "websiteId", Message: "The hosted website identity is invalid"}
		}
		base := filepath.Join(d.home, request.SystemUser, "web")
		rel, relErr := filepath.Rel(base, filepath.Clean(request.DocumentRoot))
		if relErr != nil || rel == "." || filepath.IsAbs(rel) || rel == ".." || len(rel) < len("x/public_html") || filepath.Base(rel) != "public_html" || filepath.Dir(rel) == "." || filepath.Dir(filepath.Dir(rel)) != "." {
			return certificate.Result{}, &validate.Error{Field: "documentRoot", Message: "The website document root is invalid"}
		}
		site := webserver.Site{
			ID: request.WebsiteID, Name: request.Name, Aliases: eligible[1:],
			Root:      request.DocumentRoot,
			PHPSocket: filepath.Join("/run/php", "webycp-8.3-"+request.AccountID+".sock"),
		}
		err = d.web.EnsureTLS(ctx, site, fullchain, privateKey, request.RedirectHTTPS)
	}
	if err != nil {
		return certificate.Result{}, fmt.Errorf("install certificate: %w", err)
	}
	return certificate.Result{Names: eligible, ExpiresAt: expires}, nil
}

func (d *Driver) preflight(ctx context.Context, primary string, names []string) ([]string, error) {
	primaryIPs, err := d.lookup(ctx, primary)
	if err != nil || len(primaryIPs) == 0 {
		return nil, fmt.Errorf("DNS preflight failed for %s", primary)
	}
	localIPs, err := d.local()
	if err != nil || len(localIPs) == 0 {
		if err != nil {
			return nil, fmt.Errorf("inspect managed node addresses: %w", err)
		}
		return nil, fmt.Errorf("inspect managed node addresses: no global unicast addresses found")
	}
	addresses := make(map[string]struct{}, len(localIPs))
	for _, value := range localIPs {
		if parsed := net.ParseIP(value); parsed != nil {
			addresses[parsed.String()] = struct{}{}
		}
	}
	if !hasAddress(primaryIPs, addresses) {
		return nil, fmt.Errorf("DNS preflight failed: %s does not resolve to this node", primary)
	}
	eligible := []string{primary}
	seen := map[string]bool{primary: true}
	for _, name := range names {
		if seen[name] {
			continue
		}
		normalized, domainErr := validate.Domain(name)
		if domainErr != nil || normalized != name {
			continue
		}
		values, lookupErr := d.lookup(ctx, name)
		if lookupErr != nil {
			continue
		}
		if hasAddress(values, addresses) {
			eligible = append(eligible, name)
			seen[name] = true
		}
	}
	sort.Strings(eligible[1:])
	return eligible, nil
}

func hasAddress(values []string, allowed map[string]struct{}) bool {
	for _, value := range values {
		parsed := net.ParseIP(value)
		if parsed == nil {
			continue
		}
		if _, ok := allowed[parsed.String()]; ok {
			return true
		}
	}
	return false
}

func localAddresses() ([]string, error) {
	values, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		ip, _, err := net.ParseCIDR(value.String())
		if err == nil && ip.IsGlobalUnicast() {
			result = append(result, ip.String())
		}
	}
	return result, nil
}

func readExpiry(path string) (time.Time, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, fmt.Errorf("read issued certificate: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return time.Time{}, fmt.Errorf("issued certificate PEM is invalid")
	}
	parsed, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse issued certificate: %w", err)
	}
	return parsed.NotAfter.UTC(), nil
}
