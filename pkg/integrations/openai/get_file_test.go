package openai

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

func Test__GetFile__Setup(t *testing.T) {
	component := &GetFile{}

	t.Run("missing file returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"file": ""},
		}
		require.ErrorContains(t, component.Setup(ctx), "file is required")
	})

	t.Run("expression file skips metadata lookup", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		ctx := core.SetupContext{
			Configuration: map[string]any{"file": "{{ $.trigger.data.fileId }}"},
			Integration:   &contexts.IntegrationContext{},
			HTTP:          &contexts.HTTPContext{},
			Metadata:      metadataCtx,
		}
		require.NoError(t, component.Setup(ctx))
		meta, ok := metadataCtx.Metadata.(FileNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "{{ $.trigger.data.fileId }}", meta.Filename)
	})

	t.Run("valid configuration resolves filename", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		ctx := core.SetupContext{
			Configuration: map[string]any{"file": "file-abc123"},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"file-abc123","object":"file","filename":"salesOverview.pdf","purpose":"assistants","bytes":175,"created_at":1707825600}`))},
				},
			},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}},
			Metadata:    metadataCtx,
		}
		require.NoError(t, component.Setup(ctx))
		meta, ok := metadataCtx.Metadata.(FileNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "salesOverview.pdf", meta.Filename)
	})
}

func Test__GetFile__Execute(t *testing.T) {
	component := &GetFile{}

	t.Run("emits file metadata with console link", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
					"id": "file-abc123",
					"object": "file",
					"bytes": 175,
					"created_at": 1707825600,
					"expires_at": 1710244800,
					"filename": "salesOverview.pdf",
					"purpose": "assistants"
				}`))},
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: make(map[string]string)}
		ctx := core.ExecutionContext{
			Configuration:  map[string]any{"file": "file-abc123"},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}},
			ExecutionState: execState,
		}

		require.NoError(t, component.Execute(ctx))
		assert.Equal(t, FileFetchedPayloadType, execState.Type)

		data := execState.Payloads[0].(map[string]any)["data"].(FilePayload)
		assert.Equal(t, "file-abc123", data.ID)
		assert.Equal(t, "salesOverview.pdf", data.Filename)
		assert.Equal(t, "assistants", data.Purpose)
		assert.Equal(t, int64(175), data.Bytes)
		assert.Equal(t, "2024-02-13T12:00:00Z", data.CreatedAt)
		assert.Equal(t, "2024-03-12T12:00:00Z", data.ExpiresAt)
		assert.Equal(t, "https://platform.openai.com/storage/files/file-abc123", data.URL)

		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.Path, "/files/file-abc123")
	})

	t.Run("expiresAt is omitted when not set", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"file-1","object":"file","bytes":10,"created_at":1707825600,"filename":"a.txt","purpose":"batch"}`))},
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
		data := execState.Payloads[0].(map[string]any)["data"].(FilePayload)
		assert.Empty(t, data.ExpiresAt)
	})

	t.Run("API error fails the run", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"No such file"}}`))},
			},
		}
		ctx := core.ExecutionContext{
			Configuration:  map[string]any{"file": "file-missing"},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: make(map[string]string)},
		}
		require.ErrorContains(t, component.Execute(ctx), "failed to get file")
	})
}
