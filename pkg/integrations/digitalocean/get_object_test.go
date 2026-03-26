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

// spacesConfig uses the new format: credentials are on the integration,
// bucket encodes the region as "region/bucket-name", objectKey is now filePath.
var spacesConfig = map[string]any{
	"bucket":   "fra1/my-space",
	"filePath": "reports/daily.csv",
}

var spacesIntegration = &contexts.IntegrationContext{
	Configuration: map[string]any{
		"spacesAccessKey": "test-access-key",
		"spacesSecretKey": "test-secret-key",
	},
}

var emptyTaggingResponse = `<Tagging><TagSet/></Tagging>`

var taggingResponse = `
<Tagging>
  <TagSet>
    <Tag><Key>env</Key><Value>production</Value></Tag>
    <Tag><Key>status</Key><Value>ready</Value></Tag>
  </TagSet>
</Tagging>`

func Test__GetObject__Setup(t *testing.T) {
	component := &GetObject{}

	t.Run("missing spaces access key returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: spacesConfig,
			Metadata:      &contexts.MetadataContext{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{},
			},
		})
		require.ErrorContains(t, err, "spaces access key is missing")
	})

	t.Run("missing spaces secret key returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: spacesConfig,
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

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: spacesConfig,
			Metadata:      &contexts.MetadataContext{},
			Integration:   spacesIntegration,
		})
		require.NoError(t, err)
	})
}

func Test__GetObject__Execute(t *testing.T) {
	component := &GetObject{}

	t.Run("object exists, includeBody false -> emits found with metadata and tags", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type":   []string{"text/csv"},
						"Content-Length": []string{"42300"},
						"Last-Modified":  []string{"Wed, 25 Mar 2026 08:45:00 GMT"},
						"Etag":           []string{`"a1b2c3d4ef567890"`},
					},
					Body: io.NopCloser(strings.NewReader("")),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(taggingResponse)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  spacesConfig,
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    spacesIntegration,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, channelFound, executionState.Channel)
		assert.Equal(t, "digitalocean.spaces.object.fetched", executionState.Type)
		assert.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		payload, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "my-space", payload["bucket"])
		assert.Equal(t, "reports/daily.csv", payload["filePath"])
		assert.Equal(t, "https://my-space.fra1.digitaloceanspaces.com/reports/daily.csv", payload["endpoint"])
		assert.Equal(t, "text/csv", payload["contentType"])
		assert.Equal(t, "41.31 KiB", payload["size"])
		assert.Equal(t, "Wed, 25 Mar 2026 08:45:00 GMT", payload["lastModified"])
		assert.Equal(t, "a1b2c3d4ef567890", payload["eTag"])
		assert.Nil(t, payload["body"])

		tags, ok := payload["tags"].(map[string]string)
		require.True(t, ok)
		assert.Equal(t, "production", tags["env"])
		assert.Equal(t, "ready", tags["status"])
	})

	t.Run("object exists with no tags -> emits found with empty tags", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type":   []string{"text/plain"},
						"Content-Length": []string{"13"},
						"Last-Modified":  []string{"Wed, 25 Mar 2026 08:45:00 GMT"},
						"Etag":           []string{`"abc123"`},
					},
					Body: io.NopCloser(strings.NewReader("")),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(emptyTaggingResponse)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  spacesConfig,
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    spacesIntegration,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, channelFound, executionState.Channel)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		payload, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)

		tags, ok := payload["tags"].(map[string]string)
		require.True(t, ok)
		assert.Empty(t, tags)
	})

	t.Run("object exists, includeBody true -> emits found with body", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type":   []string{"text/plain"},
						"Content-Length": []string{"13"},
						"Last-Modified":  []string{"Wed, 25 Mar 2026 08:45:00 GMT"},
						"Etag":           []string{`"abc123"`},
					},
					Body: io.NopCloser(strings.NewReader("hello, world!")),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(emptyTaggingResponse)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"bucket":      "fra1/my-space",
				"filePath":    "hello.txt",
				"includeBody": true,
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    spacesIntegration,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, channelFound, executionState.Channel)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		payload, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "hello, world!", payload["body"])
	})

	t.Run("object not found -> emits notFound", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Header:     http.Header{},
					Body:       io.NopCloser(strings.NewReader(`<?xml version="1.0"?><Error><Code>NoSuchKey</Code></Error>`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  spacesConfig,
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    spacesIntegration,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, channelNotFound, executionState.Channel)
		assert.Equal(t, "digitalocean.spaces.object.not_found", executionState.Type)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		payload, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "my-space", payload["bucket"])
		assert.Equal(t, "reports/daily.csv", payload["filePath"])
	})

	t.Run("API error on object fetch -> returns error", func(t *testing.T) {
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
			Configuration:  spacesConfig,
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    spacesIntegration,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get object")
		assert.False(t, executionState.Passed)
	})

	t.Run("API error on tagging fetch -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type":   []string{"text/csv"},
						"Content-Length": []string{"42300"},
						"Last-Modified":  []string{"Wed, 25 Mar 2026 08:45:00 GMT"},
						"Etag":           []string{`"a1b2c3d4ef567890"`},
					},
					Body: io.NopCloser(strings.NewReader("")),
				},
				{
					StatusCode: http.StatusForbidden,
					Header:     http.Header{},
					Body:       io.NopCloser(strings.NewReader(`<?xml version="1.0"?><Error><Code>AccessDenied</Code></Error>`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  spacesConfig,
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    spacesIntegration,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get object tags")
		assert.False(t, executionState.Passed)
	})
}
