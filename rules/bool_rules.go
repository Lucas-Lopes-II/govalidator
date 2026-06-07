package rules

import (
	"github.com/Lucas-Lopes-II/govalidator/domainerr"
	"github.com/Lucas-Lopes-II/govalidator/validation"
)

// IsTrue returns a rule that fails when extract(input) != true.
// Useful for "accepted terms", "is active", and similar boolean assertions.
func IsTrue[T any](extract func(T) bool, field, message string) validation.ValidationFunc[T] {
	return func(input T) error {
		if !extract(input) {
			return domainerr.NewBadRequest(message)
		}
		return nil
	}
}

// IsFalse returns a rule that fails when extract(input) != false.
func IsFalse[T any](extract func(T) bool, field, message string) validation.ValidationFunc[T] {
	return func(input T) error {
		if extract(input) {
			return domainerr.NewBadRequest(message)
		}
		return nil
	}
}
