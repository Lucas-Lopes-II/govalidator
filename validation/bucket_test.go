package validation_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/Lucas-Lopes-II/govalidator/validation"
)

func TestBucket_Add(t *testing.T) {
	t.Parallel()

	t.Run("nil is ignored", func(t *testing.T) {
		t.Parallel()
		b := validation.NewBucket()
		b.Add(nil)
		if !b.IsEmpty() {
			t.Error("IsEmpty() = false after Add(nil), want true")
		}
	})

	t.Run("non-nil error is stored", func(t *testing.T) {
		t.Parallel()
		b := validation.NewBucket()
		b.Add(errors.New("oops"))
		if b.IsEmpty() {
			t.Error("IsEmpty() = true after Add(non-nil), want false")
		}
	})

	t.Run("returns receiver for chaining", func(t *testing.T) {
		t.Parallel()
		b := validation.NewBucket()
		if got := b.Add(errors.New("e")); got != b {
			t.Error("Add did not return the receiver")
		}
	})
}

func TestBucket_AddAll(t *testing.T) {
	t.Parallel()

	t.Run("all nils → bucket stays empty", func(t *testing.T) {
		t.Parallel()
		b := validation.NewBucket()
		b.AddAll(nil, nil, nil)
		if !b.IsEmpty() {
			t.Error("IsEmpty() = false after AddAll(nil, nil, nil), want true")
		}
	})

	t.Run("mixed nils and errors → only non-nil added", func(t *testing.T) {
		t.Parallel()
		b := validation.NewBucket()
		b.AddAll(errors.New("a"), nil, errors.New("b"))
		if got := len(b.Peek()); got != 2 {
			t.Errorf("Peek() len = %d, want 2", got)
		}
	})

	t.Run("empty varargs → no-op", func(t *testing.T) {
		t.Parallel()
		b := validation.NewBucket()
		b.AddAll()
		if !b.IsEmpty() {
			t.Error("IsEmpty() = false after AddAll(), want true")
		}
	})

	t.Run("returns receiver for chaining", func(t *testing.T) {
		t.Parallel()
		b := validation.NewBucket()
		if got := b.AddAll(errors.New("e")); got != b {
			t.Error("AddAll did not return the receiver")
		}
	})
}

func TestBucket_Drain(t *testing.T) {
	t.Parallel()

	t.Run("empty bucket → returns nil", func(t *testing.T) {
		t.Parallel()
		b := validation.NewBucket()
		if got := b.Drain(); got != nil {
			t.Errorf("Drain() on empty = %v, want nil", got)
		}
	})

	t.Run("returns all errors", func(t *testing.T) {
		t.Parallel()
		b := validation.NewBucket()
		b.Add(errors.New("a")).Add(errors.New("b"))
		got := b.Drain()
		if len(got) != 2 {
			t.Errorf("Drain() len = %d, want 2", len(got))
		}
	})

	t.Run("clears the bucket", func(t *testing.T) {
		t.Parallel()
		b := validation.NewBucket()
		b.Add(errors.New("x"))
		b.Drain()
		if !b.IsEmpty() {
			t.Error("IsEmpty() = false after Drain(), want true")
		}
	})

	t.Run("returns copy — mutation does not affect bucket", func(t *testing.T) {
		t.Parallel()
		b := validation.NewBucket()
		b.Add(errors.New("original"))
		// Drain clears; add a new error and drain again to verify isolation.
		first := b.Drain()
		first[0] = errors.New("mutated")
		// Bucket was cleared; a second drain must return nil.
		if second := b.Drain(); second != nil {
			t.Error("second Drain() should return nil (bucket was cleared)")
		}
	})
}

func TestBucket_Peek(t *testing.T) {
	t.Parallel()

	t.Run("empty bucket → returns nil", func(t *testing.T) {
		t.Parallel()
		b := validation.NewBucket()
		if got := b.Peek(); got != nil {
			t.Errorf("Peek() on empty = %v, want nil", got)
		}
	})

	t.Run("does not clear the bucket", func(t *testing.T) {
		t.Parallel()
		b := validation.NewBucket()
		b.Add(errors.New("a"))
		_ = b.Peek()
		if b.IsEmpty() {
			t.Error("bucket should NOT be empty after Peek()")
		}
	})

	t.Run("returns copy — mutation does not affect bucket", func(t *testing.T) {
		t.Parallel()
		sentinel := errors.New("original")
		b := validation.NewBucket()
		b.Add(sentinel)

		got := b.Peek()
		got[0] = errors.New("mutated")

		second := b.Peek()
		if second[0] != sentinel {
			t.Error("Peek() returned a reference to internal slice; want defensive copy")
		}
	})

	t.Run("consecutive Peek calls return same content", func(t *testing.T) {
		t.Parallel()
		b := validation.NewBucket()
		b.Add(errors.New("x")).Add(errors.New("y"))
		first := b.Peek()
		second := b.Peek()
		if len(first) != len(second) {
			t.Errorf("Peek() lengths differ: %d vs %d", len(first), len(second))
		}
	})
}

func TestBucket_IsEmpty(t *testing.T) {
	t.Parallel()

	b := validation.NewBucket()
	if !b.IsEmpty() {
		t.Error("new bucket should be empty")
	}
	b.Add(errors.New("e"))
	if b.IsEmpty() {
		t.Error("bucket should not be empty after Add")
	}
	b.Drain()
	if !b.IsEmpty() {
		t.Error("bucket should be empty after Drain")
	}
}

func TestBucket_ConcurrentAdd(t *testing.T) {
	t.Parallel()

	const goroutines = 100
	b := validation.NewBucket()
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Go 1.22: for range N over integer.
	for range goroutines {
		go func() {
			defer wg.Done()
			b.Add(errors.New("concurrent"))
		}()
	}
	wg.Wait()

	if got := len(b.Peek()); got != goroutines {
		t.Errorf("after %d concurrent Add calls, Peek() len = %d, want %d",
			goroutines, got, goroutines)
	}
}

func TestBucket_ConcurrentAddAll(t *testing.T) {
	t.Parallel()

	const goroutines = 50
	b := validation.NewBucket()
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			b.AddAll(errors.New("x"), errors.New("y"))
		}()
	}
	wg.Wait()

	if got := len(b.Peek()); got != goroutines*2 {
		t.Errorf("after %d concurrent AddAll calls (2 each), Peek() len = %d, want %d",
			goroutines, got, goroutines*2)
	}
}
