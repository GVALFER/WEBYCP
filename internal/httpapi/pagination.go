package httpapi

import (
	"net/http"

	publicapi "github.com/GVALFER/WEBYCP/internal/httpapi/spec"
	"github.com/GVALFER/WEBYCP/internal/pagination"
)

func requestPage(w http.ResponseWriter, r *http.Request) (pagination.Query, bool) {
	query, err := pagination.Parse(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
		return pagination.Query{}, false
	}
	return query, true
}

func paginationResponse(query pagination.Query, total int64) publicapi.Pagination {
	return publicapi.Pagination{
		Page:       query.Page,
		Size:       query.Size,
		TotalItems: total,
		TotalPages: pagination.TotalPages(total, query.Size),
	}
}
