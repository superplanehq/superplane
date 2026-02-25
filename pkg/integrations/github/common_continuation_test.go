package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildPRContinuationKey(t *testing.T) {
	t.Run("builds key from pull_request payload", func(t *testing.T) {
		key, ok := buildPRContinuationKey(map[string]any{
			"repository": map[string]any{"full_name": "acme/service"},
			"pull_request": map[string]any{
				"number": float64(17),
			},
		})

		assert.True(t, ok)
		assert.Equal(t, "github:acme/service:pr:17", key)
	})

	t.Run("builds key from issue payload", func(t *testing.T) {
		key, ok := buildPRContinuationKey(map[string]any{
			"repository": map[string]any{
				"owner": map[string]any{"login": "acme"},
				"name":  "service",
			},
			"issue": map[string]any{
				"number": float64(44),
			},
		})

		assert.True(t, ok)
		assert.Equal(t, "github:acme/service:pr:44", key)
	})

	t.Run("returns false without repository", func(t *testing.T) {
		_, ok := buildPRContinuationKey(map[string]any{
			"pull_request": map[string]any{"number": float64(9)},
		})

		assert.False(t, ok)
	})
}
