package domainerr

import "errors"

// ErrorResponse is the JSON body returned for all HTTP error responses.
//
// Example (field validation failure):
//
//	{
//	  "status": 400,
//	  "message": "Bad Request",
//	  "errors": ["name is required", "email is invalid"],
//	  "displayable": false
//	}
type ErrorResponse struct {
	Status      int      `json:"status"`
	Message     string   `json:"message"`
	Errors      []string `json:"errors,omitempty"`
	Displayable bool     `json:"displayable"`
}

// FromDomainError converts a [DomainError] into an [ErrorResponse].
// The Messages() slice is already a defensive copy (guaranteed by baseErr).
func FromDomainError(de DomainError) ErrorResponse {
	return ErrorResponse{
		Status:      de.StatusCode(),
		Message:     de.Error(),
		Errors:      de.Messages(), // baseErr.Messages() returns a copy
		Displayable: de.Displayable(),
	}
}

// FromError converts any error into an [ErrorResponse].
//   - If err implements [DomainError] (directly or wrapped), delegates to [FromDomainError].
//   - Otherwise returns a generic 500 Internal Server Error.
//
// NEVER exposes the original message of non-DomainError errors to the caller.
// This prevents accidental leakage of internal details (e.g. SQL errors, stack traces).
func FromError(err error) ErrorResponse {
	var de DomainError
	if errors.As(err, &de) {
		return FromDomainError(de)
	}
	return ErrorResponse{Status: 500, Message: "Internal Server Error"}
}

// ─── RFC 7807 Problem Details (opt-in) ───────────────────────────────────────

// ProblemDetail represents an RFC 7807 "Problem Details for HTTP APIs" response.
// Use this when your API contract requires RFC 7807 compliance.
// Serve with Content-Type: application/problem+json.
type ProblemDetail struct {
	// Type is a URI reference identifying the problem type.
	Type string `json:"type"`
	// Title is a short, human-readable summary of the problem type.
	Title string `json:"title"`
	// Status mirrors the HTTP status code for clients that read the body only.
	Status int `json:"status"`
	// Detail is a human-readable explanation of this specific occurrence.
	Detail string `json:"detail"`
	// Instance is an optional URI identifying the specific request that failed.
	Instance string `json:"instance,omitempty"`
	// Errors lists individual field-level validation messages (extension field).
	Errors []string `json:"errors,omitempty"`
}

// statusMeta holds the URI slug and display title for a given HTTP status code.
type statusMeta struct {
	slug  string
	title string
}

// knownStatuses maps HTTP status codes to their RFC 7807 slug and title.
var knownStatuses = map[int]statusMeta{
	400: {"/bad-request", "Bad Request"},
	401: {"/unauthorized", "Unauthorized"},
	403: {"/forbidden", "Forbidden"},
	404: {"/not-found", "Not Found"},
	409: {"/conflict", "Conflict"},
	500: {"/internal-server-error", "Internal Server Error"},
}

// ToProblemDetail converts an [ErrorResponse] into an RFC 7807 [ProblemDetail].
//
//   - baseURL is the URI prefix for problem types, e.g. "https://errors.myapp.com".
//     The full type URI is baseURL + slug (e.g. "https://errors.myapp.com/not-found").
//   - instance is the request path or URI (e.g. r.URL.Path). Pass "" to omit.
//
// Status → slug mapping:
//
//	400 → /bad-request          401 → /unauthorized
//	403 → /forbidden             404 → /not-found
//	409 → /conflict              500 → /internal-server-error
func (r ErrorResponse) ToProblemDetail(baseURL, instance string) ProblemDetail {
	meta, ok := knownStatuses[r.Status]
	if !ok {
		meta = statusMeta{slug: "/error", title: "Error"}
	}

	pd := ProblemDetail{
		Type:   baseURL + meta.slug,
		Title:  meta.title,
		Status: r.Status,
		Detail: r.Message,
		Errors: r.Errors,
	}
	if instance != "" {
		pd.Instance = instance
	}
	return pd
}
