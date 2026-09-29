package models

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeActivityBroadcast(t *testing.T) {
	t.Run("requires a title", func(t *testing.T) {
		_, err := normalizeActivityBroadcast(FactoryWorkOrderActivityBroadcastParams{
			Body: "Preview is ready.",
		})
		assert.ErrorIs(t, err, ErrFactoryWorkOrderActivityBroadcastInvalid)
	})

	t.Run("requires body or url", func(t *testing.T) {
		_, err := normalizeActivityBroadcast(FactoryWorkOrderActivityBroadcastParams{
			Title: "Preview environment ready",
		})
		assert.ErrorIs(t, err, ErrFactoryWorkOrderActivityBroadcastInvalid)
	})

	t.Run("rejects a relative url", func(t *testing.T) {
		_, err := normalizeActivityBroadcast(FactoryWorkOrderActivityBroadcastParams{
			Title: "Preview environment ready",
			URL:   "/preview/42",
		})
		assert.ErrorIs(t, err, ErrFactoryWorkOrderActivityBroadcastInvalid)
	})

	t.Run("rejects a non-http url", func(t *testing.T) {
		_, err := normalizeActivityBroadcast(FactoryWorkOrderActivityBroadcastParams{
			Title: "Preview environment ready",
			URL:   "javascript:alert(1)",
		})
		assert.ErrorIs(t, err, ErrFactoryWorkOrderActivityBroadcastInvalid)
	})

	t.Run("keeps title body and url", func(t *testing.T) {
		broadcast, err := normalizeActivityBroadcast(FactoryWorkOrderActivityBroadcastParams{
			Title: "  Preview environment ready  ",
			Body:  " Open the preview. ",
			URL:   " https://preview.example.com/pr/42 ",
		})
		require.NoError(t, err)
		assert.Equal(t, "Preview environment ready", broadcast.Title)
		assert.Equal(t, "Open the preview.", broadcast.Body)
		assert.Equal(t, "https://preview.example.com/pr/42", broadcast.URL)
	})

	t.Run("rejects an oversized title", func(t *testing.T) {
		_, err := normalizeActivityBroadcast(FactoryWorkOrderActivityBroadcastParams{
			Title: strings.Repeat("a", MaxFactoryWorkOrderActivityBroadcastTitleBytes+1),
			URL:   "https://preview.example.com",
		})
		assert.ErrorIs(t, err, ErrFactoryWorkOrderActivityBroadcastInvalid)
	})
}
