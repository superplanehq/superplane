package statuspage

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test_extractComponentIDs(t *testing.T) {
	t.Run("empty config returns nil", func(t *testing.T) {
		assert.Nil(t, extractComponentIDs(map[string]any{}))
		assert.Nil(t, extractComponentIDs(map[string]any{"components": nil}))
	})

	t.Run("valid components list extracts IDs", func(t *testing.T) {
		config := map[string]any{
			"components": []any{
				map[string]any{"componentId": "comp1", "status": "degraded_performance"},
				map[string]any{"componentId": "comp2", "status": "operational"},
			},
		}
		ids := extractComponentIDs(config)
		assert.Equal(t, []string{"comp1", "comp2"}, ids)
	})

	t.Run("skips items with empty componentId", func(t *testing.T) {
		config := map[string]any{
			"components": []any{
				map[string]any{"componentId": "comp1", "status": "degraded_performance"},
				map[string]any{"componentId": "", "status": "operational"},
				map[string]any{"status": "operational"},
				map[string]any{"componentId": "comp2", "status": "operational"},
			},
		}
		ids := extractComponentIDs(config)
		assert.Equal(t, []string{"comp1", "comp2"}, ids)
	})

	t.Run("handles malformed items", func(t *testing.T) {
		config := map[string]any{
			"components": []any{
				"not an object",
				123,
				map[string]any{"componentId": "comp1", "status": "degraded_performance"},
			},
		}
		ids := extractComponentIDs(config)
		assert.Equal(t, []string{"comp1"}, ids)
	})

	t.Run("non-list returns nil", func(t *testing.T) {
		assert.Nil(t, extractComponentIDs(map[string]any{"components": "not a list"}))
		assert.Nil(t, extractComponentIDs(map[string]any{"components": 123}))
	})
}

func Test_containsExpression(t *testing.T) {
	assert.False(t, containsExpression(nil))
	assert.False(t, containsExpression([]string{}))
	assert.False(t, containsExpression([]string{"comp1", "comp2"}))
	assert.True(t, containsExpression([]string{"comp1", "{{ $['X'].data.id }}"}))
	assert.True(t, containsExpression([]string{"{{ expression }}"}))
}

func Test_resolveMetadataSetup(t *testing.T) {
	t.Run("skips verification when HTTP is nil", func(t *testing.T) {
		metadata, err := resolveMetadataSetup(core.SetupContext{
			Configuration: map[string]any{},
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "key"}},
		}, "page123", nil)
		require.NoError(t, err)
		assert.Equal(t, NodeMetadata{}, metadata)
	})

	t.Run("skips verification when pageID contains expression", func(t *testing.T) {
		metadata, err := resolveMetadataSetup(core.SetupContext{
			Configuration: map[string]any{},
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "key"}},
		}, "{{ $['X'].data.page_id }}", nil)
		require.NoError(t, err)
		assert.Equal(t, NodeMetadata{}, metadata)
	})

	t.Run("returns error when page not found", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"id":"other-page","name":"Other"}]`))},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		metadata, err := resolveMetadataSetup(core.SetupContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		}, "nonexistent-page", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "page \"nonexistent-page\" not found or not accessible")
		assert.Equal(t, NodeMetadata{}, metadata)
	})

	t.Run("returns metadata when page found", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"id":"kctbh9vrtdwd","name":"My Status Page"}]`))},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		metadata, err := resolveMetadataSetup(core.SetupContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		}, "kctbh9vrtdwd", nil)
		require.NoError(t, err)
		assert.Equal(t, "My Status Page", metadata.PageName)
	})
}

func Test_toUTCISO8601(t *testing.T) {
	t.Run("Z suffix means UTC regardless of timezone param", func(t *testing.T) {
		// "2026-02-15T02:00:00Z" is 02:00 UTC. Must not be re-interpreted as 02:00 America/New_York.
		out, err := toUTCISO8601("2026-02-15T02:00:00Z", "America/New_York")
		require.NoError(t, err)
		assert.Equal(t, "2026-02-15T02:00:00Z", out)
	})

	t.Run("no Z suffix interpreted in given timezone", func(t *testing.T) {
		// "2026-02-15T02:00" in America/New_York (EST, UTC-5) = 07:00 UTC
		out, err := toUTCISO8601("2026-02-15T02:00", "America/New_York")
		require.NoError(t, err)
		assert.Equal(t, "2026-02-15T07:00:00Z", out)
	})

	t.Run("no Z with UTC timezone", func(t *testing.T) {
		out, err := toUTCISO8601("2026-02-15T02:00", "UTC")
		require.NoError(t, err)
		assert.Equal(t, "2026-02-15T02:00:00Z", out)
	})
}
