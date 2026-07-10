package openai

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
			Configuration: map[string]any{"file": "file-abc123"},
		}
		require.NoError(t, component.Setup(ctx))
	})
}

func Test__DownloadFile__Execute(t *testing.T) {
	component := &DownloadFile{}

	metadataJSON := func(size int64) string {
		return fmt.Sprintf(`{"id":"file-1","object":"file","bytes":%d,"created_at":1707825600,"filename":"results.csv","purpose":"batch_output"}`, size)
	}

	t.Run("text file is emitted as plain text", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(metadataJSON(12)))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("a,b\n1,2\n"))},
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: make(map[string]string)}
		ctx := core.ExecutionContext{
			Configuration:  map[string]any{"file": "file-1"},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}},
			ExecutionState: execState,
		}

		require.NoError(t, component.Execute(ctx))
		assert.Equal(t, FileDownloadedPayloadType, execState.Type)

		data := execState.Payloads[0].(map[string]any)["data"].(FileDownloadPayload)
		assert.Equal(t, "text", data.Encoding)
		assert.Equal(t, "a,b\n1,2\n", data.Content)
		assert.Equal(t, "results.csv", data.Filename)
		assert.Equal(t, "batch_output", data.Purpose)
		assert.Equal(t, "https://platform.openai.com/storage/files/file-1", data.URL)

		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[1].URL.Path, "/files/file-1/content")
	})

	t.Run("binary file is base64 encoded", func(t *testing.T) {
		binary := "\x89PNG\r\n\x1a\n\x00\x01\x02"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(metadataJSON(int64(len(binary)))))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(binary))},
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: make(map[string]string)}
		ctx := core.ExecutionContext{
			Configuration:  map[string]any{"file": "file-1"},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}},
			ExecutionState: execState,
		}

		require.NoError(t, component.Execute(ctx))
		data := execState.Payloads[0].(map[string]any)["data"].(FileDownloadPayload)
		assert.Equal(t, "base64", data.Encoding)
		assert.Equal(t, base64.StdEncoding.EncodeToString([]byte(binary)), data.Content)
	})

	t.Run("oversized file is rejected before download", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(metadataJSON(maxDownloadBytes + 1)))},
			},
		}
		ctx := core.ExecutionContext{
			Configuration:  map[string]any{"file": "file-1"},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: make(map[string]string)},
		}

		require.ErrorContains(t, component.Execute(ctx), "exceeds")
		require.Len(t, httpContext.Requests, 1)
	})

	t.Run("restricted purpose surfaces API error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(metadataJSON(100)))},
				{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"Not allowed to download files of purpose: assistants"}}`))},
			},
		}
		ctx := core.ExecutionContext{
			Configuration:  map[string]any{"file": "file-1"},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: make(map[string]string)},
		}
		err := component.Execute(ctx)
		require.ErrorContains(t, err, "failed to download file content")
		require.ErrorContains(t, err, "Not allowed to download")
	})
}

func Test__isTextMIME(t *testing.T) {
	assert.True(t, isTextMIME("text/plain; charset=utf-8"))
	assert.True(t, isTextMIME("application/json"))
	assert.False(t, isTextMIME("image/png"))
	assert.False(t, isTextMIME("application/octet-stream"))
}
