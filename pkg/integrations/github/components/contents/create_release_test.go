package contents

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__CreateRelease__IncrementVersion(t *testing.T) {
	component := CreateRelease{}

	t.Run("patch increment works correctly", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{"simple version", "1.2.3", "1.2.4"},
			{"with v prefix", "v1.2.3", "v1.2.4"},
			{"with version prefix", "version-1.2.3", "version-1.2.4"},
			{"patch rollover", "1.2.9", "1.2.10"},
			{"double digit patch", "1.2.99", "1.2.100"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := component.incrementVersion(tc.input, "patch")
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("minor increment works correctly", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{"simple version", "1.2.3", "1.3.0"},
			{"with v prefix", "v1.2.3", "v1.3.0"},
			{"with version prefix", "version-1.2.3", "version-1.3.0"},
			{"minor rollover", "1.9.3", "1.10.0"},
			{"resets patch", "1.2.99", "1.3.0"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := component.incrementVersion(tc.input, "minor")
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("major increment works correctly", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{"simple version", "1.2.3", "2.0.0"},
			{"with v prefix", "v1.2.3", "v2.0.0"},
			{"with version prefix", "version-1.2.3", "version-2.0.0"},
			{"major rollover", "9.2.3", "10.0.0"},
			{"resets minor and patch", "1.99.99", "2.0.0"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := component.incrementVersion(tc.input, "major")
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("invalid version format returns error", func(t *testing.T) {
		testCases := []struct {
			name    string
			version string
		}{
			{"no dots", "123"},
			{"only major.minor", "1.2"},
			{"non-numeric", "v1.2.x"},
			{"empty string", ""},
			{"only prefix", "v"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := component.incrementVersion(tc.version, "patch")
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid version format")
			})
		}
	})

	t.Run("version with extra parts extracts first three", func(t *testing.T) {
		// The regex extracts the first major.minor.patch it finds
		result, err := component.incrementVersion("1.2.3.4", "patch")
		require.NoError(t, err)
		assert.Equal(t, "1.2.4", result)
	})

	t.Run("invalid strategy returns error", func(t *testing.T) {
		_, err := component.incrementVersion("v1.2.3", "invalid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid version strategy")
	})
}

func Test__CreateRelease__SemverRegex(t *testing.T) {
	t.Run("valid semantic versions match", func(t *testing.T) {
		testCases := []struct {
			name    string
			version string
		}{
			{"simple version", "1.2.3"},
			{"with v prefix", "v1.2.3"},
			{"with version prefix", "version-1.2.3"},
			{"double digits", "10.20.30"},
			{"triple digits", "100.200.300"},
			{"with release prefix", "release-1.2.3"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				assert.True(t, semverRegex.MatchString(tc.version))
			})
		}
	})

	t.Run("invalid formats do not match", func(t *testing.T) {
		testCases := []struct {
			name    string
			version string
		}{
			{"no dots", "123"},
			{"only major.minor", "1.2"},
			{"non-numeric major", "vx.2.3"},
			{"non-numeric minor", "v1.x.3"},
			{"non-numeric patch", "v1.2.x"},
			{"empty string", ""},
			{"only prefix", "v"},
			{"spaces", "1. 2.3"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				assert.False(t, semverRegex.MatchString(tc.version))
			})
		}
	})

	t.Run("versions with extra parts match first three", func(t *testing.T) {
		// The regex extracts the first major.minor.patch, ignoring extra parts
		assert.True(t, semverRegex.MatchString("1.2.3.4"))
		assert.True(t, semverRegex.MatchString("v10.20.30.40"))
	})
}
