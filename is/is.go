// Package is provides pure predicate functions for common validation checks.
// All functions are stateless, have no side effects, and return bool.
//
// These are the building blocks used internally by the [rules] package but are
// also available as a standalone utility for code that only needs boolean checks.
package is

import (
	"regexp"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// emailRegex matches the simplified RFC 5322 local-part@domain.tld format.
// Compiled once at package initialisation; panics on invalid pattern (never happens).
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// UUID returns true if s is a well-formed RFC 4122 UUID (any version/variant).
// Delegates to github.com/google/uuid for spec-compliant parsing.
func UUID(s string) bool {
	if s == "" {
		return false
	}
	_, err := uuid.Parse(s)
	return err == nil
}

// Email returns true if s matches the simplified RFC 5322 email format.
// Validates the structure (local-part@domain.tld); does NOT perform DNS lookup.
func Email(s string) bool {
	return s != "" && emailRegex.MatchString(s)
}

// ISODate returns true if s is a valid ISO 8601 date-time with timezone offset.
// Accepts both [time.RFC3339] ("2006-01-02T15:04:05Z07:00") and
// [time.RFC3339Nano] ("2006-01-02T15:04:05.999999999Z07:00") formats.
// Date-only strings (e.g. "2024-01-25") are rejected.
func ISODate(s string) bool {
	if s == "" {
		return false
	}
	// Try nanosecond precision first (superset of RFC3339).
	if _, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return true
	}
	_, err := time.Parse(time.RFC3339, s)
	return err == nil
}

// StrongPassword returns true if s satisfies all of the following rules:
//  1. At least 6 characters long
//  2. Contains at least one digit (0–9)
//  3. Contains at least one non-alphanumeric character (special character)
//  4. Does not contain a newline ('\n')
//  5. Contains at least one Unicode uppercase letter
//  6. Contains at least one Unicode lowercase letter
func StrongPassword(s string) bool {
	if len(s) < 6 {
		return false
	}
	var hasDigit, hasSpecial, hasUpper, hasLower bool
	for _, r := range s {
		if r == '\n' {
			return false
		}
		switch {
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		default:
			// Anything that is not a letter or digit (spaces, punctuation, symbols).
			hasSpecial = true
		}
	}
	return hasDigit && hasSpecial && hasUpper && hasLower
}

// Latitude returns true if f is within the valid latitude range [-90.0, +90.0].
func Latitude(f float64) bool {
	return f >= -90.0 && f <= 90.0
}

// Longitude returns true if f is within the valid longitude range [-180.0, +180.0].
func Longitude(f float64) bool {
	return f >= -180.0 && f <= 180.0
}
