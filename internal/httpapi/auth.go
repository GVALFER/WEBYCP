package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/GVALFER/WEBYCP/internal/audit"
	"github.com/GVALFER/WEBYCP/internal/auth"
	"github.com/GVALFER/WEBYCP/internal/httpapi/spec"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/idgen"
	"github.com/GVALFER/WEBYCP/internal/validate"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	var request publicapi.LoginRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}

	session, err := h.options.Auth.Login(r.Context(), request.Username, request.Password)
	if err != nil {
		h.record(r, audit.Event{Action: "auth.login", ResourceType: "session", Result: "failure"})
		if errors.Is(err, auth.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "Username or password is incorrect")
			return
		}
		h.internalError(w, r, err)
		return
	}

	h.setSessionCookie(w, session)
	h.record(r, audit.Event{
		UserID: session.User.ID, Action: "auth.login", ResourceType: "session",
		ResourceID: session.ID, Result: "success",
	})
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, authResponse(session))
}

func (h *handler) updateProfile(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.UpdateProfileRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	currentPassword := ""
	if request.CurrentPassword != nil {
		currentPassword = *request.CurrentPassword
	}
	password := ""
	if request.Password != nil {
		password = *request.Password
	}
	updated, err := h.options.Auth.UpdateProfile(
		r.Context(), session, request.Username, request.Name, string(request.Email),
		request.Timezone, currentPassword, password,
	)
	if err != nil {
		h.record(r, audit.Event{
			UserID: session.User.ID, Action: "auth.profile.update",
			ResourceType: "user", ResourceID: session.User.ID, Result: "failure",
		})
		if writeValidationError(w, err) {
			return
		}
		switch {
		case errors.Is(err, auth.ErrCurrentPassword):
			writeError(w, http.StatusForbidden, "invalid_current_password", "Current password is incorrect")
		case errors.Is(err, auth.ErrUsernameExists):
			writeError(w, http.StatusConflict, "username_exists", "Username is already in use")
		case errors.Is(err, auth.ErrEmailExists):
			writeError(w, http.StatusConflict, "email_exists", "Email address is already in use")
		case errors.Is(err, auth.ErrUnauthorized):
			writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		default:
			h.internalError(w, r, err)
		}
		return
	}
	h.record(r, audit.Event{
		UserID: session.User.ID, Action: "auth.profile.update",
		ResourceType: "user", ResourceID: session.User.ID, Result: "success",
	})
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, authResponse(updated))
}

func (h *handler) logout(w http.ResponseWriter, r *http.Request, session auth.Session) {
	cookie, _ := r.Cookie(sessionCookie)
	if err := h.options.Auth.Logout(r.Context(), cookie.Value); err != nil {
		h.internalError(w, r, err)
		return
	}
	h.record(r, audit.Event{
		UserID: session.User.ID, Action: "auth.logout", ResourceType: "session",
		ResourceID: session.ID, Result: "success",
	})
	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) me(w http.ResponseWriter, _ *http.Request, session auth.Session) {
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, authResponse(session))
}

func (h *handler) setSessionCookie(w http.ResponseWriter, session auth.Session) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: session.Token, Path: "/", Expires: session.ExpiresAt,
		MaxAge: int(time.Until(session.ExpiresAt).Seconds()), HttpOnly: true,
		Secure: h.options.SecureCookie, SameSite: http.SameSiteStrictMode,
	})
}

func (h *handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Path: "/", MaxAge: -1, HttpOnly: true,
		Secure: h.options.SecureCookie, SameSite: http.SameSiteStrictMode,
	})
}

func (h *handler) record(r *http.Request, event audit.Event) {
	if h.options.Audit == nil {
		return
	}
	id, err := idgen.ID()
	if err != nil {
		h.logger.Error("failed to generate audit id", "error", err)
		return
	}
	event.ID = id
	event.Metadata = "{}"
	event.CreatedAt = time.Now().UTC()
	if err := h.options.Audit.Record(r.Context(), event); err != nil {
		h.logger.Error("failed to record audit event", "error", err, "action", event.Action)
	}
}

func (h *handler) recordMutation(
	r *http.Request, userID, action, resourceType, resourceID string, err error,
) {
	h.recordJobMutation(r, userID, action, resourceType, resourceID, "", err)
}

func writeValidationError(w http.ResponseWriter, err error) bool {
	var validationError *validate.Error
	if !errors.As(err, &validationError) {
		return false
	}
	fields := map[string]string{validationError.Field: validationError.Message}
	httpx.WriteJSON(w, http.StatusUnprocessableEntity, publicapi.ErrorResponse{
		Code: "validation_error", Message: "Check the highlighted fields", Fields: &fields,
	})
	return true
}

func authResponse(session auth.Session) publicapi.AuthResponse {
	return publicapi.AuthResponse{
		CsrfToken: session.CSRFToken,
		ExpiresAt: session.ExpiresAt,
		Timezone:  session.User.Timezone,
		User: publicapi.User{
			Id: session.User.ID, Username: session.User.Username,
			Email: openapi_types.Email(session.User.Email), Name: session.User.Name,
			Role:               publicapi.UserRole(session.User.Role),
			MustChangePassword: session.User.MustChangePassword, CreatedAt: session.User.CreatedAt,
		},
	}
}
