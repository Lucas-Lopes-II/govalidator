package rules

import (
	"github.com/Lucas-Lopes-II/govalidator/domainerr"
	"github.com/Lucas-Lopes-II/govalidator/validation"
)

// Number is the type constraint for numeric fields.
// Defined locally to avoid the golang.org/x/exp/constraints dependency.
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// MinValue returns a rule that fails when extract(input) < min.
func MinValue[T any, N Number](extract func(T) N, min N, field, message string) validation.ValidationFunc[T] {
	return func(input T) error {
		if extract(input) < min {
			return domainerr.NewBadRequest(message)
		}
		return nil
	}
}

// MaxValue returns a rule that fails when extract(input) > max.
func MaxValue[T any, N Number](extract func(T) N, max N, field, message string) validation.ValidationFunc[T] {
	return func(input T) error {
		if extract(input) > max {
			return domainerr.NewBadRequest(message)
		}
		return nil
	}
}
