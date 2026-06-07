package rules_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Lucas-Lopes-II/govalidator/domainerr"
	"github.com/Lucas-Lopes-II/govalidator/rules"
	"github.com/Lucas-Lopes-II/govalidator/validation"
)

// ─── test helpers ─────────────────────────────────────────────────────────────

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

// ─── string input fixture ─────────────────────────────────────────────────────

type strInput struct{ Value string }

func strExtract(i strInput) string { return i.Value }

// ─── TestRequired ─────────────────────────────────────────────────────────────

func TestRequired(t *testing.T) {
	rule := rules.Required(strExtract, "name", "name is required")

	tests := []struct {
		name    string
		input   strInput
		wantErr bool
	}{
		{"passes when non-empty", strInput{"Lucas"}, false},
		{"passes when has surrounding whitespace", strInput{"  x  "}, false},
		{"fails when empty string", strInput{""}, true},
		{"fails when only spaces", strInput{"   "}, true},
		{"fails when only tab", strInput{"\t"}, true},
		{"fails when only newline", strInput{"\n"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.input)
			if tt.wantErr {
				assertBadRequest(t, err, "name is required")
			} else {
				assertNoError(t, err)
			}
		})
	}
}

// ─── TestMinLength ────────────────────────────────────────────────────────────

func TestMinLength(t *testing.T) {
	rule := rules.MinLength(strExtract, 3, "name", "name must have at least 3 characters")

	tests := []struct {
		name    string
		input   strInput
		wantErr bool
	}{
		{"passes when exactly min", strInput{"abc"}, false},
		{"passes when above min", strInput{"abcd"}, false},
		{"passes with UTF-8 chars", strInput{"café"}, false}, // 4 runes
		{"fails when below min", strInput{"ab"}, true},
		{"fails when empty", strInput{""}, true},
		{"trims before counting", strInput{"  a  "}, true}, // trimmed = "a" (1 rune)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.input)
			if tt.wantErr {
				assertBadRequest(t, err, "name must have at least 3 characters")
			} else {
				assertNoError(t, err)
			}
		})
	}
}

// ─── TestMaxLength ────────────────────────────────────────────────────────────

func TestMaxLength(t *testing.T) {
	rule := rules.MaxLength(strExtract, 5, "name", "name must have at most 5 characters")

	tests := []struct {
		name    string
		input   strInput
		wantErr bool
	}{
		{"passes when below max", strInput{"abc"}, false},
		{"passes when exactly max", strInput{"abcde"}, false},
		{"passes when empty", strInput{""}, false},
		{"passes with UTF-8 at limit", strInput{"café!"}, false}, // 5 runes
		{"fails when above max", strInput{"abcdef"}, true},
		{"fails with UTF-8 above max", strInput{"café!!"}, true},
		{"trims before counting", strInput{"  abc  "}, false}, // trimmed = "abc" (3)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.input)
			if tt.wantErr {
				assertBadRequest(t, err, "name must have at most 5 characters")
			} else {
				assertNoError(t, err)
			}
		})
	}
}

// ─── TestEmail ────────────────────────────────────────────────────────────────

func TestEmail(t *testing.T) {
	rule := rules.Email(strExtract, "email", "email must be a valid email")

	tests := []struct {
		name    string
		input   strInput
		wantErr bool
	}{
		{"passes simple email", strInput{"user@example.com"}, false},
		{"passes with subdomain", strInput{"user@mail.example.co.uk"}, false},
		{"passes with tag", strInput{"user+tag@example.com"}, false},
		{"fails when empty", strInput{""}, true},
		{"fails without @", strInput{"userexample.com"}, true},
		{"fails without domain", strInput{"user@"}, true},
		{"fails without tld", strInput{"user@domain"}, true},
		{"fails with space", strInput{"user @example.com"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.input)
			if tt.wantErr {
				assertBadRequest(t, err, "email must be a valid email")
			} else {
				assertNoError(t, err)
			}
		})
	}
}

// ─── TestUUID ─────────────────────────────────────────────────────────────────

func TestUUID(t *testing.T) {
	rule := rules.UUID(strExtract, "id", "id must be a valid UUID")

	tests := []struct {
		name    string
		input   strInput
		wantErr bool
	}{
		{"passes v4 UUID", strInput{"550e8400-e29b-41d4-a716-446655440000"}, false},
		{"passes v1 UUID", strInput{"6ba7b810-9dad-11d1-80b4-00c04fd430c8"}, false},
		{"passes v7 UUID", strInput{"018d6db7-7f55-7bbd-9c7a-5f2cba3bb86e"}, false},
		{"fails when empty", strInput{""}, true},
		{"fails invalid format", strInput{"not-a-uuid"}, true},
		{"fails short UUID", strInput{"550e8400-e29b-41d4"}, true},
		{"fails with extra chars", strInput{"550e8400-e29b-41d4-a716-446655440000X"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.input)
			if tt.wantErr {
				assertBadRequest(t, err, "id must be a valid UUID")
			} else {
				assertNoError(t, err)
			}
		})
	}
}

// ─── TestISODate ──────────────────────────────────────────────────────────────

func TestISODate(t *testing.T) {
	rule := rules.ISODate(strExtract, "created_at", "created_at must be a valid ISO 8601 date")

	tests := []struct {
		name    string
		input   strInput
		wantErr bool
	}{
		{"passes RFC3339 UTC", strInput{"2024-01-25T10:00:00Z"}, false},
		{"passes RFC3339 with offset", strInput{"2024-01-25T10:00:00-03:00"}, false},
		{"passes RFC3339Nano", strInput{"2024-01-25T10:00:00.929Z"}, false},
		{"passes RFC3339Nano with offset", strInput{"2024-01-25T10:00:00.929-03:00"}, false},
		{"fails date-only string", strInput{"2024-01-25"}, true},
		{"fails empty string", strInput{""}, true},
		{"fails arbitrary string", strInput{"not-a-date"}, true},
		{"fails no timezone", strInput{"2024-01-25T10:00:00"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.input)
			if tt.wantErr {
				assertBadRequest(t, err, "created_at must be a valid ISO 8601 date")
			} else {
				assertNoError(t, err)
			}
		})
	}
}

// ─── TestSafeString ───────────────────────────────────────────────────────────

func TestSafeString(t *testing.T) {
	rule := rules.SafeString(strExtract, "bio", "bio contains unsafe content")

	tests := []struct {
		name    string
		input   strInput
		wantErr bool
	}{
		{"passes normal text", strInput{"Hello, World!"}, false},
		{"passes special chars", strInput{"!@#$%^&*()_+-=[]"}, false},
		{"passes empty string", strInput{""}, false},
		{"passes text with apostrophe", strInput{"it's a test"}, false},
		{"fails HTML tag", strInput{"<script>alert(1)</script>"}, true},
		{"fails self-closing tag", strInput{"<br/>"}, true},
		{"fails event handler", strInput{"onclick=alert(1)"}, true},
		{"fails event handler with space", strInput{"onload = evil()"}, true},
		{"fails javascript URI", strInput{"javascript:alert(1)"}, true},
		{"fails vbscript URI", strInput{"vbscript:msgbox(1)"}, true},
		{"fails data URI", strInput{"data:text/html,<script>x</script>"}, true},
		{"fails CSS expression", strInput{"expression(alert(1))"}, true},
		{"fails zero-width space U+200B", strInput{"hello\u200Bworld"}, true},
		{"fails BOM U+FEFF", strInput{"\uFEFFhello"}, true},
		{"fails NUL char", strInput{"hel\x00lo"}, true},
		{"fails C1 control char U+0085", strInput{"hello"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.input)
			if tt.wantErr {
				assertBadRequest(t, err, "bio contains unsafe content")
			} else {
				assertNoError(t, err)
			}
		})
	}
}

// ─── TestMinValue ─────────────────────────────────────────────────────────────

func TestMinValue(t *testing.T) {
	type intInput struct{ Age int }
	extract := func(i intInput) int { return i.Age }
	rule := rules.MinValue(extract, 18, "age", "age must be at least 18")

	tests := []struct {
		name    string
		input   intInput
		wantErr bool
	}{
		{"passes when exactly min", intInput{18}, false},
		{"passes when above min", intInput{100}, false},
		{"fails when below min", intInput{17}, false},
		{"fails when zero", intInput{0}, true},
		{"fails when negative", intInput{-1}, true},
	}
	// fix the "fails when below min" test case — it should be true
	tests[2].wantErr = true

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.input)
			if tt.wantErr {
				assertBadRequest(t, err, "age must be at least 18")
			} else {
				assertNoError(t, err)
			}
		})
	}
}

func TestMinValueFloat64(t *testing.T) {
	type priceInput struct{ Price float64 }
	extract := func(i priceInput) float64 { return i.Price }
	rule := rules.MinValue(extract, 0.01, "price", "price must be at least 0.01")

	tests := []struct {
		name    string
		input   priceInput
		wantErr bool
	}{
		{"passes at 0.01", priceInput{0.01}, false},
		{"passes above min", priceInput{99.99}, false},
		{"fails at 0", priceInput{0}, true},
		{"fails negative", priceInput{-1.5}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.input)
			if tt.wantErr {
				assertBadRequest(t, err, "price must be at least 0.01")
			} else {
				assertNoError(t, err)
			}
		})
	}
}

// ─── TestMaxValue ─────────────────────────────────────────────────────────────

func TestMaxValue(t *testing.T) {
	type quantInput struct{ Qty int }
	extract := func(i quantInput) int { return i.Qty }
	rule := rules.MaxValue(extract, 100, "qty", "qty must be at most 100")

	tests := []struct {
		name    string
		input   quantInput
		wantErr bool
	}{
		{"passes when exactly max", quantInput{100}, false},
		{"passes when below max", quantInput{50}, false},
		{"passes when zero", quantInput{0}, false},
		{"fails when above max", quantInput{101}, true},
		{"fails when very large", quantInput{99999}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.input)
			if tt.wantErr {
				assertBadRequest(t, err, "qty must be at most 100")
			} else {
				assertNoError(t, err)
			}
		})
	}
}

// ─── TestIsTrue ───────────────────────────────────────────────────────────────

func TestIsTrue(t *testing.T) {
	type boolInput struct{ Accepted bool }
	extract := func(i boolInput) bool { return i.Accepted }
	rule := rules.IsTrue(extract, "accepted", "accepted must be true")

	tests := []struct {
		name    string
		input   boolInput
		wantErr bool
	}{
		{"passes when true", boolInput{true}, false},
		{"fails when false", boolInput{false}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.input)
			if tt.wantErr {
				assertBadRequest(t, err, "accepted must be true")
			} else {
				assertNoError(t, err)
			}
		})
	}
}

// ─── TestIsFalse ──────────────────────────────────────────────────────────────

func TestIsFalse(t *testing.T) {
	type boolInput struct{ Locked bool }
	extract := func(i boolInput) bool { return i.Locked }
	rule := rules.IsFalse(extract, "locked", "locked must be false")

	tests := []struct {
		name    string
		input   boolInput
		wantErr bool
	}{
		{"passes when false", boolInput{false}, false},
		{"fails when true", boolInput{true}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.input)
			if tt.wantErr {
				assertBadRequest(t, err, "locked must be false")
			} else {
				assertNoError(t, err)
			}
		})
	}
}

// ─── TestOneOf ────────────────────────────────────────────────────────────────

func TestOneOf(t *testing.T) {
	allowed := []string{"active", "inactive", "pending"}
	rule := rules.OneOf(strExtract, allowed, "status", "invalid status")

	tests := []struct {
		name    string
		input   strInput
		wantErr bool
	}{
		{"passes first value", strInput{"active"}, false},
		{"passes middle value", strInput{"inactive"}, false},
		{"passes last value", strInput{"pending"}, false},
		{"fails unknown value", strInput{"deleted"}, true},
		{"fails empty string", strInput{""}, true},
		{"fails case-sensitive mismatch", strInput{"Active"}, true},
		{"fails partial match", strInput{"act"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.input)
			if tt.wantErr {
				assertBadRequest(t, err, "invalid status")
			} else {
				assertNoError(t, err)
			}
		})
	}
}

// TestOneOf_MutationSafety verifies that mutating the allowed slice after rule
// creation does not affect the rule (snapshot at construction time).
func TestOneOf_MutationSafety(t *testing.T) {
	allowed := []string{"a", "b"}
	rule := rules.OneOf(strExtract, allowed, "f", "bad value")
	allowed[0] = "x" // mutate source

	assertNoError(t, rule.Validate(strInput{"a"})) // original "a" still valid
}

// ─── TestStringFieldBuilder ───────────────────────────────────────────────────

type createUserInput struct {
	Name   string
	Email  string
	Age    int
	Active bool
	Status string
	Score  float64
}

func TestStringFieldBuilder_Required_IsFirst(t *testing.T) {
	// Required is called LAST but must still be the first rule executed.
	builtRules := rules.StringField("name", func(i createUserInput) string { return i.Name }).
		MinLength(2).
		MaxLength(50).
		Required(). // called last — but must execute first
		Build()

	if len(builtRules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(builtRules))
	}

	// For an empty name, Required should produce "name is required",
	// not "name must have at least 2 characters".
	composite := validation.NewComposite(builtRules...)
	err := composite.Validate(createUserInput{Name: ""})
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}

	var ce *domainerr.CompositeErr
	var br *domainerr.BadRequestErr

	// With empty name, both Required AND MinLength fail — expect CompositeErr.
	if errors.As(err, &ce) {
		msgs := ce.Messages()
		if msgs[0] != "name is required" {
			t.Errorf("first message must be Required's; got %q", msgs[0])
		}
	} else if errors.As(err, &br) {
		if br.Error() != "name is required" {
			t.Errorf("message: got %q, want %q", br.Error(), "name is required")
		}
	} else {
		t.Fatalf("unexpected error type %T: %v", err, err)
	}
}

func TestStringFieldBuilder_DefaultMessages(t *testing.T) {
	tests := []struct {
		name    string
		build   func() []validation.Validation[createUserInput]
		input   createUserInput
		wantMsg string
	}{
		{
			"Required default",
			func() []validation.Validation[createUserInput] {
				return rules.StringField("email", func(i createUserInput) string { return i.Email }).
					Required().Build()
			},
			createUserInput{Email: ""},
			"email is required",
		},
		{
			"Email default",
			func() []validation.Validation[createUserInput] {
				return rules.StringField("email", func(i createUserInput) string { return i.Email }).
					Email().Build()
			},
			createUserInput{Email: "bad"},
			"email must be a valid email",
		},
		{
			"MinLength default",
			func() []validation.Validation[createUserInput] {
				return rules.StringField("name", func(i createUserInput) string { return i.Name }).
					MinLength(3).Build()
			},
			createUserInput{Name: "ab"},
			"name must have at least 3 characters",
		},
		{
			"MaxLength default",
			func() []validation.Validation[createUserInput] {
				return rules.StringField("name", func(i createUserInput) string { return i.Name }).
					MaxLength(5).Build()
			},
			createUserInput{Name: "toolong"},
			"name must have at most 5 characters",
		},
		{
			"UUID default",
			func() []validation.Validation[createUserInput] {
				return rules.StringField("id", func(i createUserInput) string { return i.Name }).
					UUID().Build()
			},
			createUserInput{Name: "not-uuid"},
			"id must be a valid UUID",
		},
		{
			"ISODate default",
			func() []validation.Validation[createUserInput] {
				return rules.StringField("created_at", func(i createUserInput) string { return i.Name }).
					ISODate().Build()
			},
			createUserInput{Name: "bad-date"},
			"created_at must be a valid ISO 8601 date",
		},
		{
			"OneOf default",
			func() []validation.Validation[createUserInput] {
				return rules.StringField("status", func(i createUserInput) string { return i.Status }).
					OneOf([]string{"a", "b"}).Build()
			},
			createUserInput{Status: "x"},
			"status must be one of: [a, b]",
		},
		{
			"SafeString default",
			func() []validation.Validation[createUserInput] {
				return rules.StringField("name", func(i createUserInput) string { return i.Name }).
					SafeString().Build()
			},
			createUserInput{Name: "<script>"},
			"name contains unsafe content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builtRules := tt.build()
			composite := validation.NewComposite(builtRules...)
			err := composite.Validate(tt.input)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			var br *domainerr.BadRequestErr
			var ce *domainerr.CompositeErr
			switch {
			case errors.As(err, &ce):
				found := false
				for _, m := range ce.Messages() {
					if m == tt.wantMsg {
						found = true
					}
				}
				if !found {
					t.Errorf("wanted message %q in composite %v", tt.wantMsg, ce.Messages())
				}
			case errors.As(err, &br):
				if br.Error() != tt.wantMsg {
					t.Errorf("message: got %q, want %q", br.Error(), tt.wantMsg)
				}
			default:
				t.Fatalf("unexpected error type %T", err)
			}
		})
	}
}

func TestStringFieldBuilder_CustomMessage(t *testing.T) {
	builtRules := rules.StringField("email", func(i createUserInput) string { return i.Email }).
		Required("e-mail obrigatório").
		Build()

	composite := validation.NewComposite(builtRules...)
	err := composite.Validate(createUserInput{Email: ""})
	assertBadRequest(t, err, "e-mail obrigatório")
}

func TestStringFieldBuilder_ValidInput(t *testing.T) {
	builtRules := rules.StringField("email", func(i createUserInput) string { return i.Email }).
		Required().
		Email().
		MaxLength(254).
		Build()

	composite := validation.NewComposite(builtRules...)
	assertNoError(t, composite.Validate(createUserInput{Email: "user@example.com"}))
}

func TestStringFieldBuilder_Build_ReturnsCopy(t *testing.T) {
	b := rules.StringField("x", func(i createUserInput) string { return i.Name }).Required()
	r1 := b.Build()
	r2 := b.Build()
	if len(r1) != len(r2) {
		t.Fatalf("Build() must return consistent slices; got %d and %d", len(r1), len(r2))
	}
}

// ─── TestIntFieldBuilder ──────────────────────────────────────────────────────

func TestIntFieldBuilder(t *testing.T) {
	extract := func(i createUserInput) int { return i.Age }

	t.Run("passes when in range", func(t *testing.T) {
		builtRules := rules.IntField("age", extract).Min(18).Max(120).Build()
		composite := validation.NewComposite(builtRules...)
		assertNoError(t, composite.Validate(createUserInput{Age: 25}))
	})

	t.Run("fails min with default message", func(t *testing.T) {
		builtRules := rules.IntField("age", extract).Min(18).Build()
		composite := validation.NewComposite(builtRules...)
		assertBadRequest(t, composite.Validate(createUserInput{Age: 17}),
			"age must be at least 18")
	})

	t.Run("fails max with default message", func(t *testing.T) {
		builtRules := rules.IntField("age", extract).Max(120).Build()
		composite := validation.NewComposite(builtRules...)
		assertBadRequest(t, composite.Validate(createUserInput{Age: 121}),
			"age must be at most 120")
	})

	t.Run("custom min message", func(t *testing.T) {
		builtRules := rules.IntField("age", extract).Min(18, "must be adult").Build()
		composite := validation.NewComposite(builtRules...)
		assertBadRequest(t, composite.Validate(createUserInput{Age: 0}), "must be adult")
	})

	t.Run("Build returns copy", func(t *testing.T) {
		b := rules.IntField("age", extract).Min(0)
		if len(b.Build()) != len(b.Build()) {
			t.Error("Build must be idempotent")
		}
	})
}

// ─── TestFloat64FieldBuilder ──────────────────────────────────────────────────

func TestFloat64FieldBuilder(t *testing.T) {
	extract := func(i createUserInput) float64 { return i.Score }

	t.Run("passes when in range", func(t *testing.T) {
		builtRules := rules.Float64Field("score", extract).Min(0).Max(10).Build()
		composite := validation.NewComposite(builtRules...)
		assertNoError(t, composite.Validate(createUserInput{Score: 7.5}))
	})

	t.Run("fails min with default message", func(t *testing.T) {
		builtRules := rules.Float64Field("score", extract).Min(1.5).Build()
		composite := validation.NewComposite(builtRules...)
		assertBadRequest(t, composite.Validate(createUserInput{Score: 1.0}),
			fmt.Sprintf("score must be at least %g", 1.5))
	})

	t.Run("fails max with default message", func(t *testing.T) {
		builtRules := rules.Float64Field("score", extract).Max(10.0).Build()
		composite := validation.NewComposite(builtRules...)
		assertBadRequest(t, composite.Validate(createUserInput{Score: 10.1}),
			fmt.Sprintf("score must be at most %g", 10.0))
	})

	t.Run("custom max message", func(t *testing.T) {
		builtRules := rules.Float64Field("score", extract).Max(10, "score too high").Build()
		composite := validation.NewComposite(builtRules...)
		assertBadRequest(t, composite.Validate(createUserInput{Score: 99}), "score too high")
	})
}

// ─── TestBoolFieldBuilder ─────────────────────────────────────────────────────

func TestBoolFieldBuilder(t *testing.T) {
	extractActive := func(i createUserInput) bool { return i.Active }

	t.Run("IsTrue passes when true", func(t *testing.T) {
		builtRules := rules.BoolField("active", extractActive).IsTrue().Build()
		composite := validation.NewComposite(builtRules...)
		assertNoError(t, composite.Validate(createUserInput{Active: true}))
	})

	t.Run("IsTrue fails when false — default message", func(t *testing.T) {
		builtRules := rules.BoolField("active", extractActive).IsTrue().Build()
		composite := validation.NewComposite(builtRules...)
		assertBadRequest(t, composite.Validate(createUserInput{Active: false}),
			"active must be true")
	})

	t.Run("IsFalse passes when false", func(t *testing.T) {
		builtRules := rules.BoolField("active", extractActive).IsFalse().Build()
		composite := validation.NewComposite(builtRules...)
		assertNoError(t, composite.Validate(createUserInput{Active: false}))
	})

	t.Run("IsFalse fails when true — default message", func(t *testing.T) {
		builtRules := rules.BoolField("active", extractActive).IsFalse().Build()
		composite := validation.NewComposite(builtRules...)
		assertBadRequest(t, composite.Validate(createUserInput{Active: true}),
			"active must be false")
	})

	t.Run("IsTrue custom message", func(t *testing.T) {
		builtRules := rules.BoolField("terms", extractActive).IsTrue("you must accept terms").Build()
		composite := validation.NewComposite(builtRules...)
		assertBadRequest(t, composite.Validate(createUserInput{Active: false}),
			"you must accept terms")
	})

	t.Run("Build returns copy", func(t *testing.T) {
		b := rules.BoolField("active", extractActive).IsTrue()
		if len(b.Build()) != len(b.Build()) {
			t.Error("Build must be idempotent")
		}
	})
}

// ─── TestCompositeIntegration ─────────────────────────────────────────────────

// TestCompositeIntegration verifies that StringFieldBuilder rules integrate
// correctly with ValidationComposite to collect multiple field errors at once.
func TestCompositeIntegration(t *testing.T) {
	nameRules := rules.StringField("name", func(i createUserInput) string { return i.Name }).
		Required().
		MinLength(2).
		Build()

	emailRules := rules.StringField("email", func(i createUserInput) string { return i.Email }).
		Required().
		Email().
		Build()

	composite := validation.NewComposite[createUserInput]()
	composite.Add(nameRules...).Add(emailRules...)

	t.Run("all fields valid — no error", func(t *testing.T) {
		assertNoError(t, composite.Validate(createUserInput{
			Name:  "Lucas",
			Email: "lucas@example.com",
		}))
	})

	t.Run("all fields empty — composite error with all messages", func(t *testing.T) {
		err := composite.Validate(createUserInput{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var ce *domainerr.CompositeErr
		if !errors.As(err, &ce) {
			t.Fatalf("expected *CompositeErr, got %T: %v", err, err)
		}
		msgs := ce.Messages()
		if len(msgs) < 2 {
			t.Errorf("expected at least 2 messages, got %d: %v", len(msgs), msgs)
		}
	})
}
