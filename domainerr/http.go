package domainerr

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// WriteError converts err to an [ErrorResponse], logs it appropriately, and
// writes the JSON body with the correct HTTP status code to w.
//
// Logging policy:
//   - status >= 500: [slog.Error] — unexpected infrastructure failure
//   - status  < 500: [slog.Warn]  — expected domain violation
//
// NEVER exposes raw error messages, stack traces, or internal state from
// non-[DomainError] errors. Unknown errors are always serialised as 500.
//
// Compatible with any [http.ResponseWriter], including Gin's c.Writer and
// Echo's c.Response().
func WriteError(w http.ResponseWriter, err error) {
	resp := FromError(err)

	if resp.Status >= 500 {
		slog.Error("internal server error", "err", err)
	} else {
		slog.Warn("domain error", "status", resp.Status, "message", resp.Message)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.Status)
	_ = json.NewEncoder(w).Encode(resp)
}

// Middleware wraps a [http.Handler] with panic recovery.
// Any panic is caught, logged via [slog.Error], and converted to a 500 response
// by calling [WriteError] with a generic [*InternalErr].
//
// Place this as the outermost middleware in your handler chain so that panics
// from all inner handlers are caught.
//
// For business-logic errors, call [WriteError] directly inside your handlers.
//
// Example:
//
//	mux := http.NewServeMux()
//	mux.Handle("GET /users", domainerr.Middleware(usersHandler))
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered by domainerr.Middleware", "panic", rec)
				WriteError(w, NewInternal("Internal Server Error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
