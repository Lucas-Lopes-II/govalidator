package rules

import (
	"github.com/Lucas-Lopes-II/govalidator/domainerr"
	"github.com/Lucas-Lopes-II/govalidator/validation"
)

// OneOf returns a rule that fails when extract(input) is not in allowed.
// Comparison is case-sensitive. The allowed slice is captured at rule-creation
// time — mutations after the call do not affect the rule.
//
// Example:
//
//	rules.OneOf(func(i Input) string { return i.Status }, []string{"active", "inactive"}, "status", "invalid status")
func OneOf[T any](extract func(T) string, allowed []string, field, message string) validation.ValidationFunc[T] {
	set := make(map[string]struct{}, len(allowed))
	for _, v := range allowed {
		set[v] = struct{}{}
	}
	return func(input T) error {
		if _, ok := set[extract(input)]; !ok {
			return domainerr.NewBadRequest(message)
		}
		return nil
	}
}
