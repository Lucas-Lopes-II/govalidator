// Package security provides input sanitisation helpers and HTTP handler guards.
//
// # Sanitisation
//
// Call [NormalizeString] before any other validation — it removes invisible
// control characters and trims surrounding whitespace in one pass. Use
// [StripInvisibleChars] when you need to remove invisible chars without
// collapsing them to spaces. Use [IsSafeString] as a pure boolean gate; use
// [StripHTML] to strip markup before persisting or displaying user text.
//
// # Guards
//
// [RequireUUID] validates path / query parameters that must be UUIDs and
// returns a typed domain error ready to be passed to WriteError.
// [SafeSortField] and [SafePageSize] silently fall back to safe defaults,
// protecting against SQL ORDER BY injection and out-of-range pagination.
package security

import (
	"regexp"
	"strings"
)

// Package-level compiled regexps — constant patterns, MustCompile is safe.
var (
	reSanitizeHTML   = regexp.MustCompile(`(?i)<[^>]+>`)
	reSanitizeEvent  = regexp.MustCompile(`(?i)\bon\w+\s*=`)
	reSanitizeScheme = regexp.MustCompile(`(?i)(javascript|vbscript|data)\s*:`)
	reSanitizeCSS    = regexp.MustCompile(`(?i)(expression\s*\(|url\s*\()`)
)

// isInvisible reports whether r is an invisible Unicode control character that
// should be removed or replaced during sanitisation.
//
// Covered ranges:
//
//	U+0000–U+0008  NUL…BS
//	U+000B–U+000C  VT, FF
//	U+000E–U+001F  SO…US
//	U+007F         DEL
//	U+0080–U+009F  C1 control block
//	U+200B–U+200F  zero-width chars (ZWSP, ZWNJ, ZWJ, LRM, RLM)
//	U+2028–U+202F  line/paragraph separators + bidi controls
//	U+2060         word joiner
//	U+FEFF         BOM / zero-width no-break space
//	U+FFF9–U+FFFB  interlinear annotation anchors
func isInvisible(r rune) bool {
	return (r >= 0x00 && r <= 0x08) ||
		r == 0x0B || r == 0x0C ||
		(r >= 0x0E && r <= 0x1F) ||
		r == 0x7F ||
		(r >= 0x80 && r <= 0x9F) ||
		(r >= 0x200B && r <= 0x200F) ||
		(r >= 0x2028 && r <= 0x202F) ||
		r == 0x2060 ||
		r == 0xFEFF ||
		(r >= 0xFFF9 && r <= 0xFFFB)
}

// NormalizeString returns s with invisible control characters replaced by a
// single space and leading/trailing whitespace stripped.
//
// This is the recommended first step before any further validation: it
// eliminates invisible characters that would pass a non-empty check while
// still allowing a reader to enter a legitimately-blank value.
func NormalizeString(s string) string {
	mapped := strings.Map(func(r rune) rune {
		if isInvisible(r) {
			return ' '
		}
		return r
	}, s)
	return strings.TrimSpace(mapped)
}

// StripInvisibleChars returns s with all invisible Unicode control characters
// removed (not replaced). The relative order of visible characters is preserved.
//
// Invisible characters are defined by the ranges documented on [isInvisible].
func StripInvisibleChars(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if !isInvisible(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// IsSafeString returns true when s is free from the most common XSS injection
// vectors:
//   - HTML/XML tags (<tag>)
//   - Inline event handlers (onclick=, onload=, …)
//   - Script-scheme URIs (javascript:, vbscript:, data:)
//   - CSS expression() and url() calls
//
// IsSafeString does NOT check for invisible control characters; call
// [StripInvisibleChars] or [NormalizeString] for that.
func IsSafeString(s string) bool {
	return !reSanitizeHTML.MatchString(s) &&
		!reSanitizeEvent.MatchString(s) &&
		!reSanitizeScheme.MatchString(s) &&
		!reSanitizeCSS.MatchString(s)
}

// StripHTML removes all HTML/XML tags from s and returns the result.
//
// This function does NOT decode HTML entities (e.g. &amp; remains as-is).
// It is intended for removing markup before plain-text storage or indexing,
// not for producing HTML-safe output — use html.EscapeString for that.
func StripHTML(s string) string {
	return reSanitizeHTML.ReplaceAllString(s, "")
}
