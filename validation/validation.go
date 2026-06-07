// Package validation provides the core abstractions for composable, type-safe
// validation in Go HTTP services.
//
// # Core types
//
//   - [Validation[T]]          — interface for a single reusable rule
//   - [ValidationFunc[T]]      — function adapter implementing Validation[T]
//   - [ValidationComposite[T]] — collects ALL field errors before returning
//   - [Accumulator]            — inline Check(bool, msg) pattern for one-off validation
//   - [ErrorBucket]            — thread-safe error accumulator for multi-phase flows
//   - [EntityValidatorFn[T]]   — bucket-passing pattern (Java DomainValidator<T> equivalent)
//
// # Dependency
//
// This package imports only [github.com/Lucas-Lopes-II/govalidator/domainerr].
// The [rules] package depends on this package, not the other way around.
package validation

// Validation[T] is the base interface for a reusable validation rule over inputs
// of type T. All rule functions in the [rules] package return [ValidationFunc[T]]
// which implements this interface.
//
// Return contract:
//   - nil                  — input passes the rule
//   - *domainerr.BadRequestErr — invalid client input
//   - *domainerr.InternalErr   — infrastructure failure (causes fail-fast in [ValidationComposite])
//
// Returning any other error type is allowed but may produce unexpected behaviour
// inside [ValidationComposite].
type Validation[T any] interface {
	Validate(input T) error
}

// ValidationFunc[T] is a function adapter that makes a plain func satisfy [Validation[T]].
// Use this to define a one-off rule without creating a struct.
//
// Example:
//
//	var checkName validation.Validation[CreateUserInput] = validation.ValidationFunc[CreateUserInput](
//	    func(in CreateUserInput) error {
//	        if strings.TrimSpace(in.Name) == "" {
//	            return domainerr.NewBadRequest("name is required")
//	        }
//	        return nil
//	    },
//	)
type ValidationFunc[T any] func(T) error

// Validate implements [Validation[T]].
func (f ValidationFunc[T]) Validate(input T) error { return f(input) }

// EntityValidatorFn[T] validates an entity of type T by writing errors directly
// into a shared [ErrorBucket]. It is used when a child validator fills the bucket
// and the parent calls [NewCompositeWithBucket] to include those errors.
//
//
// Example:
//
//	var validateAddr validation.EntityValidatorFn[Address] = func(a Address, b *validation.ErrorBucket) {
//	    if strings.TrimSpace(a.Street) == "" {
//	        b.Add(domainerr.NewBadRequest("address.street is required"))
//	    }
//	}
//
//	bucket := validation.NewBucket()
//	validateAddr.Run(order.Address, bucket)
//	return validation.NewCompositeWithBucket(bucket, parentRules...).Validate(order)
type EntityValidatorFn[T any] func(entity T, bucket *ErrorBucket)

// Run executes the EntityValidatorFn, passing entity and bucket.
// Provided as a convenience for fluent call chains.
func (f EntityValidatorFn[T]) Run(entity T, bucket *ErrorBucket) { f(entity, bucket) }
