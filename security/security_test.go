package security_test

import (
	"errors"
	"testing"

	"github.com/Lucas-Lopes-II/govalidator/domainerr"
	"github.com/Lucas-Lopes-II/govalidator/security"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func assertBadRequest(t *testing.T, err error, wantMsg string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var br *domainerr.BadRequestErr
	if !errors.As(err, &br) {
		t.Fatalf("expected *BadRequestErr, got %T: %v", err, err)
	}
	if got := br.Error(); got != wantMsg {
		t.Errorf("message: got %q, want %q", got, wantMsg)
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ─── TestNormalizeString ──────────────────────────────────────────────────────

func TestNormalizeString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string unchanged", "", ""},
		{"plain text unchanged", "Hello, World!", "Hello, World!"},
		{"trims leading spaces", "   hello", "hello"},
		{"trims trailing spaces", "hello   ", "hello"},
		{"trims both sides", "  hello  ", "hello"},
		{"invisible replaced then trimmed — all invisible", "\u200B\uFEFF", ""},
		{"NUL becomes space then trimmed", "\x00", ""},
		{"VT in middle becomes space", "a\x0Bb", "a b"},
		{"zero-width space in middle becomes space", "a\u200Bb", "a b"},
		{"BOM at start replaced and trimmed", "\uFEFFhello", "hello"},
		{"multiple invisibles collapsed via trim", "\u200B\u200Chello\u200D", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := security.NormalizeString(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ─── TestStripInvisibleChars ──────────────────────────────────────────────────

func TestStripInvisibleChars(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string unchanged", "", ""},
		{"plain ASCII unchanged", "hello", "hello"},
		{"space and tab preserved", "hello world\t!", "hello world\t!"},
		{"newline preserved", "line1\nline2", "line1\nline2"},
		{"carriage return preserved", "a\rb", "a\rb"},
		{"NUL removed", "hel\x00lo", "hello"},
		{"VT removed", "hel\x0Blo", "hello"},
		{"FF removed", "hel\x0Clo", "hello"},
		{"DEL removed", "hel\x7Flo", "hello"},
		{"zero-width space U+200B removed", "hello\u200Bworld", "helloworld"},
		{"BOM U+FEFF removed", "\uFEFFhello", "hello"},
		{"word joiner U+2060 removed", "hello\u2060world", "helloworld"},
		{"C1 control U+0085 removed", "hello", "hello"},
		{"interlinear annotation U+FFF9 removed", "a\uFFF9b", "ab"},
		{"multiple invisibles removed", "\x00\u200B\uFEFFhello\u2060\uFFFB", "hello"},
		{"only invisibles → empty string", "\x00\u200B\uFEFF", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := security.StripInvisibleChars(tt.input)
			if got != tt.want {
				t.Errorf("StripInvisibleChars(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ─── TestIsSafeString ─────────────────────────────────────────────────────────

func TestIsSafeString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Safe inputs
		{"empty string is safe", "", true},
		{"plain text is safe", "Hello, World!", true},
		{"special chars !@#$% are safe", "!@#$%^&*()_+-=[]", true},
		{"apostrophe is safe", "it's here", true},
		{"angle brackets in maths", "a > b", true}, // no closing >
		{"single < without >", "a < b", true},      // not a complete tag

		// HTML/XML tags
		{"script tag rejected", "<script>alert(1)</script>", false},
		{"img tag rejected", "<img src=x>", false},
		{"br tag rejected", "<br/>", false},
		{"comment-like tag rejected", "<b>bold</b>", false},

		// Event handlers
		{"onclick= rejected", "onclick=alert(1)", false},
		{"onload= rejected", "onload=evil()", false},
		{"onmouseover= rejected", "onmouseover=x", false},
		{"event handler with space rejected", "onload = evil()", false},

		// Script-scheme URIs
		{"javascript: rejected", "javascript:alert(1)", false},
		{"JAVASCRIPT: (case) rejected", "JAVASCRIPT:alert(1)", false},
		{"vbscript: rejected", "vbscript:msgbox(1)", false},
		{"data: rejected", "data:text/html,<h1>x</h1>", false},
		{"data: with space rejected", "data :image/png", false},

		// CSS expression
		{"expression() rejected", "expression(alert(1))", false},
		{"EXPRESSION() rejected", "EXPRESSION(document.cookie)", false},
		{"url() rejected", "url(javascript:alert(1))", false},
		{"url with space rejected", "url (evil)", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := security.IsSafeString(tt.input)
			if got != tt.want {
				t.Errorf("IsSafeString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ─── TestStripHTML ────────────────────────────────────────────────────────────

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string", "", ""},
		{"no tags unchanged", "Hello, World!", "Hello, World!"},
		{"single tag removed", "<b>bold</b>", "bold"},
		{"script tag removed", "<script>alert(1)</script>", "alert(1)"},
		{"img self-closing removed", "before<img src=x>after", "beforeafter"},
		{"multiple tags removed", "<p>Hello</p><br/>World", "HelloWorld"},
		{"entity not decoded", "a &amp; b", "a &amp; b"},
		{"nested tags removed", "<div><p>text</p></div>", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := security.StripHTML(tt.input)
			if got != tt.want {
				t.Errorf("StripHTML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ─── TestRequireUUID ─────────────────────────────────────────────────────────

func TestRequireUUID(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		paramName string
		wantErr   bool
		wantMsg   string
		wantValue string
	}{
		{
			name:      "valid v4 UUID returned trimmed",
			value:     "550e8400-e29b-41d4-a716-446655440000",
			paramName: "id",
			wantValue: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:      "valid UUID with surrounding spaces — trimmed",
			value:     "  550e8400-e29b-41d4-a716-446655440000  ",
			paramName: "id",
			wantValue: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:      "valid v1 UUID accepted",
			value:     "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			paramName: "user_id",
			wantValue: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		},
		{
			name:      "empty string — required error",
			value:     "",
			paramName: "id",
			wantErr:   true,
			wantMsg:   "id is required",
		},
		{
			name:      "whitespace-only — required error",
			value:     "   ",
			paramName: "id",
			wantErr:   true,
			wantMsg:   "id is required",
		},
		{
			name:      "invalid UUID — format error",
			value:     "not-a-uuid",
			paramName: "id",
			wantErr:   true,
			wantMsg:   "id must be a valid UUID",
		},
		{
			name:      "custom param name in error",
			value:     "",
			paramName: "category_id",
			wantErr:   true,
			wantMsg:   "category_id is required",
		},
		{
			name:      "partial UUID — format error",
			value:     "550e8400-e29b-41d4",
			paramName: "id",
			wantErr:   true,
			wantMsg:   "id must be a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := security.RequireUUID(tt.value, tt.paramName)
			if tt.wantErr {
				assertBadRequest(t, err, tt.wantMsg)
				if got != "" {
					t.Errorf("on error expected empty return, got %q", got)
				}
			} else {
				assertNoError(t, err)
				if got != tt.wantValue {
					t.Errorf("return value: got %q, want %q", got, tt.wantValue)
				}
			}
		})
	}
}

// ─── TestSafeSortField ────────────────────────────────────────────────────────

func TestSafeSortField(t *testing.T) {
	allowed := map[string]struct{}{
		"name":       {},
		"created_at": {},
		"email":      {},
	}

	tests := []struct {
		name         string
		field        string
		defaultField string
		want         string
	}{
		{"allowed field returned as-is", "name", "created_at", "name"},
		{"second allowed field", "email", "created_at", "email"},
		{"default returned for unknown field", "password", "created_at", "created_at"},
		{"default returned for empty field", "", "created_at", "created_at"},
		{"default returned for SQL injection attempt", "name; DROP TABLE users", "created_at", "created_at"},
		{"default returned for case mismatch", "Name", "created_at", "created_at"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := security.SafeSortField(tt.field, allowed, tt.defaultField)
			if got != tt.want {
				t.Errorf("SafeSortField(%q) = %q, want %q", tt.field, got, tt.want)
			}
		})
	}
}

// ─── TestSafePageSize ─────────────────────────────────────────────────────────

func TestSafePageSize(t *testing.T) {
	const defaultSize = 20
	const maxAllowed = 100

	tests := []struct {
		name string
		size int
		want int
	}{
		{"minimum valid size (1)", 1, 1},
		{"maximum valid size", 100, 100},
		{"middle valid size", 50, 50},
		{"default size itself is valid", 20, 20},
		{"zero → default", 0, defaultSize},
		{"negative → default", -1, defaultSize},
		{"above max → default", 101, defaultSize},
		{"very large → default", 99999, defaultSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := security.SafePageSize(tt.size, defaultSize, maxAllowed)
			if got != tt.want {
				t.Errorf("SafePageSize(%d, %d, %d) = %d, want %d",
					tt.size, defaultSize, maxAllowed, got, tt.want)
			}
		})
	}
}
