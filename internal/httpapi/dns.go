package httpapi

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/GVALFER/WEBYCP/internal/accounts"
	"github.com/GVALFER/WEBYCP/internal/auth"
	dnscontrol "github.com/GVALFER/WEBYCP/internal/dns"
	publicapi "github.com/GVALFER/WEBYCP/internal/httpapi/spec"
	"github.com/GVALFER/WEBYCP/internal/httpx"
	"github.com/GVALFER/WEBYCP/internal/jobs"
)

func (h *handler) listDNSProviders(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	values, err := h.options.DNS.Providers(r.Context())
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response := publicapi.DNSProviderListResponse{Items: make([]publicapi.DNSProvider, 0, len(values))}
	for _, value := range values {
		response.Items = append(response.Items, dnsProviderResponse(value))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) getDNSSettings(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	value, err := h.options.DNS.Settings(r.Context())
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, dnsSettingsResponse(value))
}

func (h *handler) updateDNSSettings(w http.ResponseWriter, r *http.Request, session auth.Session) {
	if session.User.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "Administrator access is required")
		return
	}
	var request publicapi.UpdateDNSSettingsRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	value, err := h.options.DNS.UpdateSettings(r.Context(), dnscontrol.Settings{
		PrimaryNameserver: request.PrimaryNameserver, SecondaryNameserver: request.SecondaryNameserver,
		DefaultTTL: request.DefaultTtl,
	})
	h.recordMutation(r, session.User.ID, "dns_settings.update", "dns_settings", "global", err)
	if err != nil {
		if writeValidationError(w, err) {
			return
		}
		h.internalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, dnsSettingsResponse(value))
}

func (h *handler) listDNSZones(w http.ResponseWriter, r *http.Request, session auth.Session) {
	query, ok := requestPage(w, r)
	if !ok {
		return
	}
	page, err := h.options.DNS.ZonePage(r.Context(), session.User.ID, session.User.Role == "admin", query)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	response := publicapi.DNSZoneListResponse{
		Items:      make([]publicapi.DNSZone, 0, len(page.Items)),
		Pagination: paginationResponse(page.Query, page.Total),
	}
	for _, value := range page.Items {
		response.Items = append(response.Items, dnsZoneResponse(value))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) getDNSZone(w http.ResponseWriter, r *http.Request, session auth.Session) {
	value, err := h.options.DNS.Zone(r.Context(), r.PathValue("dnsZoneId"), session.User.ID, session.User.Role == "admin")
	if err != nil {
		h.writeDNSError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, dnsZoneResponse(value))
}

func (h *handler) createDNSZone(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.CreateDNSZoneRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	value, job, err := h.options.DNS.CreateZone(
		r.Context(), request.AccountId, request.ProviderId, request.Name,
		session.User.ID, session.User.Role == "admin",
	)
	h.recordMutation(r, session.User.ID, "dns_zone.create", "dns_zone", value.ID, err)
	if err != nil {
		h.writeDNSError(w, r, err)
		return
	}
	writeDNSZoneJob(w, value, job)
}

func (h *handler) deleteDNSZone(w http.ResponseWriter, r *http.Request, session auth.Session) {
	id := r.PathValue("dnsZoneId")
	value, job, err := h.options.DNS.DeleteZone(r.Context(), id, session.User.ID, session.User.Role == "admin")
	h.recordMutation(r, session.User.ID, "dns_zone.delete", "dns_zone", id, err)
	if err != nil {
		h.writeDNSError(w, r, err)
		return
	}
	writeDNSZoneJob(w, value, job)
}

func (h *handler) listDNSRecords(w http.ResponseWriter, r *http.Request, session auth.Session) {
	query, ok := requestPage(w, r)
	if !ok {
		return
	}
	page, err := h.options.DNS.RecordPage(
		r.Context(), r.PathValue("dnsZoneId"), session.User.ID,
		session.User.Role == "admin", query,
	)
	if err != nil {
		h.writeDNSError(w, r, err)
		return
	}
	response := publicapi.DNSRecordListResponse{
		Items:      make([]publicapi.DNSRecord, 0, len(page.Items)),
		Pagination: paginationResponse(page.Query, page.Total),
	}
	for _, value := range page.Items {
		response.Items = append(response.Items, dnsRecordResponse(value))
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) createDNSRecord(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.WriteDNSRecordRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	value, job, err := h.options.DNS.CreateRecord(
		r.Context(), r.PathValue("dnsZoneId"), request.Name, string(request.Type),
		request.Content, request.Ttl, request.Priority, session.User.ID, session.User.Role == "admin",
	)
	h.recordMutation(r, session.User.ID, "dns_record.create", "dns_record", value.ID, err)
	if err != nil {
		h.writeDNSError(w, r, err)
		return
	}
	writeDNSRecordJob(w, value, job)
}

func (h *handler) updateDNSRecord(w http.ResponseWriter, r *http.Request, session auth.Session) {
	var request publicapi.WriteDNSRecordRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body is invalid")
		return
	}
	id := r.PathValue("dnsRecordId")
	value, job, err := h.options.DNS.UpdateRecord(
		r.Context(), id, request.Name, string(request.Type), request.Content,
		request.Ttl, request.Priority, session.User.ID, session.User.Role == "admin",
	)
	h.recordMutation(r, session.User.ID, "dns_record.update", "dns_record", id, err)
	if err != nil {
		h.writeDNSError(w, r, err)
		return
	}
	writeDNSRecordJob(w, value, job)
}

func (h *handler) deleteDNSRecord(w http.ResponseWriter, r *http.Request, session auth.Session) {
	id := r.PathValue("dnsRecordId")
	value, job, err := h.options.DNS.DeleteRecord(r.Context(), id, session.User.ID, session.User.Role == "admin")
	h.recordMutation(r, session.User.ID, "dns_record.delete", "dns_record", id, err)
	if err != nil {
		h.writeDNSError(w, r, err)
		return
	}
	writeDNSRecordJob(w, value, job)
}

func (h *handler) writeDNSError(w http.ResponseWriter, r *http.Request, err error) {
	if writeValidationError(w, err) {
		return
	}
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, "not_found", "DNS resource not found")
	case errors.Is(err, accounts.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "Account access is required")
	case errors.Is(err, accounts.ErrBusy):
		writeError(w, http.StatusConflict, "account_inactive", "The account must be active")
	case errors.Is(err, dnscontrol.ErrBusy):
		writeError(w, http.StatusConflict, "dns_busy", err.Error())
	case errors.Is(err, dnscontrol.ErrNameExists):
		writeError(w, http.StatusConflict, "dns_zone_exists", err.Error())
	case errors.Is(err, dnscontrol.ErrRecordExists):
		writeError(w, http.StatusConflict, "dns_record_exists", err.Error())
	case errors.Is(err, dnscontrol.ErrRecordConflict):
		writeError(w, http.StatusConflict, "dns_record_conflict", err.Error())
	case errors.Is(err, dnscontrol.ErrNotConfigured):
		writeError(w, http.StatusConflict, "dns_not_configured", err.Error())
	default:
		h.internalError(w, r, err)
	}
}

func dnsProviderResponse(value dnscontrol.Provider) publicapi.DNSProvider {
	return publicapi.DNSProvider{
		Id: value.ID, NodeId: value.NodeID, Name: value.Name,
		Driver: publicapi.DNSProviderDriver(value.Driver), Status: publicapi.DNSProviderStatus(value.Status),
		CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func dnsSettingsResponse(value dnscontrol.Settings) publicapi.DNSSettings {
	return publicapi.DNSSettings{
		PrimaryNameserver: value.PrimaryNameserver, SecondaryNameserver: value.SecondaryNameserver,
		DefaultTtl: value.DefaultTTL, UpdatedAt: value.UpdatedAt,
	}
}

func dnsZoneResponse(value dnscontrol.Zone) publicapi.DNSZone {
	return publicapi.DNSZone{
		Id: value.ID, AccountId: value.AccountID, NodeId: value.NodeID,
		ProviderId: value.ProviderID, Name: value.Name, Status: publicapi.DNSZoneStatus(value.Status),
		Nameservers: []string{value.PrimaryNameserver, value.SecondaryNameserver},
		CreatedAt:   value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func dnsRecordResponse(value dnscontrol.Record) publicapi.DNSRecord {
	return publicapi.DNSRecord{
		Id: value.ID, ZoneId: value.ZoneID, Name: value.Name,
		Type: publicapi.DNSRecordType(value.Type), Content: value.Content,
		Ttl: value.TTL, Priority: value.Priority, Status: publicapi.DNSRecordStatus(value.Status),
		CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func writeDNSZoneJob(w http.ResponseWriter, value dnscontrol.Zone, job jobs.Job) {
	httpx.WriteJSON(w, http.StatusAccepted, publicapi.DNSZoneJobResponse{
		Zone: dnsZoneResponse(value), Job: jobResponse(job),
	})
}

func writeDNSRecordJob(w http.ResponseWriter, value dnscontrol.Record, job jobs.Job) {
	httpx.WriteJSON(w, http.StatusAccepted, publicapi.DNSRecordJobResponse{
		Record: dnsRecordResponse(value), Job: jobResponse(job),
	})
}
