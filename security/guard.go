package security

import (
	"strings"

	"github.com/Lucas-Lopes-II/govalidator/domainerr"
	"github.com/Lucas-Lopes-II/govalidator/is"
)

// RequireUUID trims value and validates it as an RFC 4122 UUID.
// Returns the trimmed value on success, or a *[domainerr.BadRequestErr] on failure.
//
// Use this in HTTP handlers to validate path or query parameters:
//
//	id, err := security.RequireUUID(r.PathValue("id"), "id")
//	if err != nil {
//	    domainerr.WriteError(w, err)
//	    return
//	}
func RequireUUID(value, paramName string) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", domainerr.NewBadRequest(paramName + " is required")
	}
	if !is.UUID(v) {
		return "", domainerr.NewBadRequest(paramName + " must be a valid UUID")
	}
	return v, nil
}

// SafeSortField returns field if it is present in allowed, or defaultField
// otherwise. It never returns an error — invalid or absent values silently fall
// back to the default, preventing SQL ORDER BY injection via user-supplied
// column names.
//
// allowed should be the exact set of column names the query layer accepts:
//
//	allowed := map[string]struct{}{"name": {}, "created_at": {}, "email": {}}
//	col := security.SafeSortField(r.URL.Query().Get("sort"), allowed, "created_at")
func SafeSortField(field string, allowed map[string]struct{}, defaultField string) string {
	if _, ok := allowed[field]; ok {
		return field
	}
	return defaultField
}

// SafePageSize returns size when 1 ≤ size ≤ maxAllowed, or defaultSize
// otherwise. It never returns an error — out-of-range values silently fall
// back to the default.
//
//	page := security.SafePageSize(
//	    strconv.Atoi(r.URL.Query().Get("limit")),
//	    20,   // defaultSize
//	    100,  // maxAllowed
//	)
func SafePageSize(size, defaultSize, maxAllowed int) int {
	if size >= 1 && size <= maxAllowed {
		return size
	}
	return defaultSize
}
