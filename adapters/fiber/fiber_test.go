package fiberadapter_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	fiberadapter "github.com/Lucas-Lopes-II/govalidator/adapters/fiber"
	"github.com/Lucas-Lopes-II/govalidator/domainerr"
	"github.com/gofiber/fiber/v2"
)

type errBody struct {
	Status      int      `json:"status"`
	Message     string   `json:"message"`
	Errors      []string `json:"errors"`
	Displayable bool     `json:"displayable"`
}

func decode(t *testing.T, r io.Reader) errBody {
	t.Helper()
	var out errBody
	if err := json.NewDecoder(r).Decode(&out); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	return out
}

func newApp() *fiber.App {
	return fiber.New(fiber.Config{ErrorHandler: fiberadapter.ErrorHandler})
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
			app := newApp()
			app.Get("/", func(c *fiber.Ctx) error { return tc.err })

			resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("want %d, got %d", tc.wantStatus, resp.StatusCode)
			}
			body := decode(t, resp.Body)
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
	app := newApp()
	app.Get("/", func(c *fiber.Ctx) error {
		return domainerr.NewComposite([]string{"name required", "email invalid"})
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", resp.StatusCode)
	}
	body := decode(t, resp.Body)
	if len(body.Errors) != 2 {
		t.Errorf("want 2 errors, got %d: %v", len(body.Errors), body.Errors)
	}
}

// ─── *fiber.Error mapping ─────────────────────────────────────────────────────

func TestErrorHandler_FiberError_StatusMapping(t *testing.T) {
	tests := []struct {
		name       string
		fiberCode  int
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
			app := newApp()
			app.Get("/", func(c *fiber.Ctx) error {
				return fiber.NewError(tc.fiberCode, http.StatusText(tc.fiberCode))
			})

			resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("fiberCode %d: want %d, got %d", tc.fiberCode, tc.wantStatus, resp.StatusCode)
			}
			body := decode(t, resp.Body)
			if body.Status != tc.wantStatus {
				t.Errorf("body.status: want %d, got %d", tc.wantStatus, body.Status)
			}
		})
	}
}

// ─── Unknown error ────────────────────────────────────────────────────────────

func TestErrorHandler_UnknownError_Returns500(t *testing.T) {
	app := newApp()
	app.Get("/", func(c *fiber.Ctx) error {
		return http.ErrNoCookie // plain stdlib error, not a DomainError
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", resp.StatusCode)
	}
	body := decode(t, resp.Body)
	if body.Status != 500 {
		t.Errorf("body.status: want 500, got %d", body.Status)
	}
}

// ─── Handler success path ─────────────────────────────────────────────────────

func TestErrorHandler_NotCalledOnSuccess(t *testing.T) {
	app := newApp()
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(http.StatusCreated).JSON(fiber.Map{"ok": true})
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("want 201, got %d", resp.StatusCode)
	}
}
