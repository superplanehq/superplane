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

var putObjectConfig = map[string]any{
	"bucket":   "fra1/my-space",
	"filePath": "reports/daily.csv",
	"body":     "date,value\n2026-01-01,100",
}

func Test__PutObject__Setup(t *testing.T) {
	component := &PutObject{}

	t.Run("missing spaces access key returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: putObjectConfig,
			Metadata:      &contexts.MetadataContext{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{},
			},
		})
		require.ErrorContains(t, err, "spaces access key is missing")
	})

	t.Run("missing spaces secret key returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: putObjectConfig,
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
				"body":     "hello",
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
				"body":     "hello",
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
				"body":   "hello",
			},
			Metadata:    &contexts.MetadataContext{},
			Integration: spacesIntegration,
		})
		require.ErrorContains(t, err, "filePath is required")
	})

	t.Run("missing body returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"bucket":   "fra1/my-space",
				"filePath": "reports/daily.csv",
			},
			Metadata:    &contexts.MetadataContext{},
			Integration: spacesIntegration,
		})
		require.ErrorContains(t, err, "body is required")
	})

	t.Run("valid configuration returns no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: putObjectConfig,
			Metadata:      &contexts.MetadataContext{},
			Integration:   spacesIntegration,
		})
		require.NoError(t, err)
	})
}

func Test__PutObject__Execute(t *testing.T) {
	component := &PutObject{}

	t.Run("successful upload emits output with detected content type", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Etag": []string{`"a1b2c3d4ef567890"`},
					},
					Body: io.NopCloser(strings.NewReader("")),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  putObjectConfig,
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    spacesIntegration,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, "digitalocean.spaces.object.uploaded", executionState.Type)
		assert.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		payload, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)

		assert.Equal(t, "my-space", payload["bucket"])
		assert.Equal(t, "reports/daily.csv", payload["filePath"])
		assert.Equal(t, "https://my-space.fra1.digitaloceanspaces.com/reports/daily.csv", payload["endpoint"])
		assert.Equal(t, "a1b2c3d4ef567890", payload["eTag"])
		assert.Equal(t, "text/csv", payload["contentType"])
		assert.NotEmpty(t, payload["size"])
		assert.Nil(t, payload["metadata"])
		assert.Nil(t, payload["tags"])
	})

	t.Run("upload with metadata and tags includes them in output", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Etag": []string{`"abc123"`},
					},
					Body: io.NopCloser(strings.NewReader("")),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"bucket":   "fra1/my-space",
				"filePath": "config.json",
				"body":     `{"env": "production"}`,
				"acl":      "public-read",
				"metadata": map[string]any{"uploaded-by": "pipeline"},
				"tags":     map[string]any{"env": "production"},
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    spacesIntegration,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		payload, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)

		assert.Equal(t, "application/json", payload["contentType"])

		metadata, ok := payload["metadata"].(map[string]string)
		require.True(t, ok)
		assert.Equal(t, "pipeline", metadata["uploaded-by"])

		tags, ok := payload["tags"].(map[string]string)
		require.True(t, ok)
		assert.Equal(t, "production", tags["env"])
	})

	t.Run("unknown extension falls back to application/octet-stream", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Etag": []string{`"abc123"`},
					},
					Body: io.NopCloser(strings.NewReader("")),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"bucket":   "fra1/my-space",
				"filePath": "Dockerfile",
				"body":     "FROM node:18",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    spacesIntegration,
		})

		require.NoError(t, err)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		payload, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "application/octet-stream", payload["contentType"])
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
			Configuration:  putObjectConfig,
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    spacesIntegration,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to put object")
		assert.False(t, executionState.Passed)
	})
}
