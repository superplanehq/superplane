package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeEmail(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
		name     string
	}{
		{
			input:    "John@Example.com",
			expected: "john@example.com",
			name:     "mixed case email",
		},
		{
			input:    "ADMIN@COMPANY.ORG",
			expected: "admin@company.org",
			name:     "uppercase email",
		},
		{
			input:    "user@domain.com",
			expected: "user@domain.com",
			name:     "lowercase email",
		},
		{
			input:    "  test@example.com  ",
			expected: "test@example.com",
			name:     "email with whitespace",
		},
		{
			input:    "",
			expected: "",
			name:     "empty email",
		},
		{
			input:    "   ",
			expected: "",
			name:     "whitespace only",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeEmail(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}