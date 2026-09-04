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
	"github.com/GVALFER/WEBYCP/internal/domains"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/nodes"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite"
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

	nodeList, err := store.Nodes(context.Background())
	if err != nil || len(nodeList) != 1 {
		t.Fatalf("nodes = %v, error = %v", nodeList, err)
	}
	createAccount := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", strings.NewReader(`{
		"name":"Example Hosting","nodeId":"`+nodeList[0].ID+`"
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

	createDomain := httptest.NewRequest(http.MethodPost, "/api/v1/domains", strings.NewReader(`{
		"accountId":"`+accountResult.Account.ID+`","name":"Example.COM."
	}`))
	createDomain.AddCookie(cookie)
	createDomain.Header.Set("X-CSRF-Token", session.CSRFToken)
	createdDomain := httptest.NewRecorder()
	api.ServeHTTP(createdDomain, createDomain)
	if createdDomain.Code != http.StatusAccepted {
		t.Fatalf("domain status = %d, body = %s", createdDomain.Code, createdDomain.Body.String())
	}
	if body := createdDomain.Body.String(); !strings.Contains(body, `"name":"example.com"`) {
		t.Fatalf("domain body = %s", body)
	}
	var domainResult struct {
		Domain struct {
			ID string `json:"id"`
		} `json:"domain"`
	}
	if err := json.Unmarshal(createdDomain.Body.Bytes(), &domainResult); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateDomainStatus(context.Background(), domainResult.Domain.ID, "active"); err != nil {
		t.Fatal(err)
	}

	aliasPath := "/api/v1/domains/" + domainResult.Domain.ID + "/aliases"
	createAlias := httptest.NewRequest(http.MethodPost, aliasPath, strings.NewReader(`{
		"name":"WWW.Example.COM."
	}`))
	createAlias.AddCookie(cookie)
	createAlias.Header.Set("X-CSRF-Token", session.CSRFToken)
	createdAlias := httptest.NewRecorder()
	api.ServeHTTP(createdAlias, createAlias)
	if createdAlias.Code != http.StatusAccepted ||
		!strings.Contains(createdAlias.Body.String(), `"name":"www.example.com"`) ||
		!strings.Contains(createdAlias.Body.String(), `"enabled":true`) {
		t.Fatalf("alias status = %d, body = %s", createdAlias.Code, createdAlias.Body.String())
	}
	var aliasResult struct {
		Alias struct {
			ID string `json:"id"`
		} `json:"alias"`
	}
	if err := json.Unmarshal(createdAlias.Body.Bytes(), &aliasResult); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateAliasStatus(context.Background(), aliasResult.Alias.ID, "active"); err != nil {
		t.Fatal(err)
	}

	listAliases := httptest.NewRequest(http.MethodGet, aliasPath, nil)
	listAliases.AddCookie(cookie)
	listedAliases := httptest.NewRecorder()
	api.ServeHTTP(listedAliases, listAliases)
	if listedAliases.Code != http.StatusOK ||
		!strings.Contains(listedAliases.Body.String(), `"name":"www.example.com"`) {
		t.Fatalf("alias list status = %d, body = %s", listedAliases.Code, listedAliases.Body.String())
	}

	listDomains := httptest.NewRequest(http.MethodGet, "/api/v1/domains", nil)
	listDomains.AddCookie(cookie)
	listedDomains := httptest.NewRecorder()
	api.ServeHTTP(listedDomains, listDomains)
	if listedDomains.Code != http.StatusOK ||
		!strings.Contains(listedDomains.Body.String(), `"name":"example.com"`) ||
		!strings.Contains(listedDomains.Body.String(), `"enabled":true`) {
		t.Fatalf("domain list status = %d, body = %s", listedDomains.Code, listedDomains.Body.String())
	}

	aliasItemPath := aliasPath + "/" + aliasResult.Alias.ID
	renameAlias := httptest.NewRequest(
		http.MethodPatch, aliasItemPath, strings.NewReader(`{"name":"static.example.com"}`),
	)
	renameAlias.AddCookie(cookie)
	renameAlias.Header.Set("X-CSRF-Token", session.CSRFToken)
	renamedAlias := httptest.NewRecorder()
	api.ServeHTTP(renamedAlias, renameAlias)
	if renamedAlias.Code != http.StatusAccepted ||
		!strings.Contains(renamedAlias.Body.String(), `"name":"static.example.com"`) {
		t.Fatalf("alias rename status = %d, body = %s", renamedAlias.Code, renamedAlias.Body.String())
	}
	if err := store.CompleteAliasRename(context.Background(), aliasResult.Alias.ID); err != nil {
		t.Fatal(err)
	}

	domainPath := "/api/v1/domains/" + domainResult.Domain.ID
	renameDomain := httptest.NewRequest(
		http.MethodPatch, domainPath, strings.NewReader(`{"name":"renamed.example.com"}`),
	)
	renameDomain.AddCookie(cookie)
	renameDomain.Header.Set("X-CSRF-Token", session.CSRFToken)
	renamedDomain := httptest.NewRecorder()
	api.ServeHTTP(renamedDomain, renameDomain)
	if renamedDomain.Code != http.StatusAccepted ||
		!strings.Contains(renamedDomain.Body.String(), `"name":"renamed.example.com"`) {
		t.Fatalf("domain rename status = %d, body = %s", renamedDomain.Code, renamedDomain.Body.String())
	}
	reservedName := httptest.NewRequest(http.MethodPost, "/api/v1/domains", strings.NewReader(`{
		"accountId":"`+accountResult.Account.ID+`","name":"example.com"
	}`))
	reservedName.AddCookie(cookie)
	reservedName.Header.Set("X-CSRF-Token", session.CSRFToken)
	reserved := httptest.NewRecorder()
	api.ServeHTTP(reserved, reservedName)
	if reserved.Code != http.StatusConflict {
		t.Fatalf("reserved hostname status = %d, body = %s", reserved.Code, reserved.Body.String())
	}
	if err := store.CompleteDomainRename(context.Background(), domainResult.Domain.ID); err != nil {
		t.Fatal(err)
	}

	disableAlias := httptest.NewRequest(
		http.MethodPatch, aliasItemPath, strings.NewReader(`{"enabled":false}`),
	)
	disableAlias.AddCookie(cookie)
	disableAlias.Header.Set("X-CSRF-Token", session.CSRFToken)
	disabledAlias := httptest.NewRecorder()
	api.ServeHTTP(disabledAlias, disableAlias)
	if disabledAlias.Code != http.StatusAccepted ||
		!strings.Contains(disabledAlias.Body.String(), `"enabled":false`) {
		t.Fatalf("alias disable status = %d, body = %s", disabledAlias.Code, disabledAlias.Body.String())
	}
	if err := store.UpdateAliasStatus(context.Background(), aliasResult.Alias.ID, "active"); err != nil {
		t.Fatal(err)
	}
	deleteAlias := httptest.NewRequest(http.MethodDelete, aliasItemPath, nil)
	deleteAlias.AddCookie(cookie)
	deleteAlias.Header.Set("X-CSRF-Token", session.CSRFToken)
	deletedAlias := httptest.NewRecorder()
	api.ServeHTTP(deletedAlias, deleteAlias)
	if deletedAlias.Code != http.StatusAccepted {
		t.Fatalf("alias delete status = %d, body = %s", deletedAlias.Code, deletedAlias.Body.String())
	}

	disableDomain := httptest.NewRequest(
		http.MethodPatch, domainPath, strings.NewReader(`{"enabled":false}`),
	)
	disableDomain.AddCookie(cookie)
	disableDomain.Header.Set("X-CSRF-Token", session.CSRFToken)
	disabledDomain := httptest.NewRecorder()
	api.ServeHTTP(disabledDomain, disableDomain)
	if disabledDomain.Code != http.StatusAccepted ||
		!strings.Contains(disabledDomain.Body.String(), `"enabled":false`) {
		t.Fatalf("domain disable status = %d, body = %s", disabledDomain.Code, disabledDomain.Body.String())
	}
	if err := store.UpdateDomainStatus(context.Background(), domainResult.Domain.ID, "active"); err != nil {
		t.Fatal(err)
	}
	deleteDomain := httptest.NewRequest(http.MethodDelete, domainPath, nil)
	deleteDomain.AddCookie(cookie)
	deleteDomain.Header.Set("X-CSRF-Token", session.CSRFToken)
	deletedDomain := httptest.NewRecorder()
	api.ServeHTTP(deletedDomain, deleteDomain)
	if deletedDomain.Code != http.StatusAccepted {
		t.Fatalf("domain delete status = %d, body = %s", deletedDomain.Code, deletedDomain.Body.String())
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
	if _, err := store.EnsureLocal(ctx, "test-node", t.TempDir()+"/agent.sock"); err != nil {
		store.Close()
		t.Fatal(err)
	}
	worker := jobs.NewWorker(store, store, slog.Default())
	agent := agentclient.New(time.Second)
	accountService := accounts.NewService(store, store, agent, worker.Notify)
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
		Domains:  domains.NewService(store, accountService, store, agent, worker.Notify),
		Nodes:    nodes.NewService(store, agent),
		Jobs:     jobs.NewService(store, worker.Notify),
		Audit:    store,
	}), store, credentials
}
