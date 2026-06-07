package validation

import (
	"errors"

	"github.com/Lucas-Lopes-II/govalidator/domainerr"
)

// ValidationComposite[T] executes a set of [Validation[T]] rules against a single
// input, collects ALL failures before returning (never stops at the first), and
// returns:
//   - nil                        if every rule passed
//   - *domainerr.BadRequestErr   if exactly one unique error message was collected
//   - *domainerr.CompositeErr    if two or more unique messages were collected
//   - *domainerr.InternalErr     immediately (fail-fast) when any rule returns one
//
// Because [ValidationComposite.Validate] satisfies [Validation[T]], composites can
// be nested inside each other for hierarchical validation structures.
type ValidationComposite[T any] struct {
	validations []Validation[T]
	bucket      *ErrorBucket // nil when created via NewComposite
}

// NewComposite creates a ValidationComposite with the provided validations.
// Validations are executed in the order they are given.
// The validations slice is copied; the caller's slice is not retained.
func NewComposite[T any](validations ...Validation[T]) *ValidationComposite[T] {
	c := &ValidationComposite[T]{
		validations: make([]Validation[T], len(validations)),
	}
	copy(c.validations, validations)
	return c
}

// NewCompositeWithBucket creates a ValidationComposite that includes the errors
// accumulated in bucket before running its own rules.
//
// The bucket is read via [ErrorBucket.Peek] (non-destructive) so the same bucket
// can be shared across multiple composites in advanced scenarios.
//
func NewCompositeWithBucket[T any](bucket *ErrorBucket, validations ...Validation[T]) *ValidationComposite[T] {
	c := NewComposite(validations...)
	c.bucket = bucket
	return c
}

// Validate runs all validations in registration order against input.
// Implements [Validation[T]] so composites can be nested.
func (c *ValidationComposite[T]) Validate(input T) error {
	seen := make(map[string]struct{})
	msgs := make([]string, 0)

	// addMsgs deduplicates and appends the messages extracted from err.
	addMsgs := func(err error) {
		for _, m := range extractMessages(err) {
			if _, exists := seen[m]; !exists {
				seen[m] = struct{}{}
				msgs = append(msgs, m)
			}
		}
	}

	// Step 1: read errors from bucket (Peek — non-destructive).
	if c.bucket != nil {
		for _, err := range c.bucket.Peek() {
			if domainerr.IsInternal(err) {
				return err // fail-fast: infrastructure error
			}
			addMsgs(err)
		}
	}

	// Step 2: execute all registered validations in order.
	for _, v := range c.validations {
		err := v.Validate(input)
		if err == nil {
			continue
		}
		if domainerr.IsInternal(err) {
			return err // fail-fast: infrastructure error
		}
		addMsgs(err)
	}

	// Step 3: build the typed result.
	switch len(msgs) {
	case 0:
		return nil
	case 1:
		return domainerr.NewBadRequest(msgs[0])
	default:
		return domainerr.NewComposite(msgs)
	}
}

// Add appends validations to the composite after creation.
// Useful for incremental composition.
// Returns the receiver for fluent chaining.
func (c *ValidationComposite[T]) Add(validations ...Validation[T]) *ValidationComposite[T] {
	c.validations = append(c.validations, validations...)
	return c
}

// ─── package-level helper ─────────────────────────────────────────────────────

// extractMessages extracts the meaningful string messages from err for collection
// and deduplication inside [ValidationComposite] and [Accumulator].
//
// Rules:
//   - *domainerr.BadRequestErr or *domainerr.CompositeErr with Messages() non-empty
//     → returns Messages() (the individual field-level messages)
//   - *domainerr.BadRequestErr or *domainerr.CompositeErr with Messages() empty/nil
//     → returns [Error()] (the primary message)
//   - Any other non-nil error (stdlib, *NotFoundErr, *InternalErr, …)
//     → returns [Error()]
//
// This function is unexported and used by both composite.go and accumulator.go.
func extractMessages(err error) []string {
	var br *domainerr.BadRequestErr
	if errors.As(err, &br) {
		if msgs := br.Messages(); len(msgs) > 0 {
			return msgs
		}
		return []string{br.Error()}
	}

	var ce *domainerr.CompositeErr
	if errors.As(err, &ce) {
		if msgs := ce.Messages(); len(msgs) > 0 {
			return msgs
		}
		return []string{ce.Error()}
	}

	// Any other error type — use the primary message directly.
	return []string{err.Error()}
}
