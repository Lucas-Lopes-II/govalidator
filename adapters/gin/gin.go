// Package ginadapter integrates govalidator with the Gin web framework.
//
// # Middleware pattern (c.Error)
//
// Register [Middleware] once as the outermost handler. Handlers accumulate
// errors via c.Error(err) and return without writing a response; the
// middleware serializes the last error after c.Next() completes.
//
//	r := gin.New()
//	r.Use(ginadapter.Middleware())
//
//	func CreateUser(c *gin.Context) {
//	    if err := useCase.Execute(input); err != nil {
//	        _ = c.Error(err)
//	        return
//	    }
//	    c.JSON(201, result)
//	}
//
// # Direct pattern (WriteError)
//
// Use [WriteError] when you want to write the error response and abort
// the chain in a single call, without going through c.Error:
//
//	func CreateUser(c *gin.Context) {
//	    if err := useCase.Execute(input); err != nil {
//	        ginadapter.WriteError(c, err)
//	        return
//	    }
//	    c.JSON(201, result)
//	}
package ginadapter

import (
	"fmt"
	"log/slog"

	"github.com/Lucas-Lopes-II/govalidator/domainerr"
	"github.com/gin-gonic/gin"
)

// Middleware returns a gin.HandlerFunc that:
//  1. Recovers from panics and serializes them as HTTP 500 (never leaks stack traces)
//  2. After c.Next() completes, checks c.Errors for accumulated errors and
//     serializes the last one using [domainerr.WriteError]
//
// If the handler already wrote a response (e.g. c.JSON was called), the
// middleware does nothing — it never overwrites a committed response.
//
// Register as the outermost middleware so it wraps the full handler chain:
//
//	r := gin.New()
//	r.Use(ginadapter.Middleware())
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered in gin middleware", "panic", fmt.Sprintf("%v", r))
				if !c.Writer.Written() {
					domainerr.WriteError(c.Writer, domainerr.NewInternal("Internal Server Error"))
				}
				c.Abort()
			}
		}()

		c.Next()

		if c.Writer.Written() || len(c.Errors) == 0 {
			return
		}
		domainerr.WriteError(c.Writer, c.Errors.Last().Err)
	}
}

// WriteError serializes err as JSON using [domainerr.WriteError] and then
// calls c.Abort() to stop all pending handlers in the chain.
//
// Use this for immediate early returns in handlers. It is equivalent to
// calling [domainerr.WriteError] on c.Writer followed by c.Abort().
func WriteError(c *gin.Context, err error) {
	domainerr.WriteError(c.Writer, err)
	c.Abort()
}
