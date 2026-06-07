package is_test

import (
	"testing"

	"github.com/Lucas-Lopes-II/govalidator/is"
)

func TestUUID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "valid v4 UUID", in: "550e8400-e29b-41d4-a716-446655440000", want: true},
		{name: "valid v1 UUID", in: "6ba7b810-9dad-11d1-80b4-00c04fd430c8", want: true},
		{name: "valid nil UUID", in: "00000000-0000-0000-0000-000000000000", want: true},
		{name: "empty string", in: "", want: false},
		{name: "random text", in: "not-a-uuid", want: false},
		// google/uuid accepts the 32-hex-char form without dashes per its own spec.
		{name: "UUID without dashes (accepted by library)", in: "550e8400e29b41d4a716446655440000", want: true},
		{name: "partial UUID", in: "550e8400-e29b-41d4", want: false},
		{name: "UUID with extra chars", in: "550e8400-e29b-41d4-a716-44665544000X", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := is.UUID(tt.in); got != tt.want {
				t.Errorf("UUID(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "simple valid email", in: "user@example.com", want: true},
		{name: "subdomain", in: "user@mail.example.com", want: true},
		{name: "plus addressing", in: "user+tag@example.com", want: true},
		{name: "hyphen in domain", in: "user@my-domain.com", want: true},
		{name: "numeric TLD (2 chars)", in: "user@example.io", want: true},
		{name: "empty string", in: "", want: false},
		{name: "missing @", in: "userexample.com", want: false},
		{name: "missing domain", in: "user@", want: false},
		{name: "missing TLD", in: "user@example", want: false},
		{name: "space before @", in: "user @example.com", want: false},
		{name: "double @", in: "user@@example.com", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := is.Email(tt.in); got != tt.want {
				t.Errorf("Email(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestISODate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "RFC3339 UTC (Z)", in: "2024-01-25T10:00:00Z", want: true},
		{name: "RFC3339 negative offset", in: "2024-01-25T10:00:00-03:00", want: true},
		{name: "RFC3339 positive offset", in: "2024-01-25T10:00:00+05:30", want: true},
		{name: "RFC3339Nano millis UTC", in: "2024-01-25T10:00:00.929Z", want: true},
		{name: "RFC3339Nano nanos with offset", in: "2024-01-25T10:00:00.123456789-03:00", want: true},
		{name: "empty string", in: "", want: false},
		{name: "date only", in: "2024-01-25", want: false},
		{name: "dd/MM/yyyy", in: "25/01/2024", want: false},
		{name: "no timezone", in: "2024-01-25T10:00:00", want: false},
		{name: "invalid month", in: "2024-13-01T10:00:00Z", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := is.ISODate(tt.in); got != tt.want {
				t.Errorf("ISODate(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestStrongPassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "valid: all requirements met", in: "Abc1@xyz", want: true},
		{name: "valid: exactly 6 chars", in: "Ab1@cd", want: true},
		{name: "valid: unicode upper+lower+digit+special", in: "Ñoño1!", want: true},
		{name: "empty string", in: "", want: false},
		{name: "too short (5 chars)", in: "Ab1@c", want: false},
		{name: "no digit", in: "Abcdef@!", want: false},
		{name: "no special char", in: "Abcdef1", want: false},
		{name: "no uppercase", in: "abc1@xyz", want: false},
		{name: "no lowercase", in: "ABC1@XYZ", want: false},
		{name: "contains newline", in: "Abc1@\nxy", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := is.StrongPassword(tt.in); got != tt.want {
				t.Errorf("StrongPassword(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestLatitude(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   float64
		want bool
	}{
		{name: "zero", in: 0.0, want: true},
		{name: "max (+90)", in: 90.0, want: true},
		{name: "min (-90)", in: -90.0, want: true},
		{name: "São Paulo approx", in: -23.5505, want: true},
		{name: "just above max", in: 90.0001, want: false},
		{name: "just below min", in: -90.0001, want: false},
		{name: "far above", in: 180.0, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := is.Latitude(tt.in); got != tt.want {
				t.Errorf("Latitude(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestLongitude(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   float64
		want bool
	}{
		{name: "zero", in: 0.0, want: true},
		{name: "max (+180)", in: 180.0, want: true},
		{name: "min (-180)", in: -180.0, want: true},
		{name: "São Paulo approx", in: -46.6333, want: true},
		{name: "just above max", in: 180.0001, want: false},
		{name: "just below min", in: -180.0001, want: false},
		{name: "outside latitude range", in: -91.0, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := is.Longitude(tt.in); got != tt.want {
				t.Errorf("Longitude(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
