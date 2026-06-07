// Package echoadapter integrates govalidator with the Echo web framework.
//
// # Registration
//
// Assign [ErrorHandler] to the Echo instance once at startup:
//
//	e := echo.New()
//	e.HTTPErrorHandler = echoadapter.ErrorHandler
//
// From that point, any error returned from a handler is automatically converted
// to a JSON domain error response. Domain errors produced by govalidator are
// serialized as-is. Echo's own *echo.HTTPError values are mapped to the nearest
// domainerr type. All other errors become HTTP 500.
//
// # Handler example
//
//	func createUser(c echo.Context) error {
//	    if err := useCase.Execute(input); err != nil {
//	        return err // echoadapter.ErrorHandler serialises it
//	    }
//	    return c.JSON(201, result)
//	}
package echoadapter

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Lucas-Lopes-II/govalidator/domainerr"
	"github.com/labstack/echo/v4"
)

// ErrorHandler is an echo.HTTPErrorHandler that serializes any error to a JSON
// domain error response.
//
// Behaviour:
//   - If the response is already committed (c.Response().Committed), the function
//     returns immediately to prevent a double-write panic.
//   - If err is a [domainerr.DomainError] (directly or wrapped), it is written as-is.
//   - If err is a *echo.HTTPError, it is mapped to the nearest domainerr type by
//     status code (see unexported toEchoError).
//   - Any other error becomes HTTP 500 Internal Server Error.
//
// Register as:
//
//	e.HTTPErrorHandler = echoadapter.ErrorHandler
func ErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	domainerr.WriteError(c.Response(), toEchoError(err))
}

// toEchoError converts err to a domainerr-compatible error.
//
// If err is already a DomainError the value is returned unchanged so the
// original status code and messages are preserved.
// If err is a *echo.HTTPError, the status code drives the conversion.
// Anything else is treated as an opaque infrastructure failure (500).
func toEchoError(err error) error {
	if _, ok := domainerr.IsDomain(err); ok {
		return err
	}

	var he *echo.HTTPError
	if !errors.As(err, &he) {
		return domainerr.NewInternal("Internal Server Error")
	}

	msg := fmt.Sprintf("%v", he.Message)
	switch he.Code {
	case http.StatusUnauthorized:
		return domainerr.NewUnauthorized(msg)
	case http.StatusForbidden:
		return domainerr.NewForbidden(msg)
	case http.StatusNotFound:
		return domainerr.NewNotFound(msg)
	case http.StatusConflict:
		return domainerr.NewConflict(msg)
	default:
		if he.Code >= 500 {
			return domainerr.NewInternal(msg)
		}
		return domainerr.NewBadRequest(msg)
	}
}
