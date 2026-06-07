package validation_test

import (
	"errors"
	"testing"

	"github.com/Lucas-Lopes-II/govalidator/domainerr"
	"github.com/Lucas-Lopes-II/govalidator/validation"
)

// ─── test helpers ─────────────────────────────────────────────────────────────

// pass returns a ValidationFunc that always passes.
func pass[T any]() validation.ValidationFunc[T] {
	return func(_ T) error { return nil }
}

// fail returns a ValidationFunc that always fails with the given message.
func fail[T any](msg string) validation.ValidationFunc[T] {
	return func(_ T) error { return domainerr.NewBadRequest(msg) }
}

// failInternal returns a ValidationFunc that returns an *InternalErr.
func failInternal[T any]() validation.ValidationFunc[T] {
	return func(_ T) error { return domainerr.NewInternal("db connection lost") }
}

// ─── ValidationComposite.Validate ─────────────────────────────────────────────

func TestComposite_Validate_NoRules(t *testing.T) {
	t.Parallel()
	type input struct{}

	err := validation.NewComposite[input]().Validate(input{})
	if err != nil {
		t.Errorf("Validate() = %v, want nil (no rules)", err)
	}
}

func TestComposite_Validate_AllPass(t *testing.T) {
	t.Parallel()
	type input struct{}

	err := validation.NewComposite(pass[input](), pass[input]()).Validate(input{})
	if err != nil {
		t.Errorf("Validate() = %v, want nil (all rules pass)", err)
	}
}

func TestComposite_Validate_OneFailure(t *testing.T) {
	t.Parallel()
	type input struct{}

	err := validation.NewComposite(fail[input]("name is required")).Validate(input{})

	var br *domainerr.BadRequestErr
	if !errors.As(err, &br) {
		t.Fatalf("Validate() type = %T, want *domainerr.BadRequestErr", err)
	}
	if br.Error() != "name is required" {
		t.Errorf("Error() = %q, want %q", br.Error(), "name is required")
	}
}

func TestComposite_Validate_TwoFailures(t *testing.T) {
	t.Parallel()
	type input struct{}

	err := validation.NewComposite(
		fail[input]("name is required"),
		fail[input]("email is invalid"),
	).Validate(input{})

	var ce *domainerr.CompositeErr
	if !errors.As(err, &ce) {
		t.Fatalf("Validate() type = %T, want *domainerr.CompositeErr", err)
	}
	if got := len(ce.Messages()); got != 2 {
		t.Fatalf("Messages() len = %d, want 2", got)
	}
}

func TestComposite_Validate_CollectsAllBeforeReturning(t *testing.T) {
	t.Parallel()
	type input struct{}

	err := validation.NewComposite(
		fail[input]("a"),
		pass[input](),
		fail[input]("b"),
		fail[input]("c"),
	).Validate(input{})

	var ce *domainerr.CompositeErr
	if !errors.As(err, &ce) {
		t.Fatalf("type = %T, want *CompositeErr (all failures collected)", err)
	}
	if got := len(ce.Messages()); got != 3 {
		t.Errorf("Messages() len = %d, want 3", got)
	}
}

func TestComposite_Validate_DeduplicatesMessages(t *testing.T) {
	t.Parallel()
	type input struct{}

	// Two rules returning the same message → deduplicated to 1.
	err := validation.NewComposite(
		fail[input]("same message"),
		fail[input]("same message"),
	).Validate(input{})

	var br *domainerr.BadRequestErr
	if !errors.As(err, &br) {
		t.Fatalf("type = %T, want *BadRequestErr (dedup → 1 unique message)", err)
	}
	if br.Error() != "same message" {
		t.Errorf("Error() = %q, want %q", br.Error(), "same message")
	}
}

func TestComposite_Validate_InternalErrFailFast(t *testing.T) {
	t.Parallel()
	type input struct{}

	executed := 0
	counter := validation.ValidationFunc[input](func(_ input) error {
		executed++
		return nil
	})

	err := validation.NewComposite(
		failInternal[input](),
		counter,                  // must NOT be called
		fail[input]("unreached"), // must NOT be collected
	).Validate(input{})

	if !domainerr.IsInternal(err) {
		t.Fatalf("Validate() type = %T, want *domainerr.InternalErr (fail-fast)", err)
	}
	if executed != 0 {
		t.Errorf("counter executed %d time(s), want 0 (stopped at InternalErr)", executed)
	}
}

// ─── ValidationComposite.Add ──────────────────────────────────────────────────

func TestComposite_Add(t *testing.T) {
	t.Parallel()
	type input struct{}

	t.Run("adds validation after creation", func(t *testing.T) {
		t.Parallel()
		c := validation.NewComposite[input]()
		c.Add(fail[input]("late rule"))

		var br *domainerr.BadRequestErr
		if !errors.As(c.Validate(input{}), &br) {
			t.Error("rule added via Add was not executed")
		}
	})

	t.Run("returns receiver for chaining", func(t *testing.T) {
		t.Parallel()
		c := validation.NewComposite[input]()
		if got := c.Add(pass[input]()); got != c {
			t.Error("Add did not return the receiver")
		}
	})
}

// ─── NewCompositeWithBucket ───────────────────────────────────────────────────

func TestCompositeWithBucket_IncludesBucketErrors(t *testing.T) {
	t.Parallel()
	type input struct{}

	bucket := validation.NewBucket()
	bucket.Add(domainerr.NewBadRequest("from bucket"))

	err := validation.NewCompositeWithBucket(bucket,
		fail[input]("from rule"),
	).Validate(input{})

	var ce *domainerr.CompositeErr
	if !errors.As(err, &ce) {
		t.Fatalf("type = %T, want *CompositeErr (bucket + rule)", err)
	}
	if got := len(ce.Messages()); got != 2 {
		t.Errorf("Messages() len = %d, want 2", got)
	}
}

func TestCompositeWithBucket_UsesPeekNotDrain(t *testing.T) {
	t.Parallel()
	type input struct{}

	bucket := validation.NewBucket()
	bucket.Add(domainerr.NewBadRequest("persistent"))

	_ = validation.NewCompositeWithBucket[input](bucket).Validate(input{})

	// Bucket must still hold the error (Peek was used, not Drain).
	if bucket.IsEmpty() {
		t.Error("bucket was consumed (Drain used); expected Peek (non-destructive)")
	}
}

func TestCompositeWithBucket_InternalErrInBucketFailFast(t *testing.T) {
	t.Parallel()
	type input struct{}

	bucket := validation.NewBucket()
	bucket.Add(domainerr.NewInternal("infra failure"))

	err := validation.NewCompositeWithBucket(bucket,
		fail[input]("should not be collected"),
	).Validate(input{})

	if !domainerr.IsInternal(err) {
		t.Fatalf("type = %T, want *InternalErr (fail-fast from bucket)", err)
	}
}

func TestCompositeWithBucket_NilBucketBehavesLikeNewComposite(t *testing.T) {
	t.Parallel()
	type input struct{}

	err := validation.NewCompositeWithBucket[input](nil,
		fail[input]("from rule"),
	).Validate(input{})

	var br *domainerr.BadRequestErr
	if !errors.As(err, &br) {
		t.Fatalf("type = %T, want *BadRequestErr (nil bucket is safe)", err)
	}
}

func TestCompositeWithBucket_NonDomainErrorInBucket(t *testing.T) {
	t.Parallel()
	type input struct{}

	bucket := validation.NewBucket()
	bucket.Add(errors.New("raw stdlib error"))

	err := validation.NewCompositeWithBucket[input](bucket).Validate(input{})

	var br *domainerr.BadRequestErr
	if !errors.As(err, &br) {
		t.Fatalf("type = %T, want *BadRequestErr", err)
	}
	if br.Error() != "raw stdlib error" {
		t.Errorf("Error() = %q, want %q", br.Error(), "raw stdlib error")
	}
}

func TestCompositeWithBucket_OtherDomainErrorInBucket(t *testing.T) {
	t.Parallel()
	type input struct{}

	bucket := validation.NewBucket()
	bucket.Add(domainerr.NewConflict("email already exists"))

	err := validation.NewCompositeWithBucket[input](bucket).Validate(input{})

	var br *domainerr.BadRequestErr
	if !errors.As(err, &br) {
		t.Fatalf("type = %T, want *BadRequestErr (ConflictErr → Error() used)", err)
	}
	if br.Error() != "email already exists" {
		t.Errorf("Error() = %q, want %q", br.Error(), "email already exists")
	}
}

func TestCompositeWithBucket_EmptyCompositeErrInBucket(t *testing.T) {
	t.Parallel()
	type input struct{}

	// *CompositeErr with no messages → extractMessages falls back to Error().
	emptyComposite := domainerr.NewComposite([]string{})
	bucket := validation.NewBucket()
	bucket.Add(emptyComposite)

	err := validation.NewCompositeWithBucket[input](bucket).Validate(input{})

	var br *domainerr.BadRequestErr
	if !errors.As(err, &br) {
		t.Fatalf("type = %T, want *BadRequestErr (CompositeErr with no messages → Error())", err)
	}
	if br.Error() != "Bad Request" {
		t.Errorf("Error() = %q, want %q", br.Error(), "Bad Request")
	}
}

// ─── Nested composites ────────────────────────────────────────────────────────

func TestComposite_NestedComposites_FlattenMessages(t *testing.T) {
	t.Parallel()
	type input struct{}

	// inner1 returns *BadRequestErr("a")
	inner1 := validation.NewComposite(fail[input]("a"))
	// inner2 returns *CompositeErr(["b", "c"])
	inner2 := validation.NewComposite(fail[input]("b"), fail[input]("c"))

	// outer nests both — messages should be flattened: ["a", "b", "c"]
	outer := validation.NewComposite[input](inner1, inner2)
	err := outer.Validate(input{})

	var ce *domainerr.CompositeErr
	if !errors.As(err, &ce) {
		t.Fatalf("type = %T, want *CompositeErr", err)
	}
	if got := len(ce.Messages()); got != 3 {
		t.Errorf("Messages() len = %d, want 3 (a, b, c flattened)", got)
	}
}

func TestComposite_NestedInternalErrPropagates(t *testing.T) {
	t.Parallel()
	type input struct{}

	inner := validation.NewComposite(failInternal[input]())
	outer := validation.NewComposite[input](inner, fail[input]("unreached"))

	err := outer.Validate(input{})
	if !domainerr.IsInternal(err) {
		t.Fatalf("type = %T, want *InternalErr (propagated from nested composite)", err)
	}
}

// ─── ValidationFunc adapter ───────────────────────────────────────────────────

func TestValidationFunc(t *testing.T) {
	t.Parallel()

	type input struct{ Name string }

	rule := validation.ValidationFunc[input](func(i input) error {
		if i.Name == "" {
			return domainerr.NewBadRequest("name is required")
		}
		return nil
	})

	t.Run("passes with valid input", func(t *testing.T) {
		t.Parallel()
		if err := rule.Validate(input{Name: "Lucas"}); err != nil {
			t.Errorf("Validate() = %v, want nil", err)
		}
	})

	t.Run("fails with invalid input", func(t *testing.T) {
		t.Parallel()
		if err := rule.Validate(input{}); err == nil {
			t.Error("Validate() = nil, want error for empty name")
		}
	})
}

// ─── EntityValidatorFn ────────────────────────────────────────────────────────

func TestEntityValidatorFn(t *testing.T) {
	t.Parallel()

	type address struct{ Street string }

	validateAddr := validation.EntityValidatorFn[address](func(a address, b *validation.ErrorBucket) {
		if a.Street == "" {
			b.Add(domainerr.NewBadRequest("street is required"))
		}
	})

	t.Run("invalid input → fills bucket", func(t *testing.T) {
		t.Parallel()
		bucket := validation.NewBucket()
		validateAddr.Run(address{}, bucket)
		if bucket.IsEmpty() {
			t.Error("bucket should not be empty for invalid address")
		}
	})

	t.Run("valid input → bucket stays empty", func(t *testing.T) {
		t.Parallel()
		bucket := validation.NewBucket()
		validateAddr.Run(address{Street: "Rua das Flores, 1"}, bucket)
		if !bucket.IsEmpty() {
			t.Error("bucket should be empty for valid address")
		}
	})

	t.Run("bucket errors flow into NewCompositeWithBucket", func(t *testing.T) {
		t.Parallel()
		bucket := validation.NewBucket()
		validateAddr.Run(address{}, bucket) // adds "street is required"

		err := validation.NewCompositeWithBucket[address](bucket).Validate(address{})

		var br *domainerr.BadRequestErr
		if !errors.As(err, &br) {
			t.Fatalf("type = %T, want *BadRequestErr", err)
		}
		if br.Error() != "street is required" {
			t.Errorf("Error() = %q, want %q", br.Error(), "street is required")
		}
	})
}
