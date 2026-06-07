package ginadapter_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	ginadapter "github.com/Lucas-Lopes-II/govalidator/adapters/gin"
	"github.com/Lucas-Lopes-II/govalidator/domainerr"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

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

// ─── Middleware ───────────────────────────────────────────────────────────────

func TestMiddleware_PanicRecovery(t *testing.T) {
	r := gin.New()
	r.Use(ginadapter.Middleware())
	r.GET("/", func(c *gin.Context) { panic("boom") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
	body := decode(t, w.Body.Bytes())
	if body.Status != 500 {
		t.Errorf("body.status: want 500, got %d", body.Status)
	}
}

func TestMiddleware_ErrorAccumulated(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{"bad request", domainerr.NewBadRequest("invalid input"), 400, "invalid input"},
		{"not found", domainerr.NewNotFound("resource not found"), 404, "resource not found"},
		{"unauthorized", domainerr.NewUnauthorized("token expired"), 401, "token expired"},
		{"forbidden", domainerr.NewForbidden("access denied"), 403, "access denied"},
		{"conflict", domainerr.NewConflict("already exists"), 409, "already exists"},
		{"internal", domainerr.NewInternal("db failure"), 500, "db failure"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(ginadapter.Middleware())
			r.GET("/", func(c *gin.Context) {
				_ = c.Error(tc.err)
			})

			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

			if w.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d", tc.wantStatus, w.Code)
			}
			body := decode(t, w.Body.Bytes())
			if body.Status != tc.wantStatus {
				t.Errorf("body.status: want %d, got %d", tc.wantStatus, body.Status)
			}
			if body.Message != tc.wantMsg {
				t.Errorf("body.message: want %q, got %q", tc.wantMsg, body.Message)
			}
		})
	}
}

func TestMiddleware_NoErrors_HandlerWritesResponse(t *testing.T) {
	r := gin.New()
	r.Use(ginadapter.Middleware())
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d", w.Code)
	}
}

func TestMiddleware_HandlerAlreadyWritten_ErrorIgnored(t *testing.T) {
	// Handler writes 200 first, then accumulates an error.
	// Middleware must not overwrite the committed response.
	r := gin.New()
	r.Use(ginadapter.Middleware())
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		_ = c.Error(domainerr.NewBadRequest("should be ignored"))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

// ─── WriteError ───────────────────────────────────────────────────────────────

func TestWriteError_SerializesAndAborts(t *testing.T) {
	aborted := false

	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		ginadapter.WriteError(c, domainerr.NewUnauthorized("token expired"))
	}, func(c *gin.Context) {
		// Should never run — WriteError aborts the chain.
		aborted = true
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
	body := decode(t, w.Body.Bytes())
	if body.Status != 401 {
		t.Errorf("body.status: want 401, got %d", body.Status)
	}
	if body.Message != "token expired" {
		t.Errorf("body.message: want 'token expired', got %q", body.Message)
	}
	if aborted {
		t.Error("second handler ran — WriteError did not abort the chain")
	}
}

func TestWriteError_CompositeErr(t *testing.T) {
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		ginadapter.WriteError(c, domainerr.NewComposite([]string{"name required", "email invalid"}))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
	body := decode(t, w.Body.Bytes())
	if len(body.Errors) != 2 {
		t.Errorf("want 2 errors, got %d: %v", len(body.Errors), body.Errors)
	}
}

func TestWriteError_UnknownError_Returns500(t *testing.T) {
	r := gin.New()
	r.Use(ginadapter.Middleware())
	r.GET("/", func(c *gin.Context) {
		_ = c.Error(http.ErrNoCookie) // plain stdlib error, not a DomainError
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}
