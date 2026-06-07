package echoadapter_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	echoadapter "github.com/Lucas-Lopes-II/govalidator/adapters/echo"
	"github.com/Lucas-Lopes-II/govalidator/domainerr"
	"github.com/labstack/echo/v4"
)

type errBody struct {
	Status      int      `json:"status"`
	Message     string   `json:"message"`
	Errors      []string `json:"errors"`
	Displayable bool     `json:"displayable"`
}

func decode(t *testing.T, b []byte) errBody {
	t.Helper()
	var out errBody
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	return out
}

func newCtx() (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

// ─── DomainError passthrough ──────────────────────────────────────────────────

func TestErrorHandler_DomainErrors_Passthrough(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{"bad request", domainerr.NewBadRequest("invalid input"), 400, "invalid input"},
		{"unauthorized", domainerr.NewUnauthorized("token expired"), 401, "token expired"},
		{"forbidden", domainerr.NewForbidden("access denied"), 403, "access denied"},
		{"not found", domainerr.NewNotFound("item not found"), 404, "item not found"},
		{"conflict", domainerr.NewConflict("already exists"), 409, "already exists"},
		{"internal", domainerr.NewInternal("db failure"), 500, "db failure"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, rec := newCtx()
			echoadapter.ErrorHandler(tc.err, c)

			if rec.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d", tc.wantStatus, rec.Code)
			}
			body := decode(t, rec.Body.Bytes())
			if body.Status != tc.wantStatus {
				t.Errorf("body.status: want %d, got %d", tc.wantStatus, body.Status)
			}
			if body.Message != tc.wantMsg {
				t.Errorf("body.message: want %q, got %q", tc.wantMsg, body.Message)
			}
		})
	}
}

func TestErrorHandler_CompositeErr(t *testing.T) {
	c, rec := newCtx()
	echoadapter.ErrorHandler(domainerr.NewComposite([]string{"name required", "email invalid"}), c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
	body := decode(t, rec.Body.Bytes())
	if len(body.Errors) != 2 {
		t.Errorf("want 2 errors, got %d: %v", len(body.Errors), body.Errors)
	}
}

// ─── *echo.HTTPError mapping ──────────────────────────────────────────────────

func TestErrorHandler_EchoHTTPError_StatusMapping(t *testing.T) {
	tests := []struct {
		name       string
		echoCode   int
		wantStatus int
	}{
		{"400", 400, 400},
		{"401", 401, 401},
		{"403", 403, 403},
		{"404", 404, 404},
		{"409", 409, 409},
		{"422", 422, 400}, // unmapped 4xx → BadRequest
		{"500", 500, 500},
		{"503", 503, 500}, // unmapped 5xx → Internal
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, rec := newCtx()
			he := echo.NewHTTPError(tc.echoCode, http.StatusText(tc.echoCode))
			echoadapter.ErrorHandler(he, c)

			if rec.Code != tc.wantStatus {
				t.Fatalf("echoCode %d: want status %d, got %d", tc.echoCode, tc.wantStatus, rec.Code)
			}
			body := decode(t, rec.Body.Bytes())
			if body.Status != tc.wantStatus {
				t.Errorf("body.status: want %d, got %d", tc.wantStatus, body.Status)
			}
		})
	}
}

// ─── Unknown error ────────────────────────────────────────────────────────────

func TestErrorHandler_UnknownError_Returns500(t *testing.T) {
	c, rec := newCtx()
	echoadapter.ErrorHandler(http.ErrNoCookie, c)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", rec.Code)
	}
	body := decode(t, rec.Body.Bytes())
	if body.Status != 500 {
		t.Errorf("body.status: want 500, got %d", body.Status)
	}
}

// ─── Committed response guard ─────────────────────────────────────────────────

func TestErrorHandler_CommittedResponse_IsNoop(t *testing.T) {
	c, rec := newCtx()
	// Force the response to be committed (WriteHeader marks it).
	rec.WriteHeader(http.StatusOK)
	c.Response().Committed = true

	echoadapter.ErrorHandler(domainerr.NewBadRequest("should not appear"), c)

	// Body must remain empty — no second write happened.
	if rec.Body.Len() != 0 {
		t.Errorf("expected empty body after committed response, got %q", rec.Body.String())
	}
}
