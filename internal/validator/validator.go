package validator

import (
	"fmt"
	"regexp"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// Email validation
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	if len(email) > 255 {
		return fmt.Errorf("email must be less than 255 characters")
	}
	return nil
}

// Password validation
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}
	if len(password) > 128 {
		return fmt.Errorf("password must be less than 128 characters")
	}
	if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return fmt.Errorf("password must contain at least one digit")
	}
	return nil
}

// Username validation
func ValidateUsername(username string) error {
	if len(username) < 3 {
		return fmt.Errorf("username must be at least 3 characters long")
	}
	if len(username) > 30 {
		return fmt.Errorf("username must be less than 30 characters")
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(username) {
		return fmt.Errorf("username can only contain letters, numbers, underscores, and hyphens")
	}
	return nil
}

// Amount validation
func ValidateAmount(amount float64, minAmount, maxAmount float64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}
	if amount < minAmount {
		return fmt.Errorf("amount must be at least %f", minAmount)
	}
	if amount > maxAmount {
		return fmt.Errorf("amount must be less than %f", maxAmount)
	}
	return nil
}

// Struct validation
func ValidateStruct(data interface{}) error {
	return validate.Struct(data)
}
