package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	agentclient "github.com/GVALFER/WEBYCP/internal/agent/client"
	"github.com/GVALFER/WEBYCP/internal/auth"
	dnscontrol "github.com/GVALFER/WEBYCP/internal/dns"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/nodes"
	"github.com/GVALFER/WEBYCP/internal/packages"
	"github.com/GVALFER/WEBYCP/internal/services"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite"
	"github.com/GVALFER/WEBYCP/internal/tasks"
	"github.com/GVALFER/WEBYCP/internal/websites"
)

func TestHealth(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	response := httptest.NewRecorder()

	New(Options{Version: "test"}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if body := response.Body.String(); !strings.Contains(body, `"version":"test"`) {
		t.Fatalf("body = %q, want version", body)
	}
}

func TestUnknownAPIPath(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/missing", nil)
	response := httptest.NewRecorder()

	New(Options{}).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestTemporaryAdminSetupAndProbe(t *testing.T) {
	api, store, credentials := testAPI(t)
	defer store.Close()

	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{
		"username":"`+credentials.Username+`","password":"`+credentials.Password+`"
	}`))
	login.Header.Set("Content-Type", "application/json")
	created := httptest.NewRecorder()
	api.ServeHTTP(created, login)
	if created.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", created.Code, created.Body.String())
	}
	if len(created.Result().Cookies()) != 1 {
		t.Fatal("expected session cookie")
	}
	cookie := created.Result().Cookies()[0]
	var session struct {
		CSRFToken string `json:"csrfToken"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}

	blocked := httptest.NewRequest(http.MethodGet, "/api/v1/nodes", nil)
	blocked.AddCookie(cookie)
	blockedResponse := httptest.NewRecorder()
	api.ServeHTTP(blockedResponse, blocked)
	if blockedResponse.Code != http.StatusForbidden {
		t.Fatalf("temporary session status = %d, body = %s", blockedResponse.Code, blockedResponse.Body.String())
	}

	profile := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/profile", strings.NewReader(`{
		"username":"owner","name":"Test Admin","email":"admin@example.com",
		"timezone":"Europe/Lisbon",
		"password":"correct horse battery staple"
	}`))
	profile.Header.Set("Content-Type", "application/json")
	profile.Header.Set("X-CSRF-Token", session.CSRFToken)
	profile.AddCookie(cookie)
	updated := httptest.NewRecorder()
	api.ServeHTTP(updated, profile)
	if updated.Code != http.StatusOK ||
		!strings.Contains(updated.Body.String(), `"username":"owner"`) ||
		!strings.Contains(updated.Body.String(), `"timezone":"Europe/Lisbon"`) ||
		!strings.Contains(updated.Body.String(), `"mustChangePassword":false`) {
		t.Fatalf("profile status = %d, body = %s", updated.Code, updated.Body.String())
	}

	me := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	me.AddCookie(cookie)
	current := httptest.NewRecorder()
	api.ServeHTTP(current, me)
	if current.Code != http.StatusOK {
		t.Fatalf("me status = %d, body = %s", current.Code, current.Body.String())
	}

	settings := httptest.NewRequest(http.MethodGet, "/api/v1/service-settings", nil)
	settings.AddCookie(cookie)
	settingsResponse := httptest.NewRecorder()
	api.ServeHTTP(settingsResponse, settings)
	if settingsResponse.Code != http.StatusOK ||
		!strings.Contains(settingsResponse.Body.String(), `"webDriver":"nginx"`) {
		t.Fatalf("service settings status = %d, body = %s", settingsResponse.Code, settingsResponse.Body.String())
	}

	unsupported := httptest.NewRequest(http.MethodPut, "/api/v1/service-settings", strings.NewReader(`{
		"webDriver":"apache","runtimeDriver":"phpfpm","runtimeVersion":"8.3",
		"databaseDriver":"mysql","schedulerDriver":"crontab","backupDriver":"local"
	}`))
	unsupported.AddCookie(cookie)
	unsupported.Header.Set("X-CSRF-Token", session.CSRFToken)
	unsupportedResponse := httptest.NewRecorder()
	api.ServeHTTP(unsupportedResponse, unsupported)
	if unsupportedResponse.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unsupported defaults status = %d, body = %s", unsupportedResponse.Code, unsupportedResponse.Body.String())
	}

	nodeList, err := store.Nodes(context.Background())
	if err != nil || len(nodeList) != 1 {
		t.Fatalf("nodes = %v, error = %v", nodeList, err)
	}
	createAccount := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", strings.NewReader(`{
		"name":"Example Hosting","nodeId":"`+nodeList[0].ID+`","packageId":"`+packages.DefaultID+`"
	}`))
	createAccount.AddCookie(cookie)
	createAccount.Header.Set("X-CSRF-Token", session.CSRFToken)
	createdAccount := httptest.NewRecorder()
	api.ServeHTTP(createdAccount, createAccount)
	if createdAccount.Code != http.StatusAccepted {
		t.Fatalf("account status = %d, body = %s", createdAccount.Code, createdAccount.Body.String())
	}
	var accountResult struct {
		Account struct {
			ID string `json:"id"`
		} `json:"account"`
	}
	if err := json.Unmarshal(createdAccount.Body.Bytes(), &accountResult); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateStatus(context.Background(), accountResult.Account.ID, "active"); err != nil {
		t.Fatal(err)
	}
	checkTaskAPI(t, api, cookie, session.CSRFToken, accountResult.Account.ID)

	settingsRequest := httptest.NewRequest(http.MethodPut, "/api/v1/dns/settings", strings.NewReader(`{
		"primaryNameserver":"NS1.Example.COM.","secondaryNameserver":"ns2.example.com","defaultTtl":3600
	}`))
	settingsRequest.AddCookie(cookie)
	settingsRequest.Header.Set("X-CSRF-Token", session.CSRFToken)
	dnsSettingsResponse := httptest.NewRecorder()
	api.ServeHTTP(dnsSettingsResponse, settingsRequest)
	if dnsSettingsResponse.Code != http.StatusOK ||
		!strings.Contains(dnsSettingsResponse.Body.String(), `"primaryNameserver":"ns1.example.com"`) {
		t.Fatalf("DNS settings status = %d, body = %s", dnsSettingsResponse.Code, dnsSettingsResponse.Body.String())
	}

	providersRequest := httptest.NewRequest(http.MethodGet, "/api/v1/dns/providers", nil)
	providersRequest.AddCookie(cookie)
	providersResponse := httptest.NewRecorder()
	api.ServeHTTP(providersResponse, providersRequest)
	var providers struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if providersResponse.Code != http.StatusOK || json.Unmarshal(providersResponse.Body.Bytes(), &providers) != nil || len(providers.Items) != 1 {
		t.Fatalf("DNS providers status = %d, body = %s", providersResponse.Code, providersResponse.Body.String())
	}

	zoneRequest := httptest.NewRequest(http.MethodPost, "/api/v1/dns/zones", strings.NewReader(`{
		"accountId":"`+accountResult.Account.ID+`","providerId":"`+providers.Items[0].ID+`","name":"Example.TEST."
	}`))
	zoneRequest.AddCookie(cookie)
	zoneRequest.Header.Set("X-CSRF-Token", session.CSRFToken)
	zoneResponse := httptest.NewRecorder()
	api.ServeHTTP(zoneResponse, zoneRequest)
	var zoneResult struct {
		Zone struct {
			ID string `json:"id"`
		} `json:"zone"`
	}
	if zoneResponse.Code != http.StatusAccepted || json.Unmarshal(zoneResponse.Body.Bytes(), &zoneResult) != nil ||
		!strings.Contains(zoneResponse.Body.String(), `"name":"example.test"`) {
		t.Fatalf("DNS zone status = %d, body = %s", zoneResponse.Code, zoneResponse.Body.String())
	}
	if err := store.UpdateDNSZoneStatus(context.Background(), zoneResult.Zone.ID, "active"); err != nil {
		t.Fatal(err)
	}

	recordPath := "/api/v1/dns/zones/" + zoneResult.Zone.ID + "/records"
	recordRequest := httptest.NewRequest(http.MethodPost, recordPath, strings.NewReader(`{
		"name":"@","type":"A","content":"192.0.2.10","ttl":3600,"priority":0
	}`))
	recordRequest.AddCookie(cookie)
	recordRequest.Header.Set("X-CSRF-Token", session.CSRFToken)
	recordResponse := httptest.NewRecorder()
	api.ServeHTTP(recordResponse, recordRequest)
	if recordResponse.Code != http.StatusAccepted ||
		!strings.Contains(recordResponse.Body.String(), `"name":"example.test"`) {
		t.Fatalf("DNS record status = %d, body = %s", recordResponse.Code, recordResponse.Body.String())
	}
	listAccounts := httptest.NewRequest(http.MethodGet, "/api/v1/accounts?page=9&size=1", nil)
	listAccounts.AddCookie(cookie)
	listedAccounts := httptest.NewRecorder()
	api.ServeHTTP(listedAccounts, listAccounts)
	if listedAccounts.Code != http.StatusOK ||
		!strings.Contains(listedAccounts.Body.String(), `"pagination":{"page":1,"size":1,"totalItems":1,"totalPages":1}`) {
		t.Fatalf("account list status = %d, body = %s", listedAccounts.Code, listedAccounts.Body.String())
	}
	for _, path := range []string{
		"/api/v1/accounts", "/api/v1/websites",
		"/api/v1/certificates", "/api/v1/databases", "/api/v1/database-users",
		"/api/v1/database-grants", "/api/v1/scheduled-tasks", "/api/v1/backup-plans",
		"/api/v1/backup-runs", "/api/v1/backup-artifacts", "/api/v1/jobs",
	} {
		invalidPage := httptest.NewRequest(http.MethodGet, path+"?size=101", nil)
		invalidPage.AddCookie(cookie)
		invalidPageResponse := httptest.NewRecorder()
		api.ServeHTTP(invalidPageResponse, invalidPage)
		if invalidPageResponse.Code != http.StatusBadRequest {
			t.Fatalf("invalid page %s status = %d, body = %s", path, invalidPageResponse.Code, invalidPageResponse.Body.String())
		}
	}

	createWebsite := httptest.NewRequest(http.MethodPost, "/api/v1/websites", strings.NewReader(`{
		"accountId":"`+accountResult.Account.ID+`","name":"Example site",
		"primaryDomain":"Example.COM.","kind":"php","webDriver":"nginx",
		"runtimeDriver":"phpfpm","runtimeVersion":"8.3"
	}`))
	createWebsite.AddCookie(cookie)
	createWebsite.Header.Set("X-CSRF-Token", session.CSRFToken)
	createdWebsite := httptest.NewRecorder()
	api.ServeHTTP(createdWebsite, createWebsite)
	if createdWebsite.Code != http.StatusAccepted || !strings.Contains(createdWebsite.Body.String(), `"name":"Example site"`) {
		t.Fatalf("website status = %d, body = %s", createdWebsite.Code, createdWebsite.Body.String())
	}
	var websiteResult struct {
		Website struct {
			ID string `json:"id"`
		} `json:"website"`
	}
	if err := json.Unmarshal(createdWebsite.Body.Bytes(), &websiteResult); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateWebsiteStatus(context.Background(), websiteResult.Website.ID, "active"); err != nil {
		t.Fatal(err)
	}
	primary, err := store.PrimaryDomain(context.Background(), websiteResult.Website.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateWebsiteDomainStatus(context.Background(), primary.ID, "active"); err != nil {
		t.Fatal(err)
	}

	domainPath := "/api/v1/websites/" + websiteResult.Website.ID + "/domains"
	createDomain := httptest.NewRequest(http.MethodPost, domainPath, strings.NewReader(`{"hostname":"WWW.Example.COM."}`))
	createDomain.AddCookie(cookie)
	createDomain.Header.Set("X-CSRF-Token", session.CSRFToken)
	createdDomain := httptest.NewRecorder()
	api.ServeHTTP(createdDomain, createDomain)
	if createdDomain.Code != http.StatusAccepted || !strings.Contains(createdDomain.Body.String(), `"hostname":"www.example.com"`) {
		t.Fatalf("domain status = %d, body = %s", createdDomain.Code, createdDomain.Body.String())
	}
	var domainResult struct {
		Domain struct {
			ID string `json:"id"`
		} `json:"domain"`
	}
	if err := json.Unmarshal(createdDomain.Body.Bytes(), &domainResult); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateWebsiteDomainStatus(context.Background(), domainResult.Domain.ID, "active"); err != nil {
		t.Fatal(err)
	}
	duplicateDomain := httptest.NewRequest(http.MethodPost, domainPath, strings.NewReader(`{"hostname":"www.example.com"}`))
	duplicateDomain.AddCookie(cookie)
	duplicateDomain.Header.Set("X-CSRF-Token", session.CSRFToken)
	duplicateResponse := httptest.NewRecorder()
	api.ServeHTTP(duplicateResponse, duplicateDomain)
	if duplicateResponse.Code != http.StatusConflict || !strings.Contains(duplicateResponse.Body.String(), `"code":"hostname_exists"`) {
		t.Fatalf("duplicate hostname status = %d, body = %s", duplicateResponse.Code, duplicateResponse.Body.String())
	}

	listDomains := httptest.NewRequest(http.MethodGet, "/api/v1/website-domains?kind=alias", nil)
	listDomains.AddCookie(cookie)
	listedDomains := httptest.NewRecorder()
	api.ServeHTTP(listedDomains, listDomains)
	if listedDomains.Code != http.StatusOK || !strings.Contains(listedDomains.Body.String(), `"hostname":"www.example.com"`) {
		t.Fatalf("domain list status = %d, body = %s", listedDomains.Code, listedDomains.Body.String())
	}

	renameDomain := httptest.NewRequest(http.MethodPatch, "/api/v1/website-domains/"+domainResult.Domain.ID, strings.NewReader(`{"hostname":"static.example.com"}`))
	renameDomain.AddCookie(cookie)
	renameDomain.Header.Set("X-CSRF-Token", session.CSRFToken)
	renamedDomain := httptest.NewRecorder()
	api.ServeHTTP(renamedDomain, renameDomain)
	if renamedDomain.Code != http.StatusAccepted || !strings.Contains(renamedDomain.Body.String(), `"hostname":"static.example.com"`) {
		t.Fatalf("domain rename status = %d, body = %s", renamedDomain.Code, renamedDomain.Body.String())
	}
	if err := store.CompleteWebsiteDomainRename(context.Background(), domainResult.Domain.ID); err != nil {
		t.Fatal(err)
	}

	path := "/api/v1/nodes/" + nodeList[0].ID + "/probe"
	withoutCSRF := httptest.NewRequest(http.MethodPost, path, nil)
	withoutCSRF.AddCookie(cookie)
	forbidden := httptest.NewRecorder()
	api.ServeHTTP(forbidden, withoutCSRF)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("probe without CSRF status = %d", forbidden.Code)
	}

	probe := httptest.NewRequest(http.MethodPost, path, nil)
	probe.AddCookie(cookie)
	probe.Header.Set("X-CSRF-Token", session.CSRFToken)
	accepted := httptest.NewRecorder()
	api.ServeHTTP(accepted, probe)
	if accepted.Code != http.StatusAccepted {
		t.Fatalf("probe status = %d, body = %s", accepted.Code, accepted.Body.String())
	}
}

func testAPI(t *testing.T) (http.Handler, *sqlite.Store, auth.Credentials) {
	t.Helper()
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "webycp.db"))
	if err != nil {
		t.Fatal(err)
	}
	node, err := store.EnsureLocal(ctx, "test-node", t.TempDir()+"/agent.sock")
	if err != nil {
		store.Close()
		t.Fatal(err)
	}
	worker := jobs.NewWorker(store, store, slog.Default())
	agent := agentclient.New(time.Second)
	packageService := packages.NewService(store)
	accountService := accounts.NewService(store, store, agent, packageService, worker.Notify)
	dnsService := dnscontrol.NewService(store, accountService, store, agent, worker.Notify)
	if _, err := dnsService.EnsureLocalProvider(ctx, node.ID); err != nil {
		store.Close()
		t.Fatal(err)
	}
	authService := auth.NewService(store)
	credentials, created, err := authService.InitAdmin(ctx)
	if err != nil || !created {
		store.Close()
		t.Fatalf("initialize administrator: created=%v, error=%v", created, err)
	}
	return New(Options{
		Version: "test", SecureCookie: false,
		Auth:     authService,
		Accounts: accountService,
		Packages: packageService,
		Services: services.NewService(store),
		Websites: websites.NewService(store, accountService, store, agent, worker.Notify),
		Tasks:    tasks.NewService(store, accountService, store, agent, worker.Notify),
		DNS:      dnsService,
		Nodes:    nodes.NewService(store, agent),
		Jobs:     jobs.NewService(store, worker.Notify),
		Audit:    store,
	}), store, credentials
}
