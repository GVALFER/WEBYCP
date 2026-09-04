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
	cronjob "github.com/GVALFER/WEBYCP/internal/cron"
	"github.com/GVALFER/WEBYCP/internal/databases"
	"github.com/GVALFER/WEBYCP/internal/domains"
	"github.com/GVALFER/WEBYCP/internal/httpapi/spec"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/jobs"
	"github.com/GVALFER/WEBYCP/internal/nodes"
)

const sessionCookie = "webycp_session"

type Options struct {
	Version      string
	SecureCookie bool
	Auth         *auth.Service
	Accounts     *accounts.Service
	Domains      *domains.Service
	Databases    *databases.Service
	Cron         *cronjob.Service
	Certificates *certificates.Service
	Backups      *backups.Service
	Nodes        *nodes.Service
	Jobs         *jobs.Service
	Audit        audit.Recorder
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
	mux.HandleFunc("GET /api/v1/domains", h.withAuth(false, h.listDomains))
	mux.HandleFunc("POST /api/v1/domains", h.withAuth(true, h.createDomain))
	mux.HandleFunc("PATCH /api/v1/domains/{domainId}", h.withAuth(true, h.setDomain))
	mux.HandleFunc("DELETE /api/v1/domains/{domainId}", h.withAuth(true, h.deleteDomain))
	mux.HandleFunc("GET /api/v1/domains/{domainId}/aliases", h.withAuth(false, h.listAliases))
	mux.HandleFunc("POST /api/v1/domains/{domainId}/aliases", h.withAuth(true, h.createAlias))
	mux.HandleFunc("PATCH /api/v1/domains/{domainId}/aliases/{aliasId}", h.withAuth(true, h.setAlias))
	mux.HandleFunc("DELETE /api/v1/domains/{domainId}/aliases/{aliasId}", h.withAuth(true, h.deleteAlias))
	mux.HandleFunc("GET /api/v1/databases", h.withAuth(false, h.listDatabases))
	mux.HandleFunc("POST /api/v1/databases", h.withAuth(true, h.createDatabase))
	mux.HandleFunc("DELETE /api/v1/databases/{databaseId}", h.withAuth(true, h.deleteDatabase))
	mux.HandleFunc("GET /api/v1/database-users", h.withAuth(false, h.listDatabaseUsers))
	mux.HandleFunc("POST /api/v1/database-users", h.withAuth(true, h.createDatabaseUser))
	mux.HandleFunc("DELETE /api/v1/database-users/{databaseUserId}", h.withAuth(true, h.deleteDatabaseUser))
	mux.HandleFunc("GET /api/v1/database-grants", h.withAuth(false, h.listDatabaseGrants))
	mux.HandleFunc("PUT /api/v1/databases/{databaseId}/users/{databaseUserId}", h.withAuth(true, h.createDatabaseGrant))
	mux.HandleFunc("DELETE /api/v1/databases/{databaseId}/users/{databaseUserId}", h.withAuth(true, h.deleteDatabaseGrant))
	mux.HandleFunc("GET /api/v1/cron-jobs", h.withAuth(false, h.listCronJobs))
	mux.HandleFunc("POST /api/v1/cron-jobs", h.withAuth(true, h.createCronJob))
	mux.HandleFunc("PATCH /api/v1/cron-jobs/{cronJobId}", h.withAuth(true, h.setCronJob))
	mux.HandleFunc("DELETE /api/v1/cron-jobs/{cronJobId}", h.withAuth(true, h.deleteCronJob))
	mux.HandleFunc("GET /api/v1/certificates", h.withAuth(false, h.listCertificates))
	mux.HandleFunc("POST /api/v1/domains/{domainId}/certificate", h.withAuth(true, h.issueDomainCertificate))
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
	mux.HandleFunc("GET /api/v1/nodes", h.withAuth(false, h.listNodes))
	mux.HandleFunc("POST /api/v1/nodes/{nodeId}/probe", h.withAuth(true, h.probeNode))
	mux.HandleFunc("GET /api/v1/jobs", h.withAuth(false, h.listJobs))
	mux.HandleFunc("GET /api/v1/jobs/{jobId}", h.withAuth(false, h.getJob))
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
