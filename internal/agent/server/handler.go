package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"

	agentbackup "github.com/GVALFER/WEBYCP/internal/agent/backup"
	agentcertificate "github.com/GVALFER/WEBYCP/internal/agent/certificate"
	agentdatabase "github.com/GVALFER/WEBYCP/internal/agent/database"
	"github.com/GVALFER/WEBYCP/internal/agent/protocol"
	"github.com/GVALFER/WEBYCP/internal/agent/scheduler"
	"github.com/GVALFER/WEBYCP/internal/backupfmt"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

type AccountManager interface {
	Ensure(context.Context, string, string) error
}

type AccountLifecycle interface {
	Disable(context.Context, string, string) error
	Enable(context.Context, string, string) error
	Delete(context.Context, string, string) error
}

type DomainManager interface {
	Ensure(context.Context, string, string, string, string, string, []string) error
	Disable(context.Context, string, string, string) error
	Delete(context.Context, string, string, string, string) error
	Rename(context.Context, string, string, string, string, string, string, []string) error
}

type Options struct {
	Version        string
	Accounts       AccountManager
	AccountActions AccountLifecycle
	Domains        DomainManager
	Databases      agentdatabase.Driver
	Cron           scheduler.Driver
	Certificates   agentcertificate.Driver
	Backups        agentbackup.Driver
	Logger         *slog.Logger
}

func New(options Options) http.Handler {
	logger := options.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /agent/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, agentapi.HealthResponse{
			ProtocolVersion: "v1",
			Service:         "webycp-agent",
			Status:          agentapi.Ok,
			Version:         options.Version,
		})
	})
	mux.HandleFunc("POST /agent/v1/accounts", func(w http.ResponseWriter, r *http.Request) {
		var request agentapi.EnsureAccountRequest
		if err := httpx.DecodeJSON(w, r, &request); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, agentapi.ErrorResponse{
				Code: "invalid_json", Message: "The request body is invalid",
			})
			return
		}
		if err := validate.ID("accountId", request.AccountId); err != nil {
			httpx.WriteJSON(w, http.StatusUnprocessableEntity, agentapi.ErrorResponse{
				Code: "validation_error", Message: "Account ID is invalid",
			})
			return
		}
		if options.Accounts == nil {
			logger.Error("account manager is not configured")
			httpx.WriteJSON(w, http.StatusInternalServerError, agentapi.ErrorResponse{
				Code: "internal_error", Message: "An internal error occurred",
			})
			return
		}
		if err := options.Accounts.Ensure(r.Context(), request.AccountId, request.SystemUser); err != nil {
			var validationError *validate.Error
			if errors.As(err, &validationError) {
				httpx.WriteJSON(w, http.StatusUnprocessableEntity, agentapi.ErrorResponse{
					Code: "validation_error", Message: validationError.Message,
				})
				return
			}
			logger.Error("failed to ensure account", "error", err, "accountId", request.AccountId)
			httpx.WriteJSON(w, http.StatusInternalServerError, agentapi.ErrorResponse{
				Code: "account_reconcile_failed", Message: "The account could not be reconciled",
			})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
	for _, route := range []struct {
		path string
		name string
	}{
		{path: "POST /agent/v1/accounts/disable", name: "disable"},
		{path: "POST /agent/v1/accounts/enable", name: "enable"},
		{path: "POST /agent/v1/accounts/delete", name: "delete"},
	} {
		item := route
		mux.HandleFunc(item.path, func(w http.ResponseWriter, r *http.Request) {
			var request agentapi.AccountActionRequest
			if err := httpx.DecodeJSON(w, r, &request); err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, agentapi.ErrorResponse{
					Code: "invalid_json", Message: "The request body is invalid",
				})
				return
			}
			if options.AccountActions == nil || hostAccountInvalid(request.AccountId, request.SystemUser) {
				httpx.WriteJSON(w, http.StatusUnprocessableEntity, agentapi.ErrorResponse{
					Code: "validation_error", Message: "The account identity is invalid",
				})
				return
			}
			var err error
			switch item.name {
			case "disable":
				err = options.AccountActions.Disable(r.Context(), request.AccountId, request.SystemUser)
			case "enable":
				err = options.AccountActions.Enable(r.Context(), request.AccountId, request.SystemUser)
			case "delete":
				err = options.AccountActions.Delete(r.Context(), request.AccountId, request.SystemUser)
			}
			if err != nil {
				logger.Error("failed to "+item.name+" account", "error", err, "accountId", request.AccountId)
				httpx.WriteJSON(w, http.StatusInternalServerError, agentapi.ErrorResponse{
					Code: "account_action_failed", Message: "The account operation failed",
				})
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})
	}
	handleDatabase := func(delete bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var request agentapi.DatabaseRequest
			if err := httpx.DecodeJSON(w, r, &request); err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, agentapi.ErrorResponse{Code: "invalid_json", Message: "The request body is invalid"})
				return
			}
			if options.Databases == nil || validate.DatabaseSystemName(request.Name) != nil {
				httpx.WriteJSON(w, http.StatusUnprocessableEntity, agentapi.ErrorResponse{Code: "validation_error", Message: "The database request is invalid"})
				return
			}
			var err error
			if delete {
				err = options.Databases.DeleteDatabase(r.Context(), request.Name)
			} else {
				err = options.Databases.EnsureDatabase(r.Context(), request.Name)
			}
			if err != nil {
				logger.Error("database operation failed", "error", err)
				httpx.WriteJSON(w, http.StatusInternalServerError, agentapi.ErrorResponse{Code: "database_failed", Message: "The database operation failed"})
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	}
	mux.HandleFunc("POST /agent/v1/databases", handleDatabase(false))
	mux.HandleFunc("DELETE /agent/v1/databases", handleDatabase(true))
	handleDatabaseUser := func(delete bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var request agentapi.DatabaseUserRequest
			if err := httpx.DecodeJSON(w, r, &request); err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, agentapi.ErrorResponse{Code: "invalid_json", Message: "The request body is invalid"})
				return
			}
			if options.Databases == nil || validate.DatabaseSystemName(request.Name) != nil || (!delete && request.Password == nil) {
				httpx.WriteJSON(w, http.StatusUnprocessableEntity, agentapi.ErrorResponse{Code: "validation_error", Message: "The database user request is invalid"})
				return
			}
			var err error
			if delete {
				err = options.Databases.DeleteUser(r.Context(), request.Name)
			} else {
				err = options.Databases.EnsureUser(r.Context(), request.Name, *request.Password)
			}
			if err != nil {
				logger.Error("database user operation failed", "error", err)
				httpx.WriteJSON(w, http.StatusInternalServerError, agentapi.ErrorResponse{Code: "database_user_failed", Message: "The database user operation failed"})
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	}
	mux.HandleFunc("POST /agent/v1/database-users", handleDatabaseUser(false))
	mux.HandleFunc("DELETE /agent/v1/database-users", handleDatabaseUser(true))
	handleGrant := func(delete bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var request agentapi.DatabaseGrantRequest
			if err := httpx.DecodeJSON(w, r, &request); err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, agentapi.ErrorResponse{Code: "invalid_json", Message: "The request body is invalid"})
				return
			}
			if options.Databases == nil || validate.DatabaseSystemName(request.Database) != nil || validate.DatabaseSystemName(request.User) != nil {
				httpx.WriteJSON(w, http.StatusUnprocessableEntity, agentapi.ErrorResponse{Code: "validation_error", Message: "The database grant request is invalid"})
				return
			}
			var err error
			if delete {
				err = options.Databases.DeleteGrant(r.Context(), request.Database, request.User)
			} else {
				err = options.Databases.EnsureGrant(r.Context(), request.Database, request.User)
			}
			if err != nil {
				logger.Error("database grant operation failed", "error", err)
				httpx.WriteJSON(w, http.StatusInternalServerError, agentapi.ErrorResponse{Code: "database_grant_failed", Message: "The database grant operation failed"})
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	}
	mux.HandleFunc("POST /agent/v1/database-grants", handleGrant(false))
	mux.HandleFunc("DELETE /agent/v1/database-grants", handleGrant(true))
	mux.HandleFunc("PUT /agent/v1/cron", func(w http.ResponseWriter, r *http.Request) {
		var request agentapi.SyncCronRequest
		if err := httpx.DecodeJSON(w, r, &request); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, agentapi.ErrorResponse{Code: "invalid_json", Message: "The request body is invalid"})
			return
		}
		if options.Cron == nil || hostAccountInvalid(request.AccountId, request.SystemUser) {
			httpx.WriteJSON(w, http.StatusUnprocessableEntity, agentapi.ErrorResponse{Code: "validation_error", Message: "The cron request is invalid"})
			return
		}
		entries := make([]scheduler.Entry, 0, len(request.Entries))
		for _, entry := range request.Entries {
			entries = append(entries, scheduler.Entry{ID: entry.Id, Schedule: entry.Schedule, Command: entry.Command})
		}
		if err := options.Cron.Sync(r.Context(), request.AccountId, request.SystemUser, entries); err != nil {
			logger.Error("cron synchronization failed", "error", err, "accountId", request.AccountId)
			httpx.WriteJSON(w, http.StatusInternalServerError, agentapi.ErrorResponse{Code: "cron_sync_failed", Message: "The cron operation failed"})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /agent/v1/certificates", func(w http.ResponseWriter, r *http.Request) {
		var request agentapi.IssueCertificateRequest
		if err := httpx.DecodeJSON(w, r, &request); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, agentapi.ErrorResponse{Code: "invalid_json", Message: "The request body is invalid"})
			return
		}
		if options.Certificates == nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, agentapi.ErrorResponse{Code: "internal_error", Message: "Certificate driver is not configured"})
			return
		}
		value := agentcertificate.Request{
			ID: request.CertificateId, Kind: string(request.Kind), Name: request.Name,
			Names: request.Names, Email: string(request.Email), RedirectHTTPS: request.RedirectHttps,
		}
		if request.DomainId != nil {
			value.DomainID = *request.DomainId
		}
		if request.AccountId != nil {
			value.AccountID = *request.AccountId
		}
		if request.SystemUser != nil {
			value.SystemUser = *request.SystemUser
		}
		if request.PhpVersion != nil {
			value.PHPVersion = string(*request.PhpVersion)
		}
		result, err := options.Certificates.Issue(r.Context(), value)
		if err != nil {
			logger.Error("certificate issue failed", "error", err, "certificateId", request.CertificateId)
			httpx.WriteJSON(w, http.StatusInternalServerError, agentapi.ErrorResponse{Code: "certificate_issue_failed", Message: "The certificate operation failed"})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, agentapi.CertificateResult{Names: result.Names, ExpiresAt: result.ExpiresAt})
	})
	mux.HandleFunc("POST /agent/v1/backups", func(w http.ResponseWriter, r *http.Request) {
		var request agentapi.CreateBackupRequest
		if err := httpx.DecodeJSON(w, r, &request); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, agentapi.ErrorResponse{Code: "invalid_json", Message: "The request body is invalid"})
			return
		}
		if options.Backups == nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, agentapi.ErrorResponse{Code: "internal_error", Message: "Backup driver is not configured"})
			return
		}
		result, err := options.Backups.Create(r.Context(), agentbackup.CreateRequest{RunID: request.RunId, AccountID: request.AccountId, SystemUser: request.SystemUser, IncludeFiles: request.IncludeFiles, Databases: request.Databases, Metadata: request.Metadata})
		if err != nil {
			writeBackupFailure(w, logger, err, "create")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, agentapi.BackupArtifactResult{Path: result.Path, Checksum: result.Checksum, Size: result.Size, Manifest: manifestResponse(result.Manifest)})
	})
	mux.HandleFunc("POST /agent/v1/backups/preview", func(w http.ResponseWriter, r *http.Request) {
		var request agentapi.BackupArtifactRequest
		if err := httpx.DecodeJSON(w, r, &request); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, agentapi.ErrorResponse{Code: "invalid_json", Message: "The request body is invalid"})
			return
		}
		if options.Backups == nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, agentapi.ErrorResponse{Code: "internal_error", Message: "Backup driver is not configured"})
			return
		}
		result, err := options.Backups.Preview(r.Context(), agentbackup.ArtifactRequest{AccountID: request.AccountId, Path: request.Path, Checksum: request.Checksum})
		if err != nil {
			writeBackupFailure(w, logger, err, "preview")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, manifestResponse(result))
	})
	mux.HandleFunc("POST /agent/v1/backups/restore", func(w http.ResponseWriter, r *http.Request) {
		var request agentapi.RestoreBackupRequest
		if err := httpx.DecodeJSON(w, r, &request); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, agentapi.ErrorResponse{Code: "invalid_json", Message: "The request body is invalid"})
			return
		}
		if options.Backups == nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, agentapi.ErrorResponse{Code: "internal_error", Message: "Backup driver is not configured"})
			return
		}
		metadata, err := options.Backups.Restore(r.Context(), agentbackup.RestoreRequest{ArtifactRequest: agentbackup.ArtifactRequest{AccountID: request.AccountId, Path: request.Path, Checksum: request.Checksum}, SystemUser: request.SystemUser, Files: request.Files, Databases: request.Databases, Metadata: request.Metadata})
		if err != nil {
			writeBackupFailure(w, logger, err, "restore")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, agentapi.RestoreBackupResult{Metadata: metadata})
	})
	mux.HandleFunc("DELETE /agent/v1/backups", func(w http.ResponseWriter, r *http.Request) {
		var request agentapi.BackupArtifactRequest
		if err := httpx.DecodeJSON(w, r, &request); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, agentapi.ErrorResponse{Code: "invalid_json", Message: "The request body is invalid"})
			return
		}
		if options.Backups == nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, agentapi.ErrorResponse{Code: "internal_error", Message: "Backup driver is not configured"})
			return
		}
		if err := options.Backups.Delete(r.Context(), agentbackup.ArtifactRequest{AccountID: request.AccountId, Path: request.Path, Checksum: request.Checksum}); err != nil {
			writeBackupFailure(w, logger, err, "delete")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /agent/v1/domains", func(w http.ResponseWriter, r *http.Request) {
		var request agentapi.EnsureDomainRequest
		if err := httpx.DecodeJSON(w, r, &request); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, agentapi.ErrorResponse{
				Code: "invalid_json", Message: "The request body is invalid",
			})
			return
		}
		aliases, domainErr := validate.DomainAliases(request.Name, request.Aliases)
		if validate.ID("accountId", request.AccountId) != nil ||
			validate.ID("domainId", request.DomainId) != nil ||
			validate.SystemUser(request.SystemUser) != nil ||
			!request.PhpVersion.Valid() ||
			domainErr != nil {
			httpx.WriteJSON(w, http.StatusUnprocessableEntity, agentapi.ErrorResponse{
				Code: "validation_error", Message: "The domain request is invalid",
			})
			return
		}
		if options.Domains == nil {
			logger.Error("domain manager is not configured")
			httpx.WriteJSON(w, http.StatusInternalServerError, agentapi.ErrorResponse{
				Code: "internal_error", Message: "An internal error occurred",
			})
			return
		}
		if err := options.Domains.Ensure(
			r.Context(), request.AccountId, request.SystemUser, request.DomainId,
			request.Name, string(request.PhpVersion), aliases,
		); err != nil {
			writeDomainFailure(w, logger, err, request.DomainId, "reconcile", "domain_reconcile_failed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /agent/v1/domains/disable", func(w http.ResponseWriter, r *http.Request) {
		var request agentapi.DisableDomainRequest
		if err := httpx.DecodeJSON(w, r, &request); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, agentapi.ErrorResponse{
				Code: "invalid_json", Message: "The request body is invalid",
			})
			return
		}
		if validate.ID("accountId", request.AccountId) != nil ||
			validate.ID("domainId", request.DomainId) != nil ||
			validate.SystemUser(request.SystemUser) != nil {
			httpx.WriteJSON(w, http.StatusUnprocessableEntity, agentapi.ErrorResponse{
				Code: "validation_error", Message: "The domain request is invalid",
			})
			return
		}
		if options.Domains == nil {
			logger.Error("domain manager is not configured")
			httpx.WriteJSON(w, http.StatusInternalServerError, agentapi.ErrorResponse{
				Code: "internal_error", Message: "An internal error occurred",
			})
			return
		}
		if err := options.Domains.Disable(
			r.Context(), request.AccountId, request.SystemUser, request.DomainId,
		); err != nil {
			writeDomainFailure(w, logger, err, request.DomainId, "disable", "domain_disable_failed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /agent/v1/domains/delete", func(w http.ResponseWriter, r *http.Request) {
		var request agentapi.DeleteDomainRequest
		if err := httpx.DecodeJSON(w, r, &request); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, agentapi.ErrorResponse{
				Code: "invalid_json", Message: "The request body is invalid",
			})
			return
		}
		name, domainErr := validate.Domain(request.Name)
		if validate.ID("accountId", request.AccountId) != nil ||
			validate.ID("domainId", request.DomainId) != nil ||
			validate.SystemUser(request.SystemUser) != nil ||
			domainErr != nil || name != request.Name {
			httpx.WriteJSON(w, http.StatusUnprocessableEntity, agentapi.ErrorResponse{
				Code: "validation_error", Message: "The domain request is invalid",
			})
			return
		}
		if options.Domains == nil {
			logger.Error("domain manager is not configured")
			httpx.WriteJSON(w, http.StatusInternalServerError, agentapi.ErrorResponse{
				Code: "internal_error", Message: "An internal error occurred",
			})
			return
		}
		if err := options.Domains.Delete(
			r.Context(), request.AccountId, request.SystemUser, request.DomainId, request.Name,
		); err != nil {
			writeDomainFailure(w, logger, err, request.DomainId, "delete", "domain_delete_failed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /agent/v1/domains/rename", func(w http.ResponseWriter, r *http.Request) {
		var request agentapi.RenameDomainRequest
		if err := httpx.DecodeJSON(w, r, &request); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, agentapi.ErrorResponse{
				Code: "invalid_json", Message: "The request body is invalid",
			})
			return
		}
		current, currentErr := validate.Domain(request.CurrentName)
		aliases, domainErr := validate.DomainAliases(request.Name, request.Aliases)
		if validate.ID("accountId", request.AccountId) != nil ||
			validate.ID("domainId", request.DomainId) != nil ||
			validate.SystemUser(request.SystemUser) != nil ||
			!request.PhpVersion.Valid() || currentErr != nil ||
			current != request.CurrentName || current == request.Name || domainErr != nil {
			httpx.WriteJSON(w, http.StatusUnprocessableEntity, agentapi.ErrorResponse{
				Code: "validation_error", Message: "The domain request is invalid",
			})
			return
		}
		if options.Domains == nil {
			logger.Error("domain manager is not configured")
			httpx.WriteJSON(w, http.StatusInternalServerError, agentapi.ErrorResponse{
				Code: "internal_error", Message: "An internal error occurred",
			})
			return
		}
		if err := options.Domains.Rename(
			r.Context(), request.AccountId, request.SystemUser, request.DomainId,
			request.CurrentName, request.Name, string(request.PhpVersion), aliases,
		); err != nil {
			writeDomainFailure(w, logger, err, request.DomainId, "rename", "domain_rename_failed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	return mux
}

func hostAccountInvalid(accountID, systemUser string) bool {
	return validate.ID("accountId", accountID) != nil || validate.SystemUser(systemUser) != nil
}

func manifestResponse(value backupfmt.Manifest) agentapi.BackupManifest {
	entries := make([]agentapi.BackupEntry, 0, len(value.Entries))
	for _, entry := range value.Entries {
		entries = append(entries, agentapi.BackupEntry{Path: entry.Path, Size: entry.Size, Checksum: entry.Checksum})
	}
	return agentapi.BackupManifest{Version: value.Version, RunId: value.RunID, AccountId: value.AccountID, CreatedAt: value.CreatedAt, Files: value.Files, Databases: value.Databases, Metadata: value.Metadata, Entries: entries}
}

func writeBackupFailure(w http.ResponseWriter, logger *slog.Logger, err error, operation string) {
	var validationError *validate.Error
	if errors.As(err, &validationError) {
		httpx.WriteJSON(w, http.StatusUnprocessableEntity, agentapi.ErrorResponse{Code: "validation_error", Message: validationError.Message})
		return
	}
	logger.Error("backup operation failed", "operation", operation, "error", err)
	httpx.WriteJSON(w, http.StatusInternalServerError, agentapi.ErrorResponse{Code: "backup_failed", Message: "The backup operation failed"})
}

func writeDomainFailure(
	w http.ResponseWriter,
	logger *slog.Logger,
	err error,
	domainID, operation, code string,
) {
	var validationError *validate.Error
	if errors.As(err, &validationError) {
		httpx.WriteJSON(w, http.StatusUnprocessableEntity, agentapi.ErrorResponse{
			Code: "validation_error", Message: validationError.Message,
		})
		return
	}
	logger.Error("failed to "+operation+" domain", "error", err, "domainId", domainID)
	httpx.WriteJSON(w, http.StatusInternalServerError, agentapi.ErrorResponse{
		Code: code, Message: "The domain operation failed",
	})
}
