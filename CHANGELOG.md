# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v0.1.0] - 2026-06-07

### Added

#### `domainerr`
- `DomainError` interface: `StatusCode() int`, `Messages() []string`, `Displayable() bool`
- Seven typed error constructors: `NewBadRequest`, `NewUnauthorized`, `NewForbidden`, `NewNotFound`, `NewConflict`, `NewInternal`, `NewComposite`
- Functional options: `WithMessages`, `WithDisplayable`
- Inspection helpers: `IsDomain`, `IsInternal`
- `ErrorResponse` DTO with JSON serialisation
- `FromError` / `FromDomainError` conversion functions
- RFC 7807 `ProblemDetail` type and `ToProblemDetail` method (opt-in)
- `WriteError` — framework-agnostic JSON error writer with structured logging
- `Middleware` — `net/http` panic-recovery middleware

#### `validation`
- `Validation[T]` interface and `ValidationFunc[T]` function adapter
- `EntityValidatorFn[T]` for bucket-passing validation patterns
- `ValidationComposite[T]` — collects all errors, fail-fast on `*InternalErr`, deduplicates messages
- `NewCompositeWithBucket` — composite that drains an `ErrorBucket` before its own rules
- `Accumulator` — inline `Check(bool, msg)` validation with fluent chaining, `MergeErr` for nested structs
- `ErrorBucket` — thread-safe multi-phase error accumulator (`Add`, `AddAll`, `Drain`, `Peek`, `IsEmpty`)

#### `rules`
- String rules: `Required`, `MinLength`, `MaxLength`, `Email`, `UUID`, `ISODate`, `SafeString`
- Number rules: `MinValue`, `MaxValue` (generic `Number` constraint — no external deps)
- Bool rules: `IsTrue`, `IsFalse`
- Enum rule: `OneOf` (O(1) set lookup, snapshot at construction time)
- Fluent builders: `StringFieldBuilder`, `IntFieldBuilder`, `Float64FieldBuilder`, `BoolFieldBuilder`
- `Required` is always the first rule in `StringFieldBuilder.Build()` regardless of call order

#### `is`
- `UUID(s string) bool` — RFC 4122 via `github.com/google/uuid`
- `Email(s string) bool` — simplified RFC 5322 format
- `ISODate(s string) bool` — RFC3339 and RFC3339Nano with timezone
- `StrongPassword(s string) bool` — length, digit, special char, upper, lower, no newline
- `Latitude(f float64) bool` — range [-90, +90]
- `Longitude(f float64) bool` — range [-180, +180]

#### `security`
- `NormalizeString` — replace invisible chars with space, then trim
- `StripInvisibleChars` — remove 30+ invisible Unicode control characters
- `IsSafeString` — XSS vector detection (HTML tags, event handlers, script URIs, CSS expressions)
- `StripHTML` — strip all HTML/XML markup
- `RequireUUID` — validate UUID path/query parameters, return `*BadRequestErr` on failure
- `SafeSortField` — whitelist-based sort column guard (SQL injection protection)
- `SafePageSize` — bounded pagination guard

#### Infrastructure
- GitHub Actions CI: build, `go vet`, `-race` tests, coverage threshold (≥ 85%)
- `.editorconfig` and `.gitignore`
