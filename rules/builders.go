package rules

import (
	"fmt"
	"strings"

	"github.com/Lucas-Lopes-II/govalidator/validation"
)

// firstOrDefault returns the first element of msgs if it is non-empty,
// otherwise it returns def. Used by all builder methods that accept an
// optional custom message.
func firstOrDefault(msgs []string, def string) string {
	if len(msgs) > 0 && msgs[0] != "" {
		return msgs[0]
	}
	return def
}

// ─── StringFieldBuilder ───────────────────────────────────────────────────────

// StringFieldBuilder constructs a chain of [validation.Validation] rules for a
// single string field of T. Call [StringFieldBuilder.Build] to retrieve the
// slice of rules for use with [validation.NewComposite] or [validation.ValidationComposite.Add].
//
// Ordering invariant: [StringFieldBuilder.Required] is always the first rule in
// the built slice regardless of when it is called in the chain. This prevents
// MinLength, Email, and similar rules from running against an empty field.
//
// Example:
//
//	rules.StringField("email", func(u CreateUserInput) string { return u.Email }).
//	    Required().
//	    Email().
//	    MaxLength(254).
//	    Build()
type StringFieldBuilder[T any] struct {
	field       string
	extract     func(T) string
	requiredMsg string // non-empty if Required was called
	rules       []validation.Validation[T]
}

// StringField creates a new [StringFieldBuilder] for the given field name and extractor.
func StringField[T any](field string, extract func(T) string) *StringFieldBuilder[T] {
	return &StringFieldBuilder[T]{field: field, extract: extract}
}

// Required marks the field as mandatory (non-empty after trim).
// Inserted as the first rule in [StringFieldBuilder.Build] regardless of call order.
func (b *StringFieldBuilder[T]) Required(msg ...string) *StringFieldBuilder[T] {
	b.requiredMsg = firstOrDefault(msg, b.field+" is required")
	return b
}

// MinLength appends a minimum-length rule (trimmed rune count >= min).
func (b *StringFieldBuilder[T]) MinLength(min int, msg ...string) *StringFieldBuilder[T] {
	m := firstOrDefault(msg, fmt.Sprintf("%s must have at least %d characters", b.field, min))
	b.rules = append(b.rules, MinLength(b.extract, min, b.field, m))
	return b
}

// MaxLength appends a maximum-length rule (trimmed rune count <= max).
func (b *StringFieldBuilder[T]) MaxLength(max int, msg ...string) *StringFieldBuilder[T] {
	m := firstOrDefault(msg, fmt.Sprintf("%s must have at most %d characters", b.field, max))
	b.rules = append(b.rules, MaxLength(b.extract, max, b.field, m))
	return b
}

// Email appends an email-format rule.
func (b *StringFieldBuilder[T]) Email(msg ...string) *StringFieldBuilder[T] {
	m := firstOrDefault(msg, b.field+" must be a valid email")
	b.rules = append(b.rules, Email(b.extract, b.field, m))
	return b
}

// UUID appends a UUID RFC 4122 format rule.
func (b *StringFieldBuilder[T]) UUID(msg ...string) *StringFieldBuilder[T] {
	m := firstOrDefault(msg, b.field+" must be a valid UUID")
	b.rules = append(b.rules, UUID(b.extract, b.field, m))
	return b
}

// ISODate appends an ISO 8601 date-time format rule.
func (b *StringFieldBuilder[T]) ISODate(msg ...string) *StringFieldBuilder[T] {
	m := firstOrDefault(msg, b.field+" must be a valid ISO 8601 date")
	b.rules = append(b.rules, ISODate(b.extract, b.field, m))
	return b
}

// OneOf appends a membership rule that checks against a fixed set of allowed values.
func (b *StringFieldBuilder[T]) OneOf(allowed []string, msg ...string) *StringFieldBuilder[T] {
	m := firstOrDefault(msg, fmt.Sprintf("%s must be one of: [%s]", b.field, strings.Join(allowed, ", ")))
	b.rules = append(b.rules, OneOf(b.extract, allowed, b.field, m))
	return b
}

// SafeString appends a content-safety rule (HTML, XSS, invisible chars).
func (b *StringFieldBuilder[T]) SafeString(msg ...string) *StringFieldBuilder[T] {
	m := firstOrDefault(msg, b.field+" contains unsafe content")
	b.rules = append(b.rules, SafeString(b.extract, b.field, m))
	return b
}

// Build returns the collected rules as a [validation.Validation] slice.
// If [StringFieldBuilder.Required] was called, the Required rule is placed first.
// All other rules follow in the order they were added.
func (b *StringFieldBuilder[T]) Build() []validation.Validation[T] {
	out := make([]validation.Validation[T], 0, len(b.rules)+1)
	if b.requiredMsg != "" {
		out = append(out, Required(b.extract, b.field, b.requiredMsg))
	}
	out = append(out, b.rules...)
	return out
}

// ─── IntFieldBuilder ──────────────────────────────────────────────────────────

// IntFieldBuilder constructs [validation.Validation] rules for an int field of T.
type IntFieldBuilder[T any] struct {
	field   string
	extract func(T) int
	rules   []validation.Validation[T]
}

// IntField creates a new [IntFieldBuilder] for the given field name and extractor.
func IntField[T any](field string, extract func(T) int) *IntFieldBuilder[T] {
	return &IntFieldBuilder[T]{field: field, extract: extract}
}

// Min appends a minimum-value rule (extract(input) >= min).
func (b *IntFieldBuilder[T]) Min(min int, msg ...string) *IntFieldBuilder[T] {
	m := firstOrDefault(msg, fmt.Sprintf("%s must be at least %d", b.field, min))
	b.rules = append(b.rules, MinValue[T, int](b.extract, min, b.field, m))
	return b
}

// Max appends a maximum-value rule (extract(input) <= max).
func (b *IntFieldBuilder[T]) Max(max int, msg ...string) *IntFieldBuilder[T] {
	m := firstOrDefault(msg, fmt.Sprintf("%s must be at most %d", b.field, max))
	b.rules = append(b.rules, MaxValue[T, int](b.extract, max, b.field, m))
	return b
}

// Build returns the collected rules as a [validation.Validation] slice.
func (b *IntFieldBuilder[T]) Build() []validation.Validation[T] {
	out := make([]validation.Validation[T], len(b.rules))
	copy(out, b.rules)
	return out
}

// ─── Float64FieldBuilder ──────────────────────────────────────────────────────

// Float64FieldBuilder constructs [validation.Validation] rules for a float64 field of T.
type Float64FieldBuilder[T any] struct {
	field   string
	extract func(T) float64
	rules   []validation.Validation[T]
}

// Float64Field creates a new [Float64FieldBuilder] for the given field name and extractor.
func Float64Field[T any](field string, extract func(T) float64) *Float64FieldBuilder[T] {
	return &Float64FieldBuilder[T]{field: field, extract: extract}
}

// Min appends a minimum-value rule (extract(input) >= min).
func (b *Float64FieldBuilder[T]) Min(min float64, msg ...string) *Float64FieldBuilder[T] {
	m := firstOrDefault(msg, fmt.Sprintf("%s must be at least %g", b.field, min))
	b.rules = append(b.rules, MinValue[T, float64](b.extract, min, b.field, m))
	return b
}

// Max appends a maximum-value rule (extract(input) <= max).
func (b *Float64FieldBuilder[T]) Max(max float64, msg ...string) *Float64FieldBuilder[T] {
	m := firstOrDefault(msg, fmt.Sprintf("%s must be at most %g", b.field, max))
	b.rules = append(b.rules, MaxValue[T, float64](b.extract, max, b.field, m))
	return b
}

// Build returns the collected rules as a [validation.Validation] slice.
func (b *Float64FieldBuilder[T]) Build() []validation.Validation[T] {
	out := make([]validation.Validation[T], len(b.rules))
	copy(out, b.rules)
	return out
}

// ─── BoolFieldBuilder ─────────────────────────────────────────────────────────

// BoolFieldBuilder constructs [validation.Validation] rules for a bool field of T.
type BoolFieldBuilder[T any] struct {
	field   string
	extract func(T) bool
	rules   []validation.Validation[T]
}

// BoolField creates a new [BoolFieldBuilder] for the given field name and extractor.
func BoolField[T any](field string, extract func(T) bool) *BoolFieldBuilder[T] {
	return &BoolFieldBuilder[T]{field: field, extract: extract}
}

// IsTrue appends a rule that fails when extract(input) is false.
func (b *BoolFieldBuilder[T]) IsTrue(msg ...string) *BoolFieldBuilder[T] {
	m := firstOrDefault(msg, b.field+" must be true")
	b.rules = append(b.rules, IsTrue(b.extract, b.field, m))
	return b
}

// IsFalse appends a rule that fails when extract(input) is true.
func (b *BoolFieldBuilder[T]) IsFalse(msg ...string) *BoolFieldBuilder[T] {
	m := firstOrDefault(msg, b.field+" must be false")
	b.rules = append(b.rules, IsFalse(b.extract, b.field, m))
	return b
}

// Build returns the collected rules as a [validation.Validation] slice.
func (b *BoolFieldBuilder[T]) Build() []validation.Validation[T] {
	out := make([]validation.Validation[T], len(b.rules))
	copy(out, b.rules)
	return out
}
