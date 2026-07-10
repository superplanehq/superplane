package claude

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DownloadFile__Setup(t *testing.T) {
	component := &DownloadFile{}

	t.Run("missing file returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"file": ""},
		}
		require.ErrorContains(t, component.Setup(ctx), "file is required")
	})

	t.Run("valid configuration passes", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"file": "{{ $.textPrompt.data.artifacts[0].fileId }}"},
		}
		require.NoError(t, component.Setup(ctx))
	})
}

func Test__DownloadFile__Execute(t *testing.T) {
	component := &DownloadFile{}

	metadataJSON := func(mimeType string, size int64) string {
		return fmt.Sprintf(`{"id":"file_1","type":"file","filename":"report.csv","mime_type":"%s","size_bytes":%d,"created_at":"2026-07-10T12:00:00Z","downloadable":true}`, mimeType, size)
	}

	t.Run("text file is emitted as plain text", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(metadataJSON("text/csv", 12)))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("a,b\n1,2\n"))},
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: make(map[string]string)}
		ctx := core.ExecutionContext{
			Configuration:  map[string]any{"file": "file_1"},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
			ExecutionState: execState,
		}

		require.NoError(t, component.Execute(ctx))
		assert.Equal(t, FileDownloadedPayloadType, execState.Type)

		data := execState.Payloads[0].(map[string]any)["data"].(FileContentPayload)
		assert.Equal(t, "text", data.Encoding)
		assert.Equal(t, "a,b\n1,2\n", data.Content)
		assert.Equal(t, "report.csv", data.Filename)

		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[1].URL.Path, "/files/file_1/content")
	})

	t.Run("binary file is base64 encoded", func(t *testing.T) {
		binary := "\x89PNG\r\n\x1a\n"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(metadataJSON("image/png", 8)))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(binary))},
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: make(map[string]string)}
		ctx := core.ExecutionContext{
			Configuration:  map[string]any{"file": "file_1"},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
			ExecutionState: execState,
		}

		require.NoError(t, component.Execute(ctx))
		data := execState.Payloads[0].(map[string]any)["data"].(FileContentPayload)
		assert.Equal(t, "base64", data.Encoding)
		assert.Equal(t, base64.StdEncoding.EncodeToString([]byte(binary)), data.Content)
	})

	t.Run("oversized file is rejected before download", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(metadataJSON("application/zip", maxDownloadSizeBytes+1)))},
			},
		}
		ctx := core.ExecutionContext{
			Configuration:  map[string]any{"file": "file_1"},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: make(map[string]string)},
		}

		require.ErrorContains(t, component.Execute(ctx), "exceeds")
		// Only the metadata request went out; the content was never fetched.
		require.Len(t, httpContext.Requests, 1)
	})

	t.Run("non-downloadable file surfaces API error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(metadataJSON("application/pdf", 100)))},
				{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_request_error","message":"Not downloadable"}}`))},
			},
		}
		ctx := core.ExecutionContext{
			Configuration:  map[string]any{"file": "file_1"},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: make(map[string]string)},
		}
		require.ErrorContains(t, component.Execute(ctx), "failed to download file")
	})
}

func Test__isTextMIME(t *testing.T) {
	assert.True(t, isTextMIME("text/plain"))
	assert.True(t, isTextMIME("text/csv; charset=utf-8"))
	assert.True(t, isTextMIME("application/json"))
	assert.True(t, isTextMIME("application/ld+json"))
	assert.False(t, isTextMIME("image/png"))
	assert.False(t, isTextMIME("application/pdf"))
	assert.False(t, isTextMIME(""))
}
