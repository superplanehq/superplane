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
