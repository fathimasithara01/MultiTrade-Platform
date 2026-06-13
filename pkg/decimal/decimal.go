// Package decimal provides shared high-precision decimal arithmetic used
// throughout the platform (wallets, orders, trades, portfolios).
//
// All monetary values are stored as NUMERIC(20,8) in Postgres and transported
// as strings to preserve precision. This package centralises the parsing and
// formatting so every domain can rely on the same behaviour.
package decimal

import (
	"errors"
	"fmt"
	"math/big"
)

const (
	// Precision is the number of bits used for big.Float operations.
	Precision = 128
	// Scale is the number of decimal places in formatted output.
	Scale = 8
)

var (
	// ErrInvalidDecimal is returned when a string cannot be parsed as a positive decimal.
	ErrInvalidDecimal = errors.New("invalid decimal value")
	// ErrNonPositive is returned when a value is zero or negative.
	ErrNonPositive = errors.New("value must be greater than zero")
	// ErrNegativeNotAllowed is returned when a negative value is used where only >= 0 is valid.
	ErrNegativeNotAllowed = errors.New("value must not be negative")
)

// Parse parses a decimal string into a *big.Float.
// Returns ErrInvalidDecimal if parsing fails.
func Parse(s string) (*big.Float, error) {
	if s == "" {
		return nil, ErrInvalidDecimal
	}
	f, ok := new(big.Float).SetPrec(Precision).SetString(s)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrInvalidDecimal, s)
	}
	return f, nil
}

// ParsePositive parses a decimal string and enforces value > 0.
func ParsePositive(s string) (*big.Float, error) {
	f, err := Parse(s)
	if err != nil {
		return nil, err
	}
	if f.Cmp(Zero()) <= 0 {
		return nil, ErrNonPositive
	}
	return f, nil
}

// ParseNonNegative parses a decimal string and enforces value >= 0.
func ParseNonNegative(s string) (*big.Float, error) {
	f, err := Parse(s)
	if err != nil {
		return nil, err
	}
	if f.Sign() < 0 {
		return nil, ErrNegativeNotAllowed
	}
	return f, nil
}

// MustParse parses a DB-sourced NUMERIC string.
// Panics only if the DB returns malformed data — should never happen in practice.
func MustParse(s string) *big.Float {
	f, err := Parse(s)
	if err != nil {
		panic(fmt.Sprintf("decimal.MustParse: malformed DB value %q: %v", s, err))
	}
	return f
}

// Format serialises a *big.Float to an 8-decimal-place string suitable for
// storing in or comparing with Postgres NUMERIC(20,8) columns.
func Format(f *big.Float) string {
	return fmt.Sprintf("%.8f", f)
}

// Add returns a + b.
func Add(a, b *big.Float) *big.Float {
	return new(big.Float).SetPrec(Precision).Add(a, b)
}

// Sub returns a - b.
func Sub(a, b *big.Float) *big.Float {
	return new(big.Float).SetPrec(Precision).Sub(a, b)
}

// Mul returns a * b.
func Mul(a, b *big.Float) *big.Float {
	return new(big.Float).SetPrec(Precision).Mul(a, b)
}

// Zero returns a new zero-valued big.Float at the standard precision.
func Zero() *big.Float {
	return new(big.Float).SetPrec(Precision)
}

// Min returns the smaller of a and b (does not mutate inputs).
func Min(a, b *big.Float) *big.Float {
	if a.Cmp(b) <= 0 {
		return new(big.Float).SetPrec(Precision).Copy(a)
	}
	return new(big.Float).SetPrec(Precision).Copy(b)
}

// IsPositive returns true when f > 0.
func IsPositive(f *big.Float) bool { return f.Sign() > 0 }

// IsNegative returns true when f < 0.
func IsNegative(f *big.Float) bool { return f.Sign() < 0 }
