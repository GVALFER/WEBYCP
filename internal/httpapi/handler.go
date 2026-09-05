package httpapi

import (
	"crypto/subtle"
	"io"
	"log/slog"
	"net/http"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/audit"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/backups"
	"github.com/GVALFER/WEBYCP/internal/certificates"
	"github.com/GVALFER/WEBYCP/internal/databases"
	dnscontrol "github.com/GVALFER/WEBYCP/internal/dns"
	"github.com/GVALFER/WEBYCP/internal/httpapi/spec"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/nodes"
	"github.com/GVALFER/WEBYCP/internal/packages"
	"github.com/GVALFER/WEBYCP/internal/services"
	"github.com/GVALFER/WEBYCP/internal/tasks"
	"github.com/GVALFER/WEBYCP/internal/websites"
)

const sessionCookie = "webycp_session"

type Options struct {
	Version      string
	SecureCookie bool
	Auth         *auth.Service
	Accounts     *accounts.Service
	Packages     *packages.Service
	Services     *services.Service
	Websites     *websites.Service
	Databases    *databases.Service
	Tasks        *tasks.Service
	Certificates *certificates.Service
	Backups      *backups.Service
	DNS          *dnscontrol.Service
	Nodes        *nodes.Service
	Jobs         *jobs.Service
	Audit        audit.Repository
	Logger       *slog.Logger
}

type handler struct {
	options Options
	logger  *slog.Logger
}

type authedHandler func(http.ResponseWriter, *http.Request, auth.Session)

func New(options Options) http.Handler {
	logger := options.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	h := &handler{options: options, logger: logger}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", h.health)
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("POST /api/v1/auth/logout", h.withAuth(true, h.logout))
	mux.HandleFunc("GET /api/v1/auth/me", h.withAuth(false, h.me))
	mux.HandleFunc("PATCH /api/v1/auth/profile", h.withAuth(true, h.updateProfile))
	mux.HandleFunc("GET /api/v1/accounts", h.withAuth(false, h.listAccounts))
	mux.HandleFunc("POST /api/v1/accounts", h.withAuth(true, h.createAccount))
	mux.HandleFunc("PATCH /api/v1/accounts/{accountId}", h.withAuth(true, h.setAccount))
	mux.HandleFunc("DELETE /api/v1/accounts/{accountId}", h.withAuth(true, h.deleteAccount))
	mux.HandleFunc("PUT /api/v1/accounts/{accountId}/package", h.withAuth(true, h.assignAccountPackage))
	mux.HandleFunc("GET /api/v1/packages", h.withAuth(false, h.listPackages))
	mux.HandleFunc("POST /api/v1/packages", h.withAuth(true, h.createPackage))
	mux.HandleFunc("PATCH /api/v1/packages/{packageId}", h.withAuth(true, h.updatePackage))
	mux.HandleFunc("DELETE /api/v1/packages/{packageId}", h.withAuth(true, h.deletePackage))
	mux.HandleFunc("GET /api/v1/websites", h.withAuth(false, h.listWebsites))
	mux.HandleFunc("POST /api/v1/websites", h.withAuth(true, h.createWebsite))
	mux.HandleFunc("PATCH /api/v1/websites/{websiteId}", h.withAuth(true, h.setWebsite))
	mux.HandleFunc("DELETE /api/v1/websites/{websiteId}", h.withAuth(true, h.deleteWebsite))
	mux.HandleFunc("GET /api/v1/website-domains", h.withAuth(false, h.listWebsiteDomains))
	mux.HandleFunc("GET /api/v1/websites/{websiteId}/domains", h.withAuth(false, h.listWebsiteDomainsForWebsite))
	mux.HandleFunc("POST /api/v1/websites/{websiteId}/domains", h.withAuth(true, h.createWebsiteDomain))
	mux.HandleFunc("PATCH /api/v1/website-domains/{websiteDomainId}", h.withAuth(true, h.setWebsiteDomain))
	mux.HandleFunc("DELETE /api/v1/website-domains/{websiteDomainId}", h.withAuth(true, h.deleteWebsiteDomain))
	mux.HandleFunc("GET /api/v1/databases", h.withAuth(false, h.listDatabases))
	mux.HandleFunc("POST /api/v1/databases", h.withAuth(true, h.createDatabase))
	mux.HandleFunc("DELETE /api/v1/databases/{databaseId}", h.withAuth(true, h.deleteDatabase))
	mux.HandleFunc("GET /api/v1/database-users", h.withAuth(false, h.listDatabaseUsers))
	mux.HandleFunc("POST /api/v1/database-users", h.withAuth(true, h.createDatabaseUser))
	mux.HandleFunc("DELETE /api/v1/database-users/{databaseUserId}", h.withAuth(true, h.deleteDatabaseUser))
	mux.HandleFunc("GET /api/v1/database-grants", h.withAuth(false, h.listDatabaseGrants))
	mux.HandleFunc("PUT /api/v1/databases/{databaseId}/users/{databaseUserId}", h.withAuth(true, h.createDatabaseGrant))
	mux.HandleFunc("DELETE /api/v1/databases/{databaseId}/users/{databaseUserId}", h.withAuth(true, h.deleteDatabaseGrant))
	mux.HandleFunc("GET /api/v1/scheduled-tasks", h.withAuth(false, h.listScheduledTasks))
	mux.HandleFunc("POST /api/v1/scheduled-tasks", h.withAuth(true, h.createScheduledTask))
	mux.HandleFunc("PATCH /api/v1/scheduled-tasks/{scheduledTaskId}", h.withAuth(true, h.setScheduledTask))
	mux.HandleFunc("DELETE /api/v1/scheduled-tasks/{scheduledTaskId}", h.withAuth(true, h.deleteScheduledTask))
	mux.HandleFunc("GET /api/v1/certificates", h.withAuth(false, h.listCertificates))
	mux.HandleFunc("POST /api/v1/websites/{websiteId}/certificate", h.withAuth(true, h.issueWebsiteCertificate))
	mux.HandleFunc("POST /api/v1/certificates/panel", h.withAuth(true, h.issuePanelCertificate))
	mux.HandleFunc("POST /api/v1/certificates/{certificateId}/renew", h.withAuth(true, h.renewCertificate))
	mux.HandleFunc("PATCH /api/v1/certificates/{certificateId}", h.withAuth(true, h.setCertificate))
	mux.HandleFunc("GET /api/v1/backup-plans", h.withAuth(false, h.listBackupPlans))
	mux.HandleFunc("POST /api/v1/backup-plans", h.withAuth(true, h.createBackupPlan))
	mux.HandleFunc("PATCH /api/v1/backup-plans/{backupPlanId}", h.withAuth(true, h.setBackupPlan))
	mux.HandleFunc("DELETE /api/v1/backup-plans/{backupPlanId}", h.withAuth(true, h.deleteBackupPlan))
	mux.HandleFunc("POST /api/v1/backup-plans/{backupPlanId}/runs", h.withAuth(true, h.runBackupPlan))
	mux.HandleFunc("GET /api/v1/backup-runs", h.withAuth(false, h.listBackupRuns))
	mux.HandleFunc("GET /api/v1/backup-artifacts", h.withAuth(false, h.listBackupArtifacts))
	mux.HandleFunc("DELETE /api/v1/backup-artifacts/{backupArtifactId}", h.withAuth(true, h.deleteBackupArtifact))
	mux.HandleFunc("GET /api/v1/backup-artifacts/{backupArtifactId}/restore", h.withAuth(false, h.previewBackupRestore))
	mux.HandleFunc("POST /api/v1/backup-artifacts/{backupArtifactId}/restore", h.withAuth(true, h.restoreBackup))
	mux.HandleFunc("GET /api/v1/dns/providers", h.withAuth(false, h.listDNSProviders))
	mux.HandleFunc("GET /api/v1/dns/settings", h.withAuth(false, h.getDNSSettings))
	mux.HandleFunc("PUT /api/v1/dns/settings", h.withAuth(true, h.updateDNSSettings))
	mux.HandleFunc("GET /api/v1/dns/zones", h.withAuth(false, h.listDNSZones))
	mux.HandleFunc("POST /api/v1/dns/zones", h.withAuth(true, h.createDNSZone))
	mux.HandleFunc("GET /api/v1/dns/zones/{dnsZoneId}", h.withAuth(false, h.getDNSZone))
	mux.HandleFunc("DELETE /api/v1/dns/zones/{dnsZoneId}", h.withAuth(true, h.deleteDNSZone))
	mux.HandleFunc("GET /api/v1/dns/zones/{dnsZoneId}/records", h.withAuth(false, h.listDNSRecords))
	mux.HandleFunc("POST /api/v1/dns/zones/{dnsZoneId}/records", h.withAuth(true, h.createDNSRecord))
	mux.HandleFunc("PATCH /api/v1/dns/records/{dnsRecordId}", h.withAuth(true, h.updateDNSRecord))
	mux.HandleFunc("DELETE /api/v1/dns/records/{dnsRecordId}", h.withAuth(true, h.deleteDNSRecord))
	mux.HandleFunc("GET /api/v1/nodes", h.withAuth(false, h.listNodes))
	mux.HandleFunc("GET /api/v1/service-settings", h.withAuth(false, h.getServiceSettings))
	mux.HandleFunc("PUT /api/v1/service-settings", h.withAuth(true, h.updateServiceSettings))
	mux.HandleFunc("POST /api/v1/nodes/{nodeId}/probe", h.withAuth(true, h.probeNode))
	mux.HandleFunc("GET /api/v1/jobs", h.withAuth(false, h.listJobs))
	mux.HandleFunc("GET /api/v1/jobs/{jobId}", h.withAuth(false, h.getJob))
	mux.HandleFunc("GET /api/v1/audit-events", h.withAuth(false, h.listAuditEvents))
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "API route not found")
	})

	return mux
}

func (h *handler) health(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, publicapi.HealthResponse{
		Service: "webycp-server",
		Status:  publicapi.Ok,
		Version: h.options.Version,
	})
}

func (h *handler) withAuth(csrf bool, next authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookie)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required")
			return
		}
		session, err := h.options.Auth.Authenticate(r.Context(), cookie.Value)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required")
			return
		}
		if csrf && subtle.ConstantTimeCompare(
			[]byte(r.Header.Get("X-CSRF-Token")), []byte(session.CSRFToken),
		) != 1 {
			writeError(w, http.StatusForbidden, "invalid_csrf", "The CSRF token is invalid")
			return
		}
		if session.User.MustChangePassword &&
			r.URL.Path != "/api/v1/auth/me" &&
			r.URL.Path != "/api/v1/auth/logout" &&
			r.URL.Path != "/api/v1/auth/profile" {
			writeError(w, http.StatusForbidden, "password_change_required", "Change the temporary password to continue")
			return
		}

		next(w, r, session)
	}
}

func (h *handler) internalError(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.Error("request failed", "error", err, "method", r.Method, "path", r.URL.Path)
	writeError(w, http.StatusInternalServerError, "internal_error", "An internal error occurred")
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	httpx.WriteJSON(w, status, publicapi.ErrorResponse{Code: code, Message: message})
}
