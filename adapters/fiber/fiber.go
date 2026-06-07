// Package fiberadapter integrates govalidator with the Fiber web framework.
//
// Fiber uses [fasthttp] instead of [net/http], so [domainerr.WriteError] cannot
// be called directly — the adapter bridges the two APIs.
//
// # Registration
//
// Pass [ErrorHandler] via [fiber.Config] at app creation:
//
//	app := fiber.New(fiber.Config{
//	    ErrorHandler: fiberadapter.ErrorHandler,
//	})
//
// From that point, any error returned from a route handler is automatically
// converted to a JSON domain error response. DomainErrors produced by
// govalidator are serialized as-is. Fiber's own *fiber.Error values are
// mapped to the nearest domainerr type. All other errors become HTTP 500.
//
// # Handler example
//
//	func createUser(c *fiber.Ctx) error {
//	    if err := useCase.Execute(input); err != nil {
//	        return err // fiberadapter.ErrorHandler serialises it
//	    }
//	    return c.Status(fiber.StatusCreated).JSON(result)
//	}
//
// # Panic recovery
//
// ErrorHandler is only invoked for returned errors, not for panics. Add
// Fiber's built-in recover middleware to handle panics:
//
//	import "github.com/gofiber/fiber/v2/middleware/recover"
//	app.Use(recover.New())
package fiberadapter

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/Lucas-Lopes-II/govalidator/domainerr"
	"github.com/gofiber/fiber/v2"
)

// ErrorHandler is a [fiber.ErrorHandler] that converts any error to a JSON
// domain error response using govalidator's [domainerr.ErrorResponse] shape.
//
// Behaviour:
//   - If err is a [domainerr.DomainError] (directly or wrapped), it is used as-is.
//   - If err is a [*fiber.Error], it is mapped to the nearest domainerr type by
//     status code (see unexported toFiberError).
//   - Any other error becomes HTTP 500 Internal Server Error.
//
// Logging follows the same policy as [domainerr.WriteError]:
//   - status >= 500: [slog.Error]
//   - status  < 500: [slog.Warn]
//
// Register via fiber.Config:
//
//	app := fiber.New(fiber.Config{ErrorHandler: fiberadapter.ErrorHandler})
func ErrorHandler(c *fiber.Ctx, err error) error {
	resp := domainerr.FromError(toFiberError(err))

	if resp.Status >= 500 {
		slog.Error("internal server error", "err", err)
	} else {
		slog.Warn("domain error", "status", resp.Status, "message", resp.Message)
	}

	return c.Status(resp.Status).JSON(resp)
}

// toFiberError converts err to a domainerr-compatible error.
//
// If err is already a DomainError the value is returned unchanged so the
// original status code and messages are preserved.
// If err is a [*fiber.Error], the status code drives the conversion.
// Anything else is treated as an opaque infrastructure failure (500).
func toFiberError(err error) error {
	if _, ok := domainerr.IsDomain(err); ok {
		return err
	}

	var fe *fiber.Error
	if !errors.As(err, &fe) {
		return domainerr.NewInternal("Internal Server Error")
	}

	switch fe.Code {
	case http.StatusUnauthorized:
		return domainerr.NewUnauthorized(fe.Message)
	case http.StatusForbidden:
		return domainerr.NewForbidden(fe.Message)
	case http.StatusNotFound:
		return domainerr.NewNotFound(fe.Message)
	case http.StatusConflict:
		return domainerr.NewConflict(fe.Message)
	default:
		if fe.Code >= 500 {
			return domainerr.NewInternal(fe.Message)
		}
		return domainerr.NewBadRequest(fe.Message)
	}
}
