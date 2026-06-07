package validation

import "sync"

// ErrorBucket is a thread-safe error accumulator used to collect errors across
// multiple validation phases before injecting them into a [ValidationComposite].
//
//
// Usage:
//
//	bucket := validation.NewBucket()
//
//	// Phase 1: parse request body
//	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
//	    bucket.Add(domainerr.NewBadRequest("invalid request body"))
//	}
//
//	// Phase 2: domain validation — includes any bucket errors
//	if err := validation.NewCompositeWithBucket(bucket, nameRule, emailRule).Validate(input); err != nil {
//	    domainerr.WriteError(w, err)
//	    return
//	}
type ErrorBucket struct {
	mu   sync.Mutex
	errs []error
}

// NewBucket creates an empty, thread-safe ErrorBucket.
func NewBucket() *ErrorBucket { return &ErrorBucket{} }

// Add appends err to the bucket. nil errors are silently ignored.
// Returns the receiver for fluent chaining.
// Thread-safe.
func (b *ErrorBucket) Add(err error) *ErrorBucket {
	if err == nil {
		return b
	}
	b.mu.Lock()
	b.errs = append(b.errs, err)
	b.mu.Unlock()
	return b
}

// AddAll appends all non-nil errors to the bucket in a single critical section.
// nil elements are silently ignored.
// Returns the receiver for fluent chaining.
// Thread-safe.
func (b *ErrorBucket) AddAll(errs ...error) *ErrorBucket {
	// Pre-filter outside the lock to minimise lock hold time.
	filtered := make([]error, 0, len(errs))
	for _, e := range errs {
		if e != nil {
			filtered = append(filtered, e)
		}
	}
	if len(filtered) == 0 {
		return b
	}
	b.mu.Lock()
	b.errs = append(b.errs, filtered...)
	b.mu.Unlock()
	return b
}

// Drain returns a copy of the accumulated errors and clears the bucket.
// Returns nil if the bucket is empty.
// Thread-safe.
func (b *ErrorBucket) Drain() []error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.errs) == 0 {
		return nil
	}
	out := make([]error, len(b.errs))
	copy(out, b.errs)
	b.errs = nil
	return out
}

// Peek returns a copy of the accumulated errors without clearing the bucket.
// Returns nil if the bucket is empty.
// Thread-safe.
//
// Note: [NewCompositeWithBucket] uses Peek (not [Drain]) so the same bucket can
// be shared across multiple composites without being consumed.
func (b *ErrorBucket) Peek() []error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.errs) == 0 {
		return nil
	}
	out := make([]error, len(b.errs))
	copy(out, b.errs)
	return out
}

// IsEmpty reports whether the bucket contains no errors.
// Thread-safe.
func (b *ErrorBucket) IsEmpty() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.errs) == 0
}
