package validation_test

import (
	"errors"
	"testing"

	"github.com/Lucas-Lopes-II/govalidator/domainerr"
	"github.com/Lucas-Lopes-II/govalidator/validation"
)

func TestAccumulator_Check(t *testing.T) {
	t.Parallel()

	t.Run("true → nothing collected", func(t *testing.T) {
		t.Parallel()
		a := validation.NewAccumulator()
		a.Check(true, "should not appear")
		if a.HasErrors() {
			t.Error("HasErrors() = true after Check(true, ...), want false")
		}
	})

	t.Run("false → message collected", func(t *testing.T) {
		t.Parallel()
		a := validation.NewAccumulator()
		a.Check(false, "name is required")
		if !a.HasErrors() {
			t.Error("HasErrors() = false after Check(false, ...), want true")
		}
	})

	t.Run("duplicate message appears only once", func(t *testing.T) {
		t.Parallel()
		a := validation.NewAccumulator()
		a.Check(false, "email is invalid").
			Check(false, "email is invalid")
		if got := len(a.Messages()); got != 1 {
			t.Errorf("Messages() len = %d, want 1 (dedup)", got)
		}
	})

	t.Run("returns receiver for chaining", func(t *testing.T) {
		t.Parallel()
		a := validation.NewAccumulator()
		if got := a.Check(false, "msg"); got != a {
			t.Error("Check did not return the receiver")
		}
	})

	t.Run("preserves insertion order", func(t *testing.T) {
		t.Parallel()
		a := validation.NewAccumulator()
		a.Check(false, "first").
			Check(false, "second").
			Check(false, "third")
		msgs := a.Messages()
		want := []string{"first", "second", "third"}
		for i, w := range want {
			if msgs[i] != w {
				t.Errorf("Messages()[%d] = %q, want %q", i, msgs[i], w)
			}
		}
	})
}

func TestAccumulator_CheckField(t *testing.T) {
	t.Parallel()

	t.Run("false → prefixes 'field: message'", func(t *testing.T) {
		t.Parallel()
		a := validation.NewAccumulator()
		a.CheckField(false, "email", "is invalid")
		msgs := a.Messages()
		if len(msgs) != 1 || msgs[0] != "email: is invalid" {
			t.Errorf("Messages() = %v, want [%q]", msgs, "email: is invalid")
		}
	})

	t.Run("true → nothing collected", func(t *testing.T) {
		t.Parallel()
		a := validation.NewAccumulator()
		a.CheckField(true, "name", "is required")
		if a.HasErrors() {
			t.Error("HasErrors() = true after CheckField(true, ...), want false")
		}
	})
}

func TestAccumulator_Result(t *testing.T) {
	t.Parallel()

	t.Run("no errors → nil", func(t *testing.T) {
		t.Parallel()
		if err := validation.NewAccumulator().Result(); err != nil {
			t.Errorf("Result() = %v, want nil", err)
		}
	})

	t.Run("one error → *BadRequestErr with correct message", func(t *testing.T) {
		t.Parallel()
		err := validation.NewAccumulator().
			Check(false, "name is required").
			Result()

		var br *domainerr.BadRequestErr
		if !errors.As(err, &br) {
			t.Fatalf("Result() type = %T, want *domainerr.BadRequestErr", err)
		}
		if br.Error() != "name is required" {
			t.Errorf("Error() = %q, want %q", br.Error(), "name is required")
		}
	})

	t.Run("two errors → *CompositeErr", func(t *testing.T) {
		t.Parallel()
		err := validation.NewAccumulator().
			Check(false, "name is required").
			Check(false, "email is invalid").
			Result()

		var ce *domainerr.CompositeErr
		if !errors.As(err, &ce) {
			t.Fatalf("Result() type = %T, want *domainerr.CompositeErr", err)
		}
		if got := len(ce.Messages()); got != 2 {
			t.Errorf("Messages() len = %d, want 2", got)
		}
	})

	t.Run("three errors → *CompositeErr with all messages", func(t *testing.T) {
		t.Parallel()
		err := validation.NewAccumulator().
			Check(false, "a").
			Check(false, "b").
			Check(false, "c").
			Result()

		var ce *domainerr.CompositeErr
		if !errors.As(err, &ce) {
			t.Fatalf("Result() type = %T, want *domainerr.CompositeErr", err)
		}
		if got := len(ce.Messages()); got != 3 {
			t.Errorf("Messages() len = %d, want 3", got)
		}
	})
}

func TestAccumulator_Messages(t *testing.T) {
	t.Parallel()

	t.Run("empty accumulator → empty non-nil slice", func(t *testing.T) {
		t.Parallel()
		msgs := validation.NewAccumulator().Messages()
		if msgs == nil {
			t.Error("Messages() = nil, want empty non-nil slice")
		}
		if len(msgs) != 0 {
			t.Errorf("Messages() len = %d, want 0", len(msgs))
		}
	})

	t.Run("returns copy — mutation does not affect accumulator", func(t *testing.T) {
		t.Parallel()
		a := validation.NewAccumulator()
		a.Check(false, "original")

		first := a.Messages()
		first[0] = "mutated"

		second := a.Messages()
		if second[0] != "original" {
			t.Error("Messages() returned a reference to internal slice; want defensive copy")
		}
	})
}

func TestAccumulator_MergeErr(t *testing.T) {
	t.Parallel()

	t.Run("nil → ignored", func(t *testing.T) {
		t.Parallel()
		a := validation.NewAccumulator()
		a.MergeErr(nil)
		if a.HasErrors() {
			t.Error("HasErrors() = true after MergeErr(nil), want false")
		}
	})

	t.Run("*BadRequestErr (no WithMessages) → adds Error()", func(t *testing.T) {
		t.Parallel()
		br := domainerr.NewBadRequest("name is required")
		a := validation.NewAccumulator()
		a.MergeErr(br)

		msgs := a.Messages()
		if len(msgs) != 1 || msgs[0] != "name is required" {
			t.Errorf("Messages() = %v, want [%q]", msgs, "name is required")
		}
	})

	t.Run("*BadRequestErr (with WithMessages) → adds Messages()", func(t *testing.T) {
		t.Parallel()
		br := domainerr.NewBadRequest("summary", domainerr.WithMessages("detail-a", "detail-b"))
		a := validation.NewAccumulator()
		a.MergeErr(br)

		msgs := a.Messages()
		if len(msgs) != 2 {
			t.Fatalf("Messages() len = %d, want 2", len(msgs))
		}
	})

	t.Run("*CompositeErr → adds all Messages()", func(t *testing.T) {
		t.Parallel()
		ce := domainerr.NewComposite([]string{"name is required", "email is invalid"})
		a := validation.NewAccumulator()
		a.MergeErr(ce)

		msgs := a.Messages()
		if len(msgs) != 2 {
			t.Fatalf("Messages() len = %d, want 2", len(msgs))
		}
		if msgs[0] != "name is required" || msgs[1] != "email is invalid" {
			t.Errorf("Messages() = %v", msgs)
		}
	})

	t.Run("generic stdlib error → adds Error()", func(t *testing.T) {
		t.Parallel()
		generic := errors.New("database timeout")
		a := validation.NewAccumulator()
		a.MergeErr(generic)

		msgs := a.Messages()
		if len(msgs) != 1 || msgs[0] != "database timeout" {
			t.Errorf("Messages() = %v, want [%q]", msgs, "database timeout")
		}
	})

	t.Run("other DomainError type (*ConflictErr) → adds Error()", func(t *testing.T) {
		t.Parallel()
		a := validation.NewAccumulator()
		a.MergeErr(domainerr.NewConflict("email already exists"))

		msgs := a.Messages()
		if len(msgs) != 1 || msgs[0] != "email already exists" {
			t.Errorf("Messages() = %v, want [%q]", msgs, "email already exists")
		}
	})

	t.Run("composing multiple sub-struct validations", func(t *testing.T) {
		t.Parallel()
		// Simulates three self-validating sub-structs.
		err1 := validation.NewAccumulator().Check(false, "a").Result()               // *BadRequestErr
		err2 := validation.NewAccumulator().Check(false, "b").Check(false, "c").Result() // *CompositeErr

		result := validation.NewAccumulator().
			MergeErr(err1).
			MergeErr(err2).
			Result()

		var ce *domainerr.CompositeErr
		if !errors.As(result, &ce) {
			t.Fatalf("Result() type = %T, want *domainerr.CompositeErr", result)
		}
		if got := len(ce.Messages()); got != 3 {
			t.Errorf("Messages() len = %d, want 3 (a, b, c)", got)
		}
	})

	t.Run("dedup across multiple MergeErr calls", func(t *testing.T) {
		t.Parallel()
		a := validation.NewAccumulator()
		a.MergeErr(domainerr.NewBadRequest("duplicate"))
		a.MergeErr(domainerr.NewBadRequest("duplicate"))

		if got := len(a.Messages()); got != 1 {
			t.Errorf("Messages() len = %d, want 1 (dedup)", got)
		}
	})

	t.Run("returns receiver for chaining", func(t *testing.T) {
		t.Parallel()
		a := validation.NewAccumulator()
		if got := a.MergeErr(nil); got != a {
			t.Error("MergeErr did not return the receiver")
		}
	})
}
