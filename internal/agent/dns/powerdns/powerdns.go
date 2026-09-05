package powerdns

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	agentdns "github.com/GVALFER/WEBYCP/internal/agent/dns"
)

const markerPrefix = "webycp:"

type Client struct {
	baseURL string
	key     string
	http    *http.Client
}

type zone struct {
	Name        string   `json:"name"`
	Kind        string   `json:"kind,omitempty"`
	Account     string   `json:"account,omitempty"`
	Nameservers []string `json:"nameservers,omitempty"`
	SOAEditAPI  string   `json:"soa_edit_api,omitempty"`
	RRSets      []rrset  `json:"rrsets,omitempty"`
}

type record struct {
	Content  string `json:"content"`
	Disabled bool   `json:"disabled"`
}

type rrset struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	ChangeType string   `json:"changetype,omitempty"`
	TTL        int64    `json:"ttl,omitempty"`
	Records    []record `json:"records,omitempty"`
}

func New(baseURL, key string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		key:     strings.TrimSpace(key),
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) Health(ctx context.Context) error {
	response, err := c.request(ctx, http.MethodGet, "/api/v1/servers/localhost/zones", nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("PowerDNS health returned status %d", response.StatusCode)
	}
	return nil
}

func (c *Client) EnsureZone(ctx context.Context, value agentdns.Zone) error {
	if err := agentdns.ValidateZone(value); err != nil {
		return err
	}
	request := zone{
		Name: fqdn(value.Name), Kind: "Native", Account: markerPrefix + value.ID,
		SOAEditAPI: "DEFAULT", RRSets: initialRRSets(value),
	}
	response, err := c.request(ctx, http.MethodPost, "/api/v1/servers/localhost/zones", request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusCreated || response.StatusCode == http.StatusOK {
		return nil
	}
	if response.StatusCode != http.StatusConflict && response.StatusCode != http.StatusUnprocessableEntity {
		return fmt.Errorf("PowerDNS zone create returned status %d", response.StatusCode)
	}
	existing, found, err := c.getZone(ctx, value.Name)
	if err != nil {
		return err
	}
	if found && existing.Account == markerPrefix+value.ID {
		return nil
	}
	return errors.New("PowerDNS zone name is already owned by another resource")
}

func initialRRSets(value agentdns.Zone) []rrset {
	name := fqdn(value.Name)
	serial := time.Now().UTC().Format("20060102") + "01"
	soa := fmt.Sprintf(
		"%s hostmaster.%s %s 10800 3600 604800 3600",
		fqdn(value.Nameservers[0]), name, serial,
	)
	nameservers := fqdnValues(value.Nameservers)
	nsRecords := make([]record, 0, len(nameservers))
	for _, nameserver := range nameservers {
		nsRecords = append(nsRecords, record{Content: nameserver})
	}
	return []rrset{
		{Name: name, Type: "SOA", TTL: 3600, Records: []record{{Content: soa}}},
		{Name: name, Type: "NS", TTL: 3600, Records: nsRecords},
	}
}

func (c *Client) DeleteZone(ctx context.Context, value agentdns.Zone) error {
	if err := agentdns.ValidateZoneIdentity(value); err != nil {
		return err
	}
	existing, found, err := c.getZone(ctx, value.Name)
	if err != nil || !found {
		return err
	}
	if existing.Account != markerPrefix+value.ID {
		return nil
	}
	response, err := c.request(ctx, http.MethodDelete, c.zonePath(value.Name), nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent && response.StatusCode != http.StatusNotFound {
		return fmt.Errorf("PowerDNS zone delete returned status %d", response.StatusCode)
	}
	return nil
}

func (c *Client) SyncRecordSets(
	ctx context.Context, value agentdns.Zone, values []agentdns.RecordSet,
) error {
	if err := agentdns.ValidateRecordSets(value, values); err != nil {
		return err
	}
	existing, found, err := c.getZone(ctx, value.Name)
	if err != nil {
		return err
	}
	if !found || existing.Account != markerPrefix+value.ID {
		return errors.New("PowerDNS zone is not owned by this resource")
	}
	sets := make([]rrset, 0, len(values))
	for _, value := range values {
		set := rrset{Name: fqdn(value.Name), Type: value.Type, ChangeType: "DELETE"}
		if len(value.Records) > 0 {
			set.ChangeType = "REPLACE"
			set.TTL = value.TTL
			set.Records = make([]record, 0, len(value.Records))
			for _, content := range value.Records {
				set.Records = append(set.Records, record{Content: content})
			}
		}
		sets = append(sets, set)
	}
	response, err := c.request(ctx, http.MethodPatch, c.zonePath(value.Name), struct {
		RRSets []rrset `json:"rrsets"`
	}{RRSets: sets})
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("PowerDNS record sync returned status %d", response.StatusCode)
	}
	return nil
}

func (c *Client) getZone(ctx context.Context, name string) (zone, bool, error) {
	response, err := c.request(ctx, http.MethodGet, c.zonePath(name), nil)
	if err != nil {
		return zone{}, false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return zone{}, false, nil
	}
	if response.StatusCode != http.StatusOK {
		return zone{}, false, fmt.Errorf("PowerDNS zone lookup returned status %d", response.StatusCode)
	}
	var value zone
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&value); err != nil {
		return zone{}, false, fmt.Errorf("decode PowerDNS zone: %w", err)
	}
	return value, true, nil
}

func (c *Client) request(
	ctx context.Context, method, path string, value any,
) (*http.Response, error) {
	if c.baseURL == "" || c.key == "" {
		return nil, errors.New("PowerDNS API is not configured")
	}
	var body io.Reader
	if value != nil {
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("encode PowerDNS request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("create PowerDNS request: %w", err)
	}
	request.Header.Set("X-API-Key", c.key)
	if value != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request PowerDNS API: %w", err)
	}
	return response, nil
}

func (c *Client) zonePath(name string) string {
	return "/api/v1/servers/localhost/zones/" + url.PathEscape(fqdn(name))
}

func fqdn(value string) string {
	return strings.TrimSuffix(value, ".") + "."
}

func fqdnValues(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, fqdn(value))
	}
	return result
}
