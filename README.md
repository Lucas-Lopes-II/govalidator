# govalidator

> Domain-aware validation and structured error handling for Go HTTP services.

[![CI](https://github.com/Lucas-Lopes-II/govalidator/actions/workflows/ci.yml/badge.svg)](https://github.com/Lucas-Lopes-II/govalidator/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/Lucas-Lopes-II/govalidator.svg)](https://pkg.go.dev/github.com/Lucas-Lopes-II/govalidator)

## Features

- **Typed domain errors** with HTTP status codes (400 / 401 / 403 / 404 / 409 / 500)
- **Composite validation** — collects *all* field errors before returning, never stops at the first
- **Inline `Accumulator`** for simple, one-off `Check(bool, msg)` validation
- **`ErrorBucket`** for multi-phase error accumulation (parsing + domain validation)
- **Fluent field builders** — `StringField`, `IntField`, `Float64Field`, `BoolField`
- **Input sanitisation** — `NormalizeString`, `StripInvisibleChars`, `StripHTML`, `IsSafeString`
- **HTTP guards** — `RequireUUID`, `SafeSortField`, `SafePageSize`
- **Framework-agnostic** HTTP error writer (`net/http`, works with Gin/Echo via adapters)
- **RFC 7807 Problem Details** opt-in support
- **Zero framework dependencies** (only `github.com/google/uuid`)

## Requirements

- Go 1.22+

## Install

```sh
go get github.com/Lucas-Lopes-II/govalidator
```

## Quick start

### Pattern 1 — Inline Accumulator

Best for one-off validation inside a handler or use case, where the rules are not reused elsewhere.

```go
import (
    "strings"
    "github.com/Lucas-Lopes-II/govalidator/domainerr"
    "github.com/Lucas-Lopes-II/govalidator/is"
    "github.com/Lucas-Lopes-II/govalidator/validation"
)

type CreateUserInput struct {
    Name  string
    Email string
    Age   int
}

func validate(input CreateUserInput) error {
    return validation.NewAccumulator().
        Check(strings.TrimSpace(input.Name) != "", "name is required").
        Check(len(input.Name) >= 2, "name must have at least 2 characters").
        Check(is.Email(input.Email), "email is invalid").
        Check(input.Age >= 18, "must be at least 18 years old").
        Result()
}

// In your handler:
func CreateUser(w http.ResponseWriter, r *http.Request) {
    var input CreateUserInput
    // ... decode r.Body into input ...
    if err := validate(input); err != nil {
        domainerr.WriteError(w, err)
        return
    }
    // ... continue ...
}
```

### Pattern 2 — Reusable ValidationComposite with field builders

Best for validating request DTOs where the same rules are shared across multiple use cases, or when you want to separate validation logic from business logic.

```go
import (
    "github.com/Lucas-Lopes-II/govalidator/rules"
    "github.com/Lucas-Lopes-II/govalidator/validation"
)

type CreateUserInput struct {
    Name  string
    Email string
    Age   int
}

var createUserValidator = validation.NewComposite[CreateUserInput](
    rules.StringField("name", func(i CreateUserInput) string { return i.Name }).
        Required().
        MinLength(2).
        MaxLength(100).
        SafeString().
        Build()...,

    rules.StringField("email", func(i CreateUserInput) string { return i.Email }).
        Required().
        Email().
        MaxLength(254).
        Build()...,

    rules.IntField("age", func(i CreateUserInput) int { return i.Age }).
        Min(18).
        Max(120).
        Build()...,
)

// In your handler:
func CreateUser(w http.ResponseWriter, r *http.Request) {
    var input CreateUserInput
    // ... decode r.Body into input ...
    if err := createUserValidator.Validate(input); err != nil {
        domainerr.WriteError(w, err)
        return
    }
    // ... continue ...
}
```

### Pattern 3 — Raw domain errors

Use typed errors directly in your service or repository layer.

```go
import "github.com/Lucas-Lopes-II/govalidator/domainerr"

func (r *UserRepo) FindByID(id string) (*User, error) {
    user, err := r.db.Find(id)
    if err != nil {
        return nil, domainerr.NewInternal("failed to query user")
    }
    if user == nil {
        return nil, domainerr.NewNotFound("user not found", domainerr.WithDisplayable())
    }
    return user, nil
}

// WriteError serialises any error to JSON — DomainError or not:
//   {"status": 404, "message": "user not found", "displayable": true}
```

## Composing nested structs

### Self-validating structs with `MergeErr` (Pattern B)

```go
type Address struct{ Street, City string }

func (a Address) Validate() error {
    return validation.NewAccumulator().
        Check(a.Street != "", "address.street is required").
        Check(a.City != "",   "address.city is required").
        Result()
}

type CreateOrderInput struct {
    User    UserInfo
    Address Address
}

func (i CreateOrderInput) Validate() error {
    return validation.NewAccumulator().
        MergeErr(i.User.Validate()).
        MergeErr(i.Address.Validate()).
        Result() // *CompositeErr with ALL sub-struct errors combined
}
```

### Bucket-passing with `ErrorBucket` (Pattern A)

```go
bucket := validation.NewBucket()
validateAddress(input.Address, bucket)   // child writes into shared bucket
validatePayment(input.Payment, bucket)   // another child

return validation.NewCompositeWithBucket(bucket, parentRules...).Validate(input)
```

## Input sanitisation

```go
import "github.com/Lucas-Lopes-II/govalidator/security"

// Always sanitise before validation:
input.Name = security.NormalizeString(input.Name)   // strip invisible chars + trim
input.Bio  = security.StripHTML(input.Bio)           // remove HTML tags

// Guards for HTTP path/query params:
id, err := security.RequireUUID(r.PathValue("id"), "id")
if err != nil {
    domainerr.WriteError(w, err)
    return
}

sortCol := security.SafeSortField(r.URL.Query().Get("sort"),
    map[string]struct{}{"name": {}, "created_at": {}},
    "created_at",
)
limit := security.SafePageSize(pageSize, 20, 100)
```

## Adapters

### Gin

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/Lucas-Lopes-II/govalidator/domainerr"
)

func ErrorMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        if len(c.Errors) > 0 {
            domainerr.WriteError(c.Writer, c.Errors.Last().Err)
        }
    }
}

// In a handler — abort immediately:
func CreateUser(c *gin.Context) {
    if err := useCase.Execute(input); err != nil {
        domainerr.WriteError(c.Writer, err)
        c.Abort()
        return
    }
    c.JSON(201, result)
}
```

### Echo

```go
import (
    "github.com/labstack/echo/v4"
    "github.com/Lucas-Lopes-II/govalidator/domainerr"
)

func ErrorHandler(err error, c echo.Context) {
    domainerr.WriteError(c.Response(), err)
}

// Register on your Echo instance:
e.HTTPErrorHandler = ErrorHandler
```

### RFC 7807 Problem Details

```go
resp := domainerr.FromError(err)
pd := resp.ToProblemDetail("https://errors.myapp.com", r.URL.Path)

w.Header().Set("Content-Type", "application/problem+json")
w.WriteHeader(pd.Status)
json.NewEncoder(w).Encode(pd)
```

## API Reference

Full documentation is available on **[pkg.go.dev](https://pkg.go.dev/github.com/Lucas-Lopes-II/govalidator)**.

| Package | Responsibility |
|---------|---------------|
| `domainerr` | Typed errors + HTTP serialisation + panic-recovery middleware |
| `validation` | `Validation[T]` interface, `ValidationComposite[T]`, `Accumulator`, `ErrorBucket` |
| `rules` | Built-in rule functions and fluent field builders |
| `is` | Pure predicates: `UUID`, `Email`, `ISODate`, `StrongPassword`, `Latitude`, `Longitude` |
| `security` | Input sanitisation and HTTP guard utilities |
