package models

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeContentBroadcast(t *testing.T) {
	t.Run("requires a summary", func(t *testing.T) {
		_, err := normalizeContentBroadcast(FactoryWorkOrderContentBroadcastParams{
			Body: "details",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrFactoryWorkOrderBroadcastInvalid)
		assert.ErrorContains(t, err, "summary is required")
	})

	t.Run("requires content or a link", func(t *testing.T) {
		_, err := normalizeContentBroadcast(FactoryWorkOrderContentBroadcastParams{
			Summary: "Preview environment is ready",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrFactoryWorkOrderBroadcastInvalid)
		assert.ErrorContains(t, err, "content or link URL is required")
	})

	t.Run("rejects an unsafe link", func(t *testing.T) {
		_, err := normalizeContentBroadcast(FactoryWorkOrderContentBroadcastParams{
			Summary: "Preview environment is ready",
			URL:     "javascript:alert(1)",
		})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrFactoryWorkOrderBroadcastInvalid))
	})

	t.Run("keeps a preview link and drops a label when the URL is empty", func(t *testing.T) {
		normalized, err := normalizeContentBroadcast(FactoryWorkOrderContentBroadcastParams{
			Summary:  "  Preview environment is ready  ",
			Body:     "  Open the environment from the activity log.  ",
			URLLabel: "Preview",
		})
		require.NoError(t, err)
		assert.Equal(t, "Preview environment is ready", normalized.Summary)
		assert.Equal(t, "Open the environment from the activity log.", normalized.Body)
		assert.Empty(t, normalized.URL)
		assert.Empty(t, normalized.URLLabel)
	})

	t.Run("accepts a preview URL", func(t *testing.T) {
		normalized, err := normalizeContentBroadcast(FactoryWorkOrderContentBroadcastParams{
			Summary:  "Preview environment is ready",
			URL:      "  https://preview.example.com/orders/12  ",
			URLLabel: " Preview ",
		})
		require.NoError(t, err)
		assert.Equal(t, "https://preview.example.com/orders/12", normalized.URL)
		assert.Equal(t, "Preview", normalized.URLLabel)
	})

	t.Run("rejects an oversized summary", func(t *testing.T) {
		_, err := normalizeContentBroadcast(FactoryWorkOrderContentBroadcastParams{
			Summary: strings.Repeat("a", MaxFactoryWorkOrderBroadcastSummaryBytes+1),
			Body:    "details",
		})
		require.ErrorIs(t, err, ErrFactoryWorkOrderBroadcastInvalid)
	})
}
