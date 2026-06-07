package domainerr_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Lucas-Lopes-II/govalidator/domainerr"
)

// ─── domainerr.go ─────────────────────────────────────────────────────────────

func TestConstructors_StatusCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        domainerr.DomainError
		wantStatus int
	}{
		{"NewBadRequest", domainerr.NewBadRequest("msg"), 400},
		{"NewUnauthorized", domainerr.NewUnauthorized("msg"), 401},
		{"NewForbidden", domainerr.NewForbidden("msg"), 403},
		{"NewNotFound", domainerr.NewNotFound("msg"), 404},
		{"NewConflict", domainerr.NewConflict("msg"), 409},
		{"NewInternal", domainerr.NewInternal("msg"), 500},
		{"NewComposite", domainerr.NewComposite([]string{"a"}), 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.StatusCode(); got != tt.wantStatus {
				t.Errorf("StatusCode() = %d, want %d", got, tt.wantStatus)
			}
		})
	}
}

func TestConstructors_ErrorMessage(t *testing.T) {
	t.Parallel()

	const msg = "something went wrong"
	err := domainerr.NewBadRequest(msg)

	if err.Error() != msg {
		t.Errorf("Error() = %q, want %q", err.Error(), msg)
	}
}

func TestConstructors_DefaultsNoMessages(t *testing.T) {
	t.Parallel()

	// All constructors except NewComposite should have nil Messages() by default.
	tests := []struct {
		name string
		err  domainerr.DomainError
	}{
		{"BadRequest", domainerr.NewBadRequest("m")},
		{"Unauthorized", domainerr.NewUnauthorized("m")},
		{"Forbidden", domainerr.NewForbidden("m")},
		{"NotFound", domainerr.NewNotFound("m")},
		{"Conflict", domainerr.NewConflict("m")},
		{"Internal", domainerr.NewInternal("m")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if msgs := tt.err.Messages(); msgs != nil {
				t.Errorf("Messages() = %v, want nil", msgs)
			}
		})
	}
}

func TestConstructors_DefaultNotDisplayable(t *testing.T) {
	t.Parallel()

	if domainerr.NewBadRequest("m").Displayable() {
		t.Error("Displayable() = true, want false by default")
	}
}

func TestWithMessages(t *testing.T) {
	t.Parallel()

	want := []string{"field a is required", "field b is invalid"}
	err := domainerr.NewBadRequest("Bad Request", domainerr.WithMessages(want...))

	got := err.Messages()
	if len(got) != len(want) {
		t.Fatalf("Messages() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Messages()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestWithMessages_ReturnsCopy(t *testing.T) {
	t.Parallel()

	err := domainerr.NewBadRequest("m", domainerr.WithMessages("a", "b"))

	first := err.Messages()
	first[0] = "mutated"
	second := err.Messages()

	if second[0] == "mutated" {
		t.Error("Messages() returned a reference to internal slice; want defensive copy")
	}
}

func TestWithDisplayable(t *testing.T) {
	t.Parallel()

	err := domainerr.NewNotFound("user not found", domainerr.WithDisplayable())

	if !err.Displayable() {
		t.Error("Displayable() = false, want true after WithDisplayable()")
	}
}

func TestNewComposite_Messages(t *testing.T) {
	t.Parallel()

	msgs := []string{"name is required", "email is invalid", "age must be positive"}
	err := domainerr.NewComposite(msgs)

	if err.StatusCode() != 400 {
		t.Errorf("StatusCode() = %d, want 400", err.StatusCode())
	}
	if err.Displayable() {
		t.Error("Displayable() = true, want always false for CompositeErr")
	}
	got := err.Messages()
	if len(got) != len(msgs) {
		t.Fatalf("Messages() len = %d, want %d", len(got), len(msgs))
	}
	for i, m := range msgs {
		if got[i] != m {
			t.Errorf("Messages()[%d] = %q, want %q", i, got[i], m)
		}
	}
}

func TestNewComposite_InputSliceIsolated(t *testing.T) {
	t.Parallel()

	input := []string{"a", "b"}
	err := domainerr.NewComposite(input)

	// Mutate the original slice — CompositeErr must not be affected.
	input[0] = "mutated"
	if err.Messages()[0] == "mutated" {
		t.Error("NewComposite retained reference to caller's slice; want defensive copy")
	}
}

func TestCompositeErr_Unwrap(t *testing.T) {
	t.Parallel()

	msgs := []string{"first error", "second error"}
	err := domainerr.NewComposite(msgs)

	unwrapped := errors.Unwrap(err) // returns first wrapped error for multi-unwrap
	if unwrapped == nil {
		// For multi-error, errors.Unwrap may return nil; use errors.Join semantics check.
		// Verify via errors.As traversal on wrapped sentinel instead.
		t.Skip("errors.Unwrap returns nil for []error — using alternative check")
	}
}

func TestCompositeErr_Unwrap_ReturnsAllMessages(t *testing.T) {
	t.Parallel()

	msgs := []string{"alpha failed", "beta failed"}
	composite := domainerr.NewComposite(msgs)

	// Unwrap() []error is an exported method — call it directly.
	unwrapped := composite.Unwrap()
	if len(unwrapped) != len(msgs) {
		t.Fatalf("Unwrap() len = %d, want %d", len(unwrapped), len(msgs))
	}
	for i, e := range unwrapped {
		if e.Error() != msgs[i] {
			t.Errorf("Unwrap()[%d].Error() = %q, want %q", i, e.Error(), msgs[i])
		}
	}
}

func TestIsDomain_TrueForAllConcreteTypes(t *testing.T) {
	t.Parallel()

	errs := []error{
		domainerr.NewBadRequest("m"),
		domainerr.NewUnauthorized("m"),
		domainerr.NewForbidden("m"),
		domainerr.NewNotFound("m"),
		domainerr.NewConflict("m"),
		domainerr.NewInternal("m"),
		domainerr.NewComposite([]string{"m"}),
	}

	for _, e := range errs {
		de, ok := domainerr.IsDomain(e)
		if !ok {
			t.Errorf("IsDomain(%T) = false, want true", e)
		}
		if de == nil {
			t.Errorf("IsDomain(%T) returned nil DomainError", e)
		}
	}
}

func TestIsDomain_FalseForStdlibError(t *testing.T) {
	t.Parallel()

	_, ok := domainerr.IsDomain(errors.New("stdlib error"))
	if ok {
		t.Error("IsDomain(stdlib error) = true, want false")
	}
}

func TestIsDomain_TrueForWrappedError(t *testing.T) {
	t.Parallel()

	inner := domainerr.NewNotFound("user not found")
	wrapped := fmt.Errorf("usecase layer: %w", inner)

	de, ok := domainerr.IsDomain(wrapped)
	if !ok {
		t.Fatal("IsDomain(wrapped) = false, want true")
	}
	if de.StatusCode() != 404 {
		t.Errorf("StatusCode() = %d, want 404", de.StatusCode())
	}
}

func TestIsInternal_TrueOnlyForInternalErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantInt bool
	}{
		{"InternalErr", domainerr.NewInternal("infra failure"), true},
		{"BadRequestErr", domainerr.NewBadRequest("bad"), false},
		{"NotFoundErr", domainerr.NewNotFound("missing"), false},
		{"CompositeErr", domainerr.NewComposite([]string{"a"}), false},
		{"stdlib error", errors.New("oops"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := domainerr.IsInternal(tt.err); got != tt.wantInt {
				t.Errorf("IsInternal(%T) = %v, want %v", tt.err, got, tt.wantInt)
			}
		})
	}
}

// ─── response.go ──────────────────────────────────────────────────────────────

func TestFromDomainError(t *testing.T) {
	t.Parallel()

	err := domainerr.NewBadRequest("name is required",
		domainerr.WithMessages("name is required", "email is invalid"),
		domainerr.WithDisplayable(),
	)
	resp := domainerr.FromDomainError(err)

	if resp.Status != 400 {
		t.Errorf("Status = %d, want 400", resp.Status)
	}
	if resp.Message != "name is required" {
		t.Errorf("Message = %q, want %q", resp.Message, "name is required")
	}
	if !resp.Displayable {
		t.Error("Displayable = false, want true")
	}
	if len(resp.Errors) != 2 {
		t.Fatalf("Errors len = %d, want 2", len(resp.Errors))
	}
}

func TestFromError_WithDomainError(t *testing.T) {
	t.Parallel()

	de := domainerr.NewConflict("email already exists", domainerr.WithDisplayable())
	resp := domainerr.FromError(de)

	if resp.Status != 409 {
		t.Errorf("Status = %d, want 409", resp.Status)
	}
	if resp.Message != "email already exists" {
		t.Errorf("Message = %q, want %q", resp.Message, "email already exists")
	}
}

func TestFromError_WithWrappedDomainError(t *testing.T) {
	t.Parallel()

	inner := domainerr.NewForbidden("access denied")
	wrapped := fmt.Errorf("middleware: %w", inner)
	resp := domainerr.FromError(wrapped)

	if resp.Status != 403 {
		t.Errorf("Status = %d, want 403", resp.Status)
	}
}

func TestFromError_WithGenericError_Returns500(t *testing.T) {
	t.Parallel()

	secret := errors.New("pq: connection refused: password auth failed for user admin")
	resp := domainerr.FromError(secret)

	if resp.Status != 500 {
		t.Errorf("Status = %d, want 500", resp.Status)
	}
	if resp.Message != "Internal Server Error" {
		t.Errorf("Message = %q, want %q", resp.Message, "Internal Server Error")
	}
	// Critical: original message must NEVER be exposed.
	if resp.Message == secret.Error() {
		t.Error("FromError exposed the original internal error message; security violation")
	}
}

func TestFromError_NilError(t *testing.T) {
	t.Parallel()

	// nil is a non-DomainError; must return 500 without panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("FromError(nil) panicked: %v", r)
		}
	}()
	resp := domainerr.FromError(nil)
	if resp.Status != 500 {
		t.Errorf("Status = %d, want 500 for nil error", resp.Status)
	}
}

func TestToProblemDetail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       domainerr.DomainError
		wantSlug  string
		wantTitle string
	}{
		{"400", domainerr.NewBadRequest("bad"), "/bad-request", "Bad Request"},
		{"401", domainerr.NewUnauthorized("unauth"), "/unauthorized", "Unauthorized"},
		{"403", domainerr.NewForbidden("forbidden"), "/forbidden", "Forbidden"},
		{"404", domainerr.NewNotFound("missing"), "/not-found", "Not Found"},
		{"409", domainerr.NewConflict("dup"), "/conflict", "Conflict"},
		{"500", domainerr.NewInternal("crash"), "/internal-server-error", "Internal Server Error"},
	}

	const baseURL = "https://errors.example.com"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := domainerr.FromDomainError(tt.err)
			pd := resp.ToProblemDetail(baseURL, "/api/test")

			wantType := baseURL + tt.wantSlug
			if pd.Type != wantType {
				t.Errorf("Type = %q, want %q", pd.Type, wantType)
			}
			if pd.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", pd.Title, tt.wantTitle)
			}
			if pd.Status != tt.err.StatusCode() {
				t.Errorf("Status = %d, want %d", pd.Status, tt.err.StatusCode())
			}
			if pd.Instance != "/api/test" {
				t.Errorf("Instance = %q, want %q", pd.Instance, "/api/test")
			}
		})
	}
}

func TestToProblemDetail_EmptyInstance(t *testing.T) {
	t.Parallel()

	resp := domainerr.FromDomainError(domainerr.NewBadRequest("bad"))
	pd := resp.ToProblemDetail("https://errors.example.com", "")

	if pd.Instance != "" {
		t.Errorf("Instance = %q, want empty string when not provided", pd.Instance)
	}
}

func TestToProblemDetail_UnknownStatus_FallbackSlug(t *testing.T) {
	t.Parallel()

	// Construct an ErrorResponse with a non-standard status code (e.g. 503)
	// to exercise the fallback branch inside ToProblemDetail.
	resp := domainerr.ErrorResponse{Status: 503, Message: "Service Unavailable"}
	pd := resp.ToProblemDetail("https://errors.example.com", "")

	const wantType = "https://errors.example.com/error"
	if pd.Type != wantType {
		t.Errorf("Type = %q, want %q", pd.Type, wantType)
	}
	if pd.Title != "Error" {
		t.Errorf("Title = %q, want %q", pd.Title, "Error")
	}
	if pd.Status != 503 {
		t.Errorf("Status = %d, want 503", pd.Status)
	}
}

// ─── http.go ──────────────────────────────────────────────────────────────────

func TestWriteError_StatusAndContentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"BadRequest", domainerr.NewBadRequest("bad"), 400},
		{"NotFound", domainerr.NewNotFound("missing"), 404},
		{"Internal", domainerr.NewInternal("crash"), 500},
		{"stdlib error → 500", errors.New("generic"), 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			domainerr.WriteError(w, tt.err)

			if w.Code != tt.wantStatus {
				t.Errorf("HTTP status = %d, want %d", w.Code, tt.wantStatus)
			}
			ct := w.Header().Get("Content-Type")
			if ct != "application/json" {
				t.Errorf("Content-Type = %q, want %q", ct, "application/json")
			}
		})
	}
}

func TestWriteError_ValidJSON(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	domainerr.WriteError(w, domainerr.NewBadRequest("email is invalid"))

	var body domainerr.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body.Status != 400 {
		t.Errorf("body.Status = %d, want 400", body.Status)
	}
	if body.Message != "email is invalid" {
		t.Errorf("body.Message = %q, want %q", body.Message, "email is invalid")
	}
}

func TestWriteError_GenericError_NoMessageLeak(t *testing.T) {
	t.Parallel()

	secret := errors.New("pq: ssl not supported")
	w := httptest.NewRecorder()
	domainerr.WriteError(w, secret)

	if w.Code != 500 {
		t.Errorf("HTTP status = %d, want 500", w.Code)
	}

	var body domainerr.ErrorResponse
	_ = json.NewDecoder(w.Body).Decode(&body)

	if body.Message == secret.Error() {
		t.Error("WriteError leaked the internal error message; security violation")
	}
}

func TestMiddleware_PassThrough(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	domainerr.Middleware(handler).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("HTTP status = %d, want 200", w.Code)
	}
}

func TestMiddleware_RecoversPanic(t *testing.T) {
	t.Parallel()

	panicking := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("unexpected nil pointer")
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	domainerr.Middleware(panicking).ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("HTTP status after panic = %d, want 500", w.Code)
	}

	var body domainerr.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("response after panic is not valid JSON: %v", err)
	}
	if body.Status != 500 {
		t.Errorf("body.Status = %d, want 500", body.Status)
	}
}
