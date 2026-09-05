package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	agentbackup "github.com/GVALFER/WEBYCP/internal/agent/backup"
	agentapi "github.com/GVALFER/WEBYCP/internal/agent/protocol"
	"github.com/GVALFER/WEBYCP/internal/backupfmt"
	"github.com/GVALFER/WEBYCP/internal/certificates"
	cronjob "github.com/GVALFER/WEBYCP/internal/cron"
	dnscontrol "github.com/GVALFER/WEBYCP/internal/dns"
	"github.com/GVALFER/WEBYCP/internal/services"
	"github.com/GVALFER/WEBYCP/internal/websites"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

const protocolVersion = "v1"

type Client struct {
	timeout time.Duration
}

func New(timeout time.Duration) *Client {
	return &Client{timeout: timeout}
}

func (c *Client) Probe(ctx context.Context, socket string) (services.Capabilities, error) {
	response, cleanup, err := c.do(ctx, socket, http.MethodGet, "/agent/v1/health", nil)
	if err != nil {
		return services.Capabilities{}, fmt.Errorf("request agent health: %w", err)
	}
	defer cleanup()
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
		return services.Capabilities{}, fmt.Errorf("agent health returned status %d", response.StatusCode)
	}

	var health agentapi.HealthResponse
	decoder := json.NewDecoder(io.LimitReader(response.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&health); err != nil {
		return services.Capabilities{}, fmt.Errorf("decode agent health: %w", err)
	}
	if health.Service != "webycp-agent" || health.Status != agentapi.Ok {
		return services.Capabilities{}, fmt.Errorf("agent health response is invalid")
	}
	if health.ProtocolVersion != protocolVersion {
		return services.Capabilities{}, fmt.Errorf("agent protocol %q is not supported", health.ProtocolVersion)
	}

	return capabilitiesValue(health.Capabilities), nil
}

func capabilitiesValue(value agentapi.ServiceCapabilities) services.Capabilities {
	return services.Capabilities{
		Webservers: capabilityValues(value.Webservers),
		Runtimes:   capabilityValues(value.Runtimes),
		Databases:  capabilityValues(value.Databases),
		Schedulers: capabilityValues(value.Schedulers),
		Backups:    capabilityValues(value.Backups),
		DNS:        capabilityValues(value.Dns),
	}
}

func capabilityValues(values []agentapi.ServiceCapability) []services.Capability {
	result := make([]services.Capability, 0, len(values))
	for _, value := range values {
		result = append(result, services.Capability{
			Driver: value.Driver, Version: value.Version, Status: string(value.Status),
		})
	}
	return result
}

func (c *Client) EnsureAccount(ctx context.Context, socket, accountID, systemUser string) error {
	return c.postJSON(ctx, socket, "/agent/v1/accounts", agentapi.EnsureAccountRequest{
		AccountId: accountID, SystemUser: systemUser,
	}, "account reconcile")
}

func (c *Client) DisableAccount(ctx context.Context, socket, accountID, systemUser string) error {
	return c.accountAction(ctx, socket, "/agent/v1/accounts/disable", accountID, systemUser)
}

func (c *Client) EnableAccount(ctx context.Context, socket, accountID, systemUser string) error {
	return c.accountAction(ctx, socket, "/agent/v1/accounts/enable", accountID, systemUser)
}

func (c *Client) DeleteAccount(ctx context.Context, socket, accountID, systemUser string) error {
	return c.accountAction(ctx, socket, "/agent/v1/accounts/delete", accountID, systemUser)
}

func (c *Client) accountAction(
	ctx context.Context, socket, path, accountID, systemUser string,
) error {
	return c.postJSON(ctx, socket, path, agentapi.AccountActionRequest{
		AccountId: accountID, SystemUser: systemUser,
	}, "account action")
}

func (c *Client) EnsureWebsite(ctx context.Context, socket string, spec websites.Spec) error {
	return c.postJSON(ctx, socket, "/agent/v1/websites", websiteRequest(spec), "website reconcile")
}

func (c *Client) DisableWebsite(ctx context.Context, socket string, spec websites.Spec) error {
	return c.postJSON(ctx, socket, "/agent/v1/websites/disable", websiteRequest(spec), "website disable")
}

func (c *Client) DeleteWebsite(ctx context.Context, socket string, spec websites.Spec) error {
	return c.postJSON(ctx, socket, "/agent/v1/websites/delete", websiteRequest(spec), "website delete")
}

func websiteRequest(spec websites.Spec) agentapi.WebsiteRequest {
	return agentapi.WebsiteRequest{
		AccountId: spec.AccountID, SystemUser: spec.SystemUser, WebsiteId: spec.WebsiteID,
		DocumentRoot: spec.DocumentRoot, Kind: agentapi.WebsiteRequestKind(spec.Kind),
		WebDriver:      agentapi.WebsiteRequestWebDriver(spec.WebDriver),
		RuntimeDriver:  agentapi.WebsiteRequestRuntimeDriver(spec.RuntimeDriver),
		RuntimeVersion: agentapi.WebsiteRequestRuntimeVersion(spec.RuntimeVersion),
		PrimaryDomain:  spec.PrimaryDomain, Aliases: spec.Aliases,
	}
}

func (c *Client) postJSON(
	ctx context.Context,
	socket, path string,
	request any,
	operation string,
) error {
	return c.sendJSON(ctx, socket, http.MethodPost, path, request, operation)
}

func (c *Client) sendJSON(
	ctx context.Context, socket, method, path string, request any, operation string,
) error {
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encode %s request: %w", operation, err)
	}
	response, cleanup, err := c.do(ctx, socket, method, path, body)
	if err != nil {
		return fmt.Errorf("request %s: %w", operation, err)
	}
	return expectNoContent(response, cleanup, operation)
}

func (c *Client) EnsureDatabase(ctx context.Context, socket, name string) error {
	return c.postJSON(ctx, socket, "/agent/v1/databases", agentapi.DatabaseRequest{Name: name}, "database create")
}

func (c *Client) DeleteDatabase(ctx context.Context, socket, name string) error {
	return c.sendJSON(ctx, socket, http.MethodDelete, "/agent/v1/databases", agentapi.DatabaseRequest{Name: name}, "database delete")
}

func (c *Client) EnsureDatabaseUser(ctx context.Context, socket, name, password string) error {
	return c.postJSON(ctx, socket, "/agent/v1/database-users", agentapi.DatabaseUserRequest{Name: name, Password: &password}, "database user create")
}

func (c *Client) DeleteDatabaseUser(ctx context.Context, socket, name string) error {
	return c.sendJSON(ctx, socket, http.MethodDelete, "/agent/v1/database-users", agentapi.DatabaseUserRequest{Name: name}, "database user delete")
}

func (c *Client) EnsureDatabaseGrant(ctx context.Context, socket, database, user string) error {
	return c.postJSON(ctx, socket, "/agent/v1/database-grants", agentapi.DatabaseGrantRequest{Database: database, User: user}, "database grant create")
}

func (c *Client) DeleteDatabaseGrant(ctx context.Context, socket, database, user string) error {
	return c.sendJSON(ctx, socket, http.MethodDelete, "/agent/v1/database-grants", agentapi.DatabaseGrantRequest{Database: database, User: user}, "database grant delete")
}

func (c *Client) SyncCron(ctx context.Context, socket, accountID, systemUser string, entries []cronjob.Entry) error {
	values := make([]agentapi.CronEntry, 0, len(entries))
	for _, entry := range entries {
		values = append(values, agentapi.CronEntry{Id: entry.ID, Schedule: entry.Schedule, Command: entry.Command})
	}
	return c.sendJSON(ctx, socket, http.MethodPut, "/agent/v1/cron", agentapi.SyncCronRequest{
		AccountId: accountID, SystemUser: systemUser, Entries: values,
	}, "cron sync")
}

func (c *Client) EnsureDNSZone(ctx context.Context, socket string, value dnscontrol.ZoneSpec) error {
	return c.postJSON(ctx, socket, "/agent/v1/dns/zones", dnsZoneRequest(value), "DNS zone create")
}

func (c *Client) DeleteDNSZone(ctx context.Context, socket string, value dnscontrol.ZoneSpec) error {
	return c.sendJSON(ctx, socket, http.MethodDelete, "/agent/v1/dns/zones", dnsZoneRequest(value), "DNS zone delete")
}

func (c *Client) SyncDNSRecordSets(
	ctx context.Context, socket string, zone dnscontrol.ZoneSpec, values []dnscontrol.RecordSet,
) error {
	sets := make([]agentapi.DNSRecordSet, 0, len(values))
	for _, value := range values {
		sets = append(sets, agentapi.DNSRecordSet{
			Name: value.Name, Type: agentapi.DNSRecordSetType(value.Type),
			Ttl: value.TTL, Records: value.Records,
		})
	}
	return c.sendJSON(ctx, socket, http.MethodPut, "/agent/v1/dns/records", agentapi.SyncDNSRecordSetsRequest{
		Zone: dnsZoneRequest(zone), Rrsets: sets,
	}, "DNS record sync")
}

func dnsZoneRequest(value dnscontrol.ZoneSpec) agentapi.DNSZoneRequest {
	return agentapi.DNSZoneRequest{Id: value.ID, Name: value.Name, Nameservers: value.Nameservers}
}

func (c *Client) IssueCertificate(ctx context.Context, socket string, request certificates.Request) (certificates.Result, error) {
	body := agentapi.IssueCertificateRequest{
		CertificateId: request.CertificateID, Kind: agentapi.IssueCertificateRequestKind(request.Kind),
		Name: request.Name, Names: request.Names, Email: openapi_types.Email(request.Email), RedirectHttps: request.RedirectHTTPS,
	}
	if request.WebsiteID != "" {
		body.WebsiteId = &request.WebsiteID
	}
	if request.AccountID != "" {
		body.AccountId = &request.AccountID
	}
	if request.SystemUser != "" {
		body.SystemUser = &request.SystemUser
	}
	if request.DocumentRoot != "" {
		body.DocumentRoot = &request.DocumentRoot
	}
	if request.RuntimeVersion != "" {
		version := agentapi.IssueCertificateRequestRuntimeVersion(request.RuntimeVersion)
		body.RuntimeVersion = &version
	}
	var result agentapi.CertificateResult
	if err := c.jsonResult(ctx, socket, http.MethodPost, "/agent/v1/certificates", body, &result, "certificate issue"); err != nil {
		return certificates.Result{}, err
	}
	return certificates.Result{Names: result.Names, ExpiresAt: result.ExpiresAt}, nil
}

func (c *Client) jsonResult(ctx context.Context, socket, method, path string, request, result any, operation string) error {
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encode %s request: %w", operation, err)
	}
	response, cleanup, err := c.do(ctx, socket, method, path, body)
	if err != nil {
		return fmt.Errorf("request %s: %w", operation, err)
	}
	defer cleanup()
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
		return fmt.Errorf("%s returned status %d", operation, response.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(result); err != nil {
		return fmt.Errorf("decode %s response: %w", operation, err)
	}
	return nil
}

func (c *Client) CreateBackup(ctx context.Context, socket string, request agentbackup.CreateRequest) (agentbackup.Artifact, error) {
	body := agentapi.CreateBackupRequest{RunId: request.RunID, AccountId: request.AccountID, SystemUser: request.SystemUser, IncludeFiles: request.IncludeFiles, Databases: request.Databases, Metadata: request.Metadata}
	var result agentapi.BackupArtifactResult
	if err := c.jsonResult(ctx, socket, http.MethodPost, "/agent/v1/backups", body, &result, "backup create"); err != nil {
		return agentbackup.Artifact{}, err
	}
	return agentbackup.Artifact{Path: result.Path, Checksum: result.Checksum, Size: result.Size, Manifest: manifestValue(result.Manifest)}, nil
}

func (c *Client) PreviewBackup(ctx context.Context, socket string, request agentbackup.ArtifactRequest) (backupfmt.Manifest, error) {
	body := agentapi.BackupArtifactRequest{AccountId: request.AccountID, Path: request.Path, Checksum: request.Checksum}
	var result agentapi.BackupManifest
	if err := c.jsonResult(ctx, socket, http.MethodPost, "/agent/v1/backups/preview", body, &result, "backup preview"); err != nil {
		return backupfmt.Manifest{}, err
	}
	return manifestValue(result), nil
}

func (c *Client) RestoreBackup(ctx context.Context, socket string, request agentbackup.RestoreRequest) (string, error) {
	body := agentapi.RestoreBackupRequest{AccountId: request.AccountID, Path: request.Path, Checksum: request.Checksum, SystemUser: request.SystemUser, Files: request.Files, Databases: request.Databases, Metadata: request.Metadata}
	var result agentapi.RestoreBackupResult
	if err := c.jsonResult(ctx, socket, http.MethodPost, "/agent/v1/backups/restore", body, &result, "backup restore"); err != nil {
		return "", err
	}
	return result.Metadata, nil
}

func (c *Client) DeleteBackup(ctx context.Context, socket string, request agentbackup.ArtifactRequest) error {
	return c.sendJSON(ctx, socket, http.MethodDelete, "/agent/v1/backups", agentapi.BackupArtifactRequest{AccountId: request.AccountID, Path: request.Path, Checksum: request.Checksum}, "backup delete")
}

func manifestValue(value agentapi.BackupManifest) backupfmt.Manifest {
	entries := make([]backupfmt.Entry, 0, len(value.Entries))
	for _, entry := range value.Entries {
		entries = append(entries, backupfmt.Entry{Path: entry.Path, Size: entry.Size, Checksum: entry.Checksum})
	}
	return backupfmt.Manifest{Version: value.Version, RunID: value.RunId, AccountID: value.AccountId, CreatedAt: value.CreatedAt, Files: value.Files, Databases: value.Databases, Metadata: value.Metadata, Entries: entries}
}

func (c *Client) do(
	ctx context.Context,
	socket, method, path string,
	body []byte,
) (*http.Response, func(), error) {
	dialer := &net.Dialer{Timeout: c.timeout}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, "unix", socket)
		},
	}
	request, err := http.NewRequestWithContext(ctx, method, "http://agent"+path, bytes.NewReader(body))
	if err != nil {
		transport.CloseIdleConnections()
		return nil, func() {}, err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := (&http.Client{Transport: transport, Timeout: c.timeout}).Do(request)
	if err != nil {
		transport.CloseIdleConnections()
		return nil, func() {}, err
	}
	return response, transport.CloseIdleConnections, nil
}

func expectNoContent(response *http.Response, cleanup func(), operation string) error {
	defer cleanup()
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("%s returned status %d", operation, response.StatusCode)
	}
	return nil
}
