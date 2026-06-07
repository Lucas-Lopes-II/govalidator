package validation

import "github.com/Lucas-Lopes-II/govalidator/domainerr"

// Accumulator collects validation errors via sequential [Accumulator.Check] calls.
// Prefer this over [ValidationComposite] for inline, one-off validation where the
// rules are not reused across multiple use cases.
//
// Difference from [ValidationComposite]:
//   - Accumulator: you extract the value and pass the boolean condition
//   - ValidationComposite: you define reusable rules that receive the full input
//
// Usage:
//
//	func validate(input CreateUserInput) error {
//	    return validation.NewAccumulator().
//	        Check(strings.TrimSpace(input.Name) != "", "name is required").
//	        Check(len(input.Name) >= 2, "name must have at least 2 characters").
//	        Check(is.Email(input.Email), "email is invalid").
//	        Result()
//	}
type Accumulator struct {
	seen map[string]struct{}
	msgs []string
}

// NewAccumulator creates an empty Accumulator ready for use.
func NewAccumulator() *Accumulator {
	return &Accumulator{
		seen: make(map[string]struct{}),
		msgs: make([]string, 0),
	}
}

// Check registers a validation failure when valid is false.
// Duplicate messages are silently ignored (automatic deduplication).
// Returns the receiver for fluent chaining.
func (a *Accumulator) Check(valid bool, message string) *Accumulator {
	if !valid {
		a.addMsg(message)
	}
	return a
}

// CheckField is like [Check] but prefixes the message with "field: ".
// Use this to make per-field failures self-documenting.
//
// Example: CheckField(false, "email", "is invalid") → "email: is invalid"
func (a *Accumulator) CheckField(valid bool, field, message string) *Accumulator {
	return a.Check(valid, field+": "+message)
}

// Result returns the accumulated errors as a typed domain error:
//   - nil                       if no Check failed
//   - *domainerr.BadRequestErr  if exactly one unique message was collected
//   - *domainerr.CompositeErr   if two or more unique messages were collected
func (a *Accumulator) Result() error {
	switch len(a.msgs) {
	case 0:
		return nil
	case 1:
		return domainerr.NewBadRequest(a.msgs[0])
	default:
		return domainerr.NewComposite(a.msgs)
	}
}

// HasErrors reports whether at least one Check has failed.
// Use this to short-circuit processing before calling Result.
func (a *Accumulator) HasErrors() bool { return len(a.msgs) > 0 }

// Messages returns a copy of the messages collected so far.
// Returns an empty (non-nil) slice when no checks have failed.
func (a *Accumulator) Messages() []string {
	out := make([]string, len(a.msgs))
	copy(out, a.msgs)
	return out
}

// MergeErr extracts error messages from err and adds them to the accumulator.
// nil is silently ignored.
//
// Extraction rules:
//   - *domainerr.BadRequestErr or *domainerr.CompositeErr: if Messages() is non-empty,
//     those individual messages are added; otherwise Error() is added.
//   - Any other error type: Error() is added directly.
//
// This enables composing self-validating sub-structs into a single final error:
//
//	func (i CreateOrderInput) Validate() error {
//	    return validation.NewAccumulator().
//	        MergeErr(i.User.Validate()).
//	        MergeErr(i.Address.Validate()).
//	        MergeErr(i.Payment.Validate()).
//	        Result() // returns *CompositeErr with all field errors combined
//	}
func (a *Accumulator) MergeErr(err error) *Accumulator {
	if err == nil {
		return a
	}
	for _, m := range extractMessages(err) {
		a.addMsg(m)
	}
	return a
}

// addMsg appends msg only if it has not been seen before (dedup + order preserved).
func (a *Accumulator) addMsg(msg string) {
	if _, exists := a.seen[msg]; !exists {
		a.seen[msg] = struct{}{}
		a.msgs = append(a.msgs, msg)
	}
}
