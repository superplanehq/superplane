package diagnostics

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeFailureText(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		values     []string
	}{
		{name: "environment value", text: "rejected private value", values: []string{"private value"}},
		{name: "escaped environment value", text: `{"error":"rejected private\"value"}`, values: []string{`private"value`}},
		{name: "authorization", text: "Bearer private-value"},
		{name: "basic authorization", text: "Basic private-value"},
		{name: "URL credentials", text: "https://user:private-value@example.com/path"},
		{name: "query credentials", text: "https://example.com/?token=private-value&ok=true"},
		{name: "JSON credentials", text: `{"api_key":"private-value"}`},
		{name: "assignment", text: "GITHUB_TOKEN=private-value"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Sanitize(tc.text, tc.values, 4096)
			assert.NotContains(t, result, "private")
			assert.Contains(t, result, "[REDACTED]")
		})
	}
	t.Run("redact before truncation", func(t *testing.T) {
		text := strings.Repeat("a", 70) + "private-value" + strings.Repeat("é", 100)
		result := Sanitize(text, []string{"private-value"}, 80)
		assert.LessOrEqual(t, len(result), 80)
		assert.True(t, utf8.ValidString(result))
		assert.NotContains(t, result, "private")
		assert.Contains(t, result, "[truncated]")
	})
}

func TestSanitizeURLQueryCredentials(t *testing.T) {
	for _, tc := range []struct {
		name, text, expected string
	}{
		{
			name:     "key in first parameter",
			text:     "HTTP 401: https://example.com/check?key=private-value&status=401#details",
			expected: "HTTP 401: https://example.com/check?key=[REDACTED]&status=401#details",
		},
		{
			name:     "key in later parameter",
			text:     "https://example.com/check?status=401&key=private-value",
			expected: "https://example.com/check?status=401&key=[REDACTED]",
		},
		{
			name:     "case insensitive key",
			text:     "https://example.com/check?KEY=private-value",
			expected: "https://example.com/check?KEY=[REDACTED]",
		},
		{
			name:     "access key",
			text:     "https://example.com/check?access_key=private-value",
			expected: "https://example.com/check?access_key=[REDACTED]",
		},
		{
			name:     "repeated keys",
			text:     "https://example.com/check?key=first-private-value&key=second-private-value&status=401",
			expected: "https://example.com/check?key=[REDACTED]&key=[REDACTED]&status=401",
		},
		{
			name:     "encoded key name and value",
			text:     "https://example.com/check?%6b%65%79=private%2Dvalue&status=401",
			expected: "https://example.com/check?%6b%65%79=[REDACTED]&status=401",
		},
		{
			name:     "encoded token name",
			text:     "https://example.com/check?%74oken=private-value&status=401",
			expected: "https://example.com/check?%74oken=[REDACTED]&status=401",
		},
		{
			name:     "ordinary parameters and settings",
			text:     "HTTP 401 after 3600 seconds: key=lookup https://example.com/check?status=401&timeout=3600",
			expected: "HTTP 401 after 3600 seconds: key=lookup https://example.com/check?status=401&timeout=3600",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, Sanitize(tc.text, nil, 4096))
		})
	}
}
