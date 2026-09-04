package httpapi

import (
	"net/http"

	"github.com/GVALFER/WEBYCP/internal/auth"
	publicapi "github.com/GVALFER/WEBYCP/internal/httpapi/spec"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/services"
)

func (h *handler) getServiceSettings(
	w http.ResponseWriter, r *http.Request, _ auth.Session,
) {
	value, err := h.options.Services.Settings(r.Context())
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, serviceSettingsResponse(value))
}

func (h *handler) updateServiceSettings(
	w http.ResponseWriter, r *http.Request, session auth.Session,
) {
	if session.User.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "Administrator access is required")
		return
	}
	var request publicapi.ServiceDefaults
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	value, err := h.options.Services.Update(r.Context(), services.Defaults{
		WebDriver: string(request.WebDriver), RuntimeDriver: string(request.RuntimeDriver),
		RuntimeVersion: string(request.RuntimeVersion), DatabaseDriver: string(request.DatabaseDriver),
		SchedulerDriver: string(request.SchedulerDriver), BackupDriver: string(request.BackupDriver),
	})
	h.recordMutation(r, session.User.ID, "service_settings.update", "service_settings", "global", err)
	if err != nil {
		if writeValidationError(w, err) {
			return
		}
		h.internalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, serviceSettingsResponse(value))
}

func serviceSettingsResponse(value services.Settings) publicapi.ServiceSettings {
	return publicapi.ServiceSettings{
		Defaults: publicapi.ServiceDefaults{
			WebDriver:       publicapi.ServiceDefaultsWebDriver(value.Defaults.WebDriver),
			RuntimeDriver:   publicapi.ServiceDefaultsRuntimeDriver(value.Defaults.RuntimeDriver),
			RuntimeVersion:  publicapi.ServiceDefaultsRuntimeVersion(value.Defaults.RuntimeVersion),
			DatabaseDriver:  publicapi.ServiceDefaultsDatabaseDriver(value.Defaults.DatabaseDriver),
			SchedulerDriver: publicapi.ServiceDefaultsSchedulerDriver(value.Defaults.SchedulerDriver),
			BackupDriver:    publicapi.ServiceDefaultsBackupDriver(value.Defaults.BackupDriver),
		},
		UpdatedAt: value.UpdatedAt,
	}
}
