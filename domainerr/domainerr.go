// Package domainerr provides typed domain errors, HTTP serialization helpers,
// and a net/http middleware for consistent error handling across services.
//
// # Error hierarchy
//
//	DomainError (interface)
//	├── *BadRequestErr   HTTP 400 — invalid client input
//	├── *UnauthorizedErr HTTP 401 — missing or invalid authentication
//	├── *ForbiddenErr    HTTP 403 — authenticated but not permitted
//	├── *NotFoundErr     HTTP 404 — resource not found
//	├── *ConflictErr     HTTP 409 — duplicate or state violation
//	├── *InternalErr     HTTP 500 — infrastructure / unexpected failures
//	└── *CompositeErr    HTTP 400 — multiple validation errors in one pass
//
// # Basic usage
//
//	err := domainerr.NewBadRequest("email is required", domainerr.WithDisplayable())
//	de, ok := domainerr.IsDomain(err)   // ok == true
//	domainerr.WriteError(w, err)        // writes {"status":400,...} JSON
//
// # Layered wrapping
//
//	wrapped := fmt.Errorf("usecase: %w", domainerr.NewNotFound("user not found"))
//	de, ok := domainerr.IsDomain(wrapped)  // ok == true — works through wrappers
package domainerr

import "errors"

// ─── Public interface ─────────────────────────────────────────────────────────

// DomainError is the base interface implemented by all typed domain errors.
// Use [IsDomain] to extract it from arbitrary wrapped errors.
type DomainError interface {
	error
	// StatusCode returns the associated HTTP status code (400, 401, 403, 404, 409, or 500).
	StatusCode() int
	// Messages returns a copy of the supplementary error messages (e.g. per-field
	// validation failures). Returns nil when no extra messages are set.
	Messages() []string
	// Displayable reports whether Error() is safe to show directly to end users.
	// Defaults to false; opt-in via [WithDisplayable].
	Displayable() bool
}

// ─── Internal base ────────────────────────────────────────────────────────────

// baseErr is the shared implementation for all domain errors.
// It is intentionally unexported — the public contract is [DomainError].
type baseErr struct {
	status  int
	message string
	msgs    []string
	display bool
}

func (e *baseErr) Error() string { return e.message }

// StatusCode implements [DomainError].
func (e *baseErr) StatusCode() int { return e.status }

// Messages implements [DomainError].
// Returns a copy of the supplementary messages slice to preserve immutability.
func (e *baseErr) Messages() []string {
	if len(e.msgs) == 0 {
		return nil
	}
	out := make([]string, len(e.msgs))
	copy(out, e.msgs)
	return out
}

// Displayable implements [DomainError].
func (e *baseErr) Displayable() bool { return e.display }

// ─── Concrete error types ─────────────────────────────────────────────────────

// BadRequestErr represents an HTTP 400 Bad Request error.
// Use for invalid client input, failed field validations, or malformed requests.
type BadRequestErr struct{ baseErr }

// UnauthorizedErr represents an HTTP 401 Unauthorized error.
// Use when authentication credentials are missing, expired, or invalid.
type UnauthorizedErr struct{ baseErr }

// ForbiddenErr represents an HTTP 403 Forbidden error.
// Use when the client is authenticated but lacks permission for the operation.
type ForbiddenErr struct{ baseErr }

// NotFoundErr represents an HTTP 404 Not Found error.
// Use when a requested resource or entity does not exist.
type NotFoundErr struct{ baseErr }

// ConflictErr represents an HTTP 409 Conflict error.
// Use for duplicate resources, optimistic-lock failures, or invalid state transitions.
type ConflictErr struct{ baseErr }

// InternalErr represents an HTTP 500 Internal Server Error.
// Use only for infrastructure failures or truly unexpected conditions.
// NEVER mark an InternalErr as [WithDisplayable] — it would leak internal details.
type InternalErr struct{ baseErr }

// CompositeErr collects multiple validation error messages from a single pass.
// Equivalent to ErrorCompositionException in the Java companion library.
// Always HTTP 400. Never displayable.
// Implements Unwrap() []error (Go 1.20+ multi-error) so errors.Is/As traverse it.
type CompositeErr struct{ baseErr }

// Unwrap implements the multi-error interface (Go 1.20+).
// Each supplementary message is wrapped in [errors.New] so that callers can
// inspect individual errors via [errors.Is] or [errors.As].
func (e *CompositeErr) Unwrap() []error {
	errs := make([]error, len(e.msgs))
	for i, msg := range e.msgs {
		errs[i] = errors.New(msg)
	}
	return errs
}

// ─── Constructors ─────────────────────────────────────────────────────────────

// NewBadRequest creates an HTTP 400 Bad Request error.
func NewBadRequest(message string, opts ...Option) *BadRequestErr {
	return &BadRequestErr{newBase(400, message, opts...)}
}

// NewUnauthorized creates an HTTP 401 Unauthorized error.
func NewUnauthorized(message string, opts ...Option) *UnauthorizedErr {
	return &UnauthorizedErr{newBase(401, message, opts...)}
}

// NewForbidden creates an HTTP 403 Forbidden error.
func NewForbidden(message string, opts ...Option) *ForbiddenErr {
	return &ForbiddenErr{newBase(403, message, opts...)}
}

// NewNotFound creates an HTTP 404 Not Found error.
func NewNotFound(message string, opts ...Option) *NotFoundErr {
	return &NotFoundErr{newBase(404, message, opts...)}
}

// NewConflict creates an HTTP 409 Conflict error.
func NewConflict(message string, opts ...Option) *ConflictErr {
	return &ConflictErr{newBase(409, message, opts...)}
}

// NewInternal creates an HTTP 500 Internal Server Error.
func NewInternal(message string, opts ...Option) *InternalErr {
	return &InternalErr{newBase(500, message, opts...)}
}

// NewComposite creates a CompositeErr that bundles multiple validation messages.
// Always HTTP 400 and never displayable; does not accept [Option] parameters.
// The messages slice is copied — the caller's slice is not retained.
func NewComposite(messages []string) *CompositeErr {
	msgs := make([]string, len(messages))
	copy(msgs, messages)
	return &CompositeErr{baseErr{
		status:  400,
		message: "Bad Request",
		msgs:    msgs,
		display: false,
	}}
}

// ─── Functional options ───────────────────────────────────────────────────────

// Option is a functional option that configures a domain error at construction time.
// Use the provided constructors ([WithMessages], [WithDisplayable]) to create Options.
type Option func(*baseErr)

// WithMessages appends supplementary messages to the error (e.g. per-field errors).
// The provided strings are copied into the error; the caller's values are not retained.
func WithMessages(msgs ...string) Option {
	return func(b *baseErr) {
		cp := make([]string, len(msgs))
		copy(cp, msgs)
		b.msgs = cp
	}
}

// WithDisplayable marks the error message as safe to expose to end users.
// Use only when the message is intentional, sanitised, and domain-appropriate.
// NEVER apply to [InternalErr].
func WithDisplayable() Option {
	return func(b *baseErr) { b.display = true }
}

// newBase is the unexported constructor shared by all typed error constructors.
func newBase(status int, message string, opts ...Option) baseErr {
	b := baseErr{status: status, message: message}
	for _, o := range opts {
		o(&b)
	}
	return b
}

// ─── Inspection helpers ───────────────────────────────────────────────────────

// IsDomain reports whether err (or any error it wraps) implements [DomainError].
// Returns the typed interface value and true on success; nil and false otherwise.
//
// Works correctly with wrapped errors created by fmt.Errorf("%w", ...).
//
//	de, ok := domainerr.IsDomain(err)
//	if ok {
//	    log.Printf("domain error %d: %s", de.StatusCode(), de.Error())
//	}
func IsDomain(err error) (DomainError, bool) {
	var de DomainError
	if errors.As(err, &de) {
		return de, true
	}
	return nil, false
}

// IsInternal reports whether err (or any error it wraps) is an [*InternalErr].
// Used by [ValidationComposite] to abort validation on infrastructure failures.
func IsInternal(err error) bool {
	var e *InternalErr
	return errors.As(err, &e)
}
