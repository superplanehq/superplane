package digitalocean

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

var deleteObjectConfig = map[string]any{
	"bucket":   "fra1/my-space",
	"filePath": "reports/daily.csv",
}

func Test__DeleteObject__Setup(t *testing.T) {
	component := &DeleteObject{}

	t.Run("missing spaces access key returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: deleteObjectConfig,
			Metadata:      &contexts.MetadataContext{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{},
			},
		})
		require.ErrorContains(t, err, "spaces access key is missing")
	})

	t.Run("missing spaces secret key returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: deleteObjectConfig,
			Metadata:      &contexts.MetadataContext{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"spacesAccessKey": "test-access-key",
				},
			},
		})
		require.ErrorContains(t, err, "spaces secret key is missing")
	})

	t.Run("missing bucket returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"filePath": "reports/daily.csv",
			},
			Metadata:    &contexts.MetadataContext{},
			Integration: spacesIntegration,
		})
		require.ErrorContains(t, err, "bucket is required")
	})

	t.Run("invalid bucket format returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"bucket":   "my-space",
				"filePath": "reports/daily.csv",
			},
			Metadata:    &contexts.MetadataContext{},
			Integration: spacesIntegration,
		})
		require.ErrorContains(t, err, "invalid bucket value")
	})

	t.Run("missing filePath returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"bucket": "fra1/my-space",
			},
			Metadata:    &contexts.MetadataContext{},
			Integration: spacesIntegration,
		})
		require.ErrorContains(t, err, "filePath is required")
	})

	t.Run("valid configuration returns no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: deleteObjectConfig,
			Metadata:      &contexts.MetadataContext{},
			Integration:   spacesIntegration,
		})
		require.NoError(t, err)
	})
}

func Test__DeleteObject__Execute(t *testing.T) {
	component := &DeleteObject{}

	t.Run("successful delete emits output", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Header:     http.Header{},
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  deleteObjectConfig,
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    spacesIntegration,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, "digitalocean.spaces.object.deleted", executionState.Type)
		assert.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		payload, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)

		assert.Equal(t, "my-space", payload["bucket"])
		assert.Equal(t, "reports/daily.csv", payload["filePath"])
		assert.Equal(t, true, payload["deleted"])
	})

	t.Run("delete non-existent object succeeds (idempotent)", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Header:     http.Header{},
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  deleteObjectConfig,
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    spacesIntegration,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusForbidden,
					Header:     http.Header{},
					Body:       io.NopCloser(strings.NewReader(`<?xml version="1.0"?><Error><Code>AccessDenied</Code></Error>`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  deleteObjectConfig,
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    spacesIntegration,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete object")
		assert.False(t, executionState.Passed)
	})
}
