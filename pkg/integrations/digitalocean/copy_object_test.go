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

var copyObjectConfig = map[string]any{
	"sourceBucket":        "fra1/my-space",
	"sourceFilePath":      "reports/daily.csv",
	"destinationBucket":   "fra1/my-archive",
	"destinationFilePath": "archive/2026/daily.csv",
}

func Test__CopyObject__Setup(t *testing.T) {
	component := &CopyObject{}

	t.Run("missing spaces access key returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: copyObjectConfig,
			Metadata:      &contexts.MetadataContext{},
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{}},
		})
		require.ErrorContains(t, err, "spaces access key is missing")
	})

	t.Run("missing spaces secret key returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: copyObjectConfig,
			Metadata:      &contexts.MetadataContext{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"spacesAccessKey": "key"},
			},
		})
		require.ErrorContains(t, err, "spaces secret key is missing")
	})

	t.Run("missing sourceBucket returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"sourceFilePath":      "reports/daily.csv",
				"destinationBucket":   "fra1/my-archive",
				"destinationFilePath": "archive/2026/daily.csv",
			},
			Metadata:    &contexts.MetadataContext{},
			Integration: spacesIntegration,
		})
		require.ErrorContains(t, err, "sourceBucket is required")
	})

	t.Run("invalid sourceBucket format returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"sourceBucket":        "my-space",
				"sourceFilePath":      "reports/daily.csv",
				"destinationBucket":   "fra1/my-archive",
				"destinationFilePath": "archive/2026/daily.csv",
			},
			Metadata:    &contexts.MetadataContext{},
			Integration: spacesIntegration,
		})
		require.ErrorContains(t, err, "invalid sourceBucket")
	})

	t.Run("missing sourceFilePath returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"sourceBucket":        "fra1/my-space",
				"destinationBucket":   "fra1/my-archive",
				"destinationFilePath": "archive/2026/daily.csv",
			},
			Metadata:    &contexts.MetadataContext{},
			Integration: spacesIntegration,
		})
		require.ErrorContains(t, err, "sourceFilePath is required")
	})

	t.Run("missing destinationBucket returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"sourceBucket":        "fra1/my-space",
				"sourceFilePath":      "reports/daily.csv",
				"destinationFilePath": "archive/2026/daily.csv",
			},
			Metadata:    &contexts.MetadataContext{},
			Integration: spacesIntegration,
		})
		require.ErrorContains(t, err, "destinationBucket is required")
	})

	t.Run("missing destinationFilePath returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"sourceBucket":      "fra1/my-space",
				"sourceFilePath":    "reports/daily.csv",
				"destinationBucket": "fra1/my-archive",
			},
			Metadata:    &contexts.MetadataContext{},
			Integration: spacesIntegration,
		})
		require.ErrorContains(t, err, "destinationFilePath is required")
	})

	t.Run("valid configuration returns no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: copyObjectConfig,
			Metadata:      &contexts.MetadataContext{},
			Integration:   spacesIntegration,
		})
		require.NoError(t, err)
	})
}

func Test__CopyObject__Execute(t *testing.T) {
	component := &CopyObject{}

	copyResponseBody := `<?xml version="1.0" encoding="UTF-8"?>
<CopyObjectResult>
  <ETag>"a1b2c3d4ef567890a1b2c3d4ef567890"</ETag>
  <LastModified>2026-03-25T09:00:00.000Z</LastModified>
</CopyObjectResult>`

	newCopyResponse := func() *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader(copyResponseBody)),
		}
	}

	t.Run("successful copy emits output with moved=false", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{newCopyResponse()},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  copyObjectConfig,
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    spacesIntegration,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "digitalocean.spaces.object.copied", executionState.Type)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		payload, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)

		assert.Equal(t, "my-space", payload["sourceBucket"])
		assert.Equal(t, "reports/daily.csv", payload["sourceFilePath"])
		assert.Equal(t, "my-archive", payload["destinationBucket"])
		assert.Equal(t, "archive/2026/daily.csv", payload["destinationFilePath"])
		assert.Equal(t, "https://my-archive.fra1.digitaloceanspaces.com/archive/2026/daily.csv", payload["endpoint"])
		assert.Equal(t, "a1b2c3d4ef567890a1b2c3d4ef567890", payload["eTag"])
		assert.Equal(t, false, payload["moved"])
	})

	t.Run("copy with deleteSource=true emits output with moved=true", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				newCopyResponse(),
				{
					StatusCode: http.StatusNoContent,
					Header:     http.Header{},
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		config := map[string]any{
			"sourceBucket":        "fra1/my-space",
			"sourceFilePath":      "reports/daily.csv",
			"destinationBucket":   "fra1/my-archive",
			"destinationFilePath": "archive/2026/daily.csv",
			"deleteSource":        true,
		}

		err := component.Execute(core.ExecutionContext{
			Configuration:  config,
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

		assert.Equal(t, "a1b2c3d4ef567890a1b2c3d4ef567890", payload["eTag"])
		assert.Equal(t, true, payload["moved"])
	})

	t.Run("copy API error returns error", func(t *testing.T) {
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
			Configuration:  copyObjectConfig,
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    spacesIntegration,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to copy object")
	})

	t.Run("copy returns 200 with <Error> XML and deleteSource=true does not delete source", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Header:     http.Header{},
					Body:       io.NopCloser(strings.NewReader(`<?xml version="1.0"?><Error><Code>AccessDenied</Code><Message>Denied</Message></Error>`)),
				},
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		config := map[string]any{
			"sourceBucket":        "fra1/my-space",
			"sourceFilePath":      "reports/daily.csv",
			"destinationBucket":   "fra1/my-archive",
			"destinationFilePath": "archive/2026/daily.csv",
			"deleteSource":        true,
		}

		err := component.Execute(core.ExecutionContext{
			Configuration:  config,
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    spacesIntegration,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to copy object")
		assert.Len(t, httpContext.Requests, 1)
		assert.False(t, executionState.Passed)
	})

	t.Run("delete source error after successful copy returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				newCopyResponse(),
				{
					StatusCode: http.StatusForbidden,
					Header:     http.Header{},
					Body:       io.NopCloser(strings.NewReader(`<?xml version="1.0"?><Error><Code>AccessDenied</Code></Error>`)),
				},
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		config := map[string]any{
			"sourceBucket":        "fra1/my-space",
			"sourceFilePath":      "reports/daily.csv",
			"destinationBucket":   "fra1/my-archive",
			"destinationFilePath": "archive/2026/daily.csv",
			"deleteSource":        true,
		}

		err := component.Execute(core.ExecutionContext{
			Configuration:  config,
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    spacesIntegration,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete source")
	})
}
