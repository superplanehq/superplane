package crypto

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

const (
	MinPasswordLength = 8
	PasswordRulesText = "Password must be at least 8 characters and include uppercase, lowercase, a number, and a symbol."
)

// ValidatePassword checks that a password meets the minimum complexity
// requirements: at least 8 characters, one uppercase letter, one lowercase
// letter, one digit, and one symbol.
func ValidatePassword(password string) error {
	var reasons []string

	if len(password) < MinPasswordLength {
		reasons = append(reasons, fmt.Sprintf("at least %d characters", MinPasswordLength))
	}

	var hasUpper, hasLower, hasDigit, hasSymbol bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSymbol = true
		}
	}

	if !hasUpper {
		reasons = append(reasons, "an uppercase letter")
	}
	if !hasLower {
		reasons = append(reasons, "a lowercase letter")
	}
	if !hasDigit {
		reasons = append(reasons, "a number")
	}
	if !hasSymbol {
		reasons = append(reasons, "a symbol")
	}

	if len(reasons) > 0 {
		return fmt.Errorf("Password must include %s.", strings.Join(reasons, ", "))
	}

	return nil
}

// HashPassword hashes a plaintext password using bcrypt
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword verifies a plaintext password against a bcrypt hash
func VerifyPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
