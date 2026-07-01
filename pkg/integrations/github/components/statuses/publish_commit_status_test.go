package statuses

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__PublishCommitStatus__ValidateSHA(t *testing.T) {
	t.Run("valid 40-char hex SHA passes", func(t *testing.T) {
		validSHA := "abc123def456789012345678901234567890abcd"
		assert.True(t, shaRegex.MatchString(validSHA))
	})

	t.Run("invalid SHA formats fail", func(t *testing.T) {
		testCases := []struct {
			name string
			sha  string
		}{
			{"too short", "abc123"},
			{"too long", "abc123def456789012345678901234567890abcdef"},
			{"uppercase letters", "ABC123DEF456789012345678901234567890ABCD"},
			{"invalid characters", "xyz123def456789012345678901234567890abcd"},
			{"with spaces", "abc123 ef456789012345678901234567890abcd"},
			{"empty string", ""},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				assert.False(t, shaRegex.MatchString(tc.sha))
			})
		}
	})
}
