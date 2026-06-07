// Package rules provides typed validation rules and fluent field builders
// for use with [github.com/Lucas-Lopes-II/govalidator/validation].
//
// # Rule functions vs. builders
//
// Rule functions (Required, Email, UUID, …) accept an extractor, field name, and
// explicit message. Use them when you need full control over the error text.
//
// Fluent builders ([StringField], [IntField], [Float64Field], [BoolField]) wrap the
// rule functions with sensible default messages and enforce ordering constraints
// (e.g. [StringFieldBuilder.Required] is always inserted first).
package rules

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Lucas-Lopes-II/govalidator/domainerr"
	"github.com/Lucas-Lopes-II/govalidator/is"
	"github.com/Lucas-Lopes-II/govalidator/validation"
)

// Package-level compiled regexps for [SafeString] content checks.
// MustCompile is safe for constant patterns — panics only at program start.
var (
	reHTML   = regexp.MustCompile(`(?i)<[^>]+>`)
	reEvent  = regexp.MustCompile(`(?i)\bon\w+\s*=`)
	reScheme = regexp.MustCompile(`(?i)(javascript|vbscript|data)\s*:`)
	reCSS    = regexp.MustCompile(`(?i)expression\s*\(`)
)

// isSafeContent reports whether s is free from XSS vectors and invisible chars.
func isSafeContent(s string) bool {
	return !reHTML.MatchString(s) &&
		!reEvent.MatchString(s) &&
		!reScheme.MatchString(s) &&
		!reCSS.MatchString(s) &&
		!hasInvisibleChars(s)
}

// isInvisible reports whether r is an invisible Unicode control character.
// Ranges match those removed by security.StripInvisibleChars.
func isInvisible(r rune) bool {
	return (r >= 0x00 && r <= 0x08) || // NUL…BS
		r == 0x0B || r == 0x0C || // VT, FF
		(r >= 0x0E && r <= 0x1F) || // SO…US
		r == 0x7F || // DEL
		(r >= 0x80 && r <= 0x9F) || // C1 control block
		(r >= 0x200B && r <= 0x200F) || // zero-width chars
		(r >= 0x2028 && r <= 0x202F) || // line/paragraph separators
		r == 0x2060 || // word joiner
		r == 0xFEFF || // BOM
		(r >= 0xFFF9 && r <= 0xFFFB) // interlinear annotation
}

// hasInvisibleChars reports whether s contains any invisible control character.
func hasInvisibleChars(s string) bool {
	for _, r := range s {
		if isInvisible(r) {
			return true
		}
	}
	return false
}

// Required returns a rule that fails when strings.TrimSpace(extract(input)) is empty.
// Always place this as the first rule for mandatory string fields;
// [StringFieldBuilder] enforces this automatically.
func Required[T any](extract func(T) string, field, message string) validation.ValidationFunc[T] {
	return func(input T) error {
		if strings.TrimSpace(extract(input)) == "" {
			return domainerr.NewBadRequest(message)
		}
		return nil
	}
}

// MinLength returns a rule that fails when the trimmed rune count of extract(input) < min.
// Does not check for presence — combine with [Required] for mandatory fields.
func MinLength[T any](extract func(T) string, min int, field, message string) validation.ValidationFunc[T] {
	return func(input T) error {
		if utf8.RuneCountInString(strings.TrimSpace(extract(input))) < min {
			return domainerr.NewBadRequest(message)
		}
		return nil
	}
}

// MaxLength returns a rule that fails when the trimmed rune count of extract(input) > max.
func MaxLength[T any](extract func(T) string, max int, field, message string) validation.ValidationFunc[T] {
	return func(input T) error {
		if utf8.RuneCountInString(strings.TrimSpace(extract(input))) > max {
			return domainerr.NewBadRequest(message)
		}
		return nil
	}
}

// Email returns a rule that fails when extract(input) is not a valid email address.
// Delegates to [is.Email] for the format check.
// Does not verify presence — combine with [Required] for mandatory fields.
func Email[T any](extract func(T) string, field, message string) validation.ValidationFunc[T] {
	return func(input T) error {
		if !is.Email(extract(input)) {
			return domainerr.NewBadRequest(message)
		}
		return nil
	}
}

// UUID returns a rule that fails when extract(input) is not a valid RFC 4122 UUID.
// Delegates to [is.UUID] for the format check.
func UUID[T any](extract func(T) string, field, message string) validation.ValidationFunc[T] {
	return func(input T) error {
		if !is.UUID(extract(input)) {
			return domainerr.NewBadRequest(message)
		}
		return nil
	}
}

// ISODate returns a rule that fails when extract(input) is not a valid ISO 8601
// date-time string. Delegates to [is.ISODate] (accepts RFC3339 and RFC3339Nano).
func ISODate[T any](extract func(T) string, field, message string) validation.ValidationFunc[T] {
	return func(input T) error {
		if !is.ISODate(extract(input)) {
			return domainerr.NewBadRequest(message)
		}
		return nil
	}
}

// SafeString returns a rule that fails when extract(input) contains potentially
// dangerous content: HTML/XML tags, inline event handlers (onclick=…),
// script-scheme URIs (javascript:, vbscript:, data:), CSS expression() calls,
// or invisible Unicode control characters.
func SafeString[T any](extract func(T) string, field, message string) validation.ValidationFunc[T] {
	return func(input T) error {
		if !isSafeContent(extract(input)) {
			return domainerr.NewBadRequest(message)
		}
		return nil
	}
}
