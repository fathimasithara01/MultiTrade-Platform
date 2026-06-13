package idempotency

import (
	"errors"
)

var (
	ErrInvalidKey = errors.New("invalid idempotency key format")
)

func Validate(key string) error {
	if key == "" {
		return errors.New("idempotency key is required")
	}
	if len(key) < 8 || len(key) > 64 {
		return ErrInvalidKey
	}
	return nil
}
