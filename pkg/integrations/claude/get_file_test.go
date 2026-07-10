package claude

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

	t.Run("expression file skips metadata resolution", func(t *testing.T) {
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
			Configuration: map[string]any{"file": "file_1"},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"file_1","filename":"report.pdf","mime_type":"application/pdf","size_bytes":100,"downloadable":true}`))},
				},
			},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
			Metadata:    metadataCtx,
		}
		require.NoError(t, component.Setup(ctx))
		meta, ok := metadataCtx.Metadata.(FileNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "report.pdf", meta.Filename)
	})

	t.Run("metadata fetch failure falls back to file id", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		ctx := core.SetupContext{
			Configuration: map[string]any{"file": "file_1"},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`boom`))},
				},
			},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
			Metadata:    metadataCtx,
		}
		require.NoError(t, component.Setup(ctx))
		meta, ok := metadataCtx.Metadata.(FileNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "file_1", meta.Filename)
	})
}

func Test__GetFile__Execute(t *testing.T) {
	component := &GetFile{}

	t.Run("emits file metadata with download link", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
					"id": "file_1",
					"type": "file",
					"filename": "quarterly-report.pdf",
					"mime_type": "application/pdf",
					"size_bytes": 102400,
					"created_at": "2026-04-15T18:37:24.100435Z",
					"downloadable": true
				}`))},
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
		assert.Equal(t, FileFetchedPayloadType, execState.Type)
		require.Len(t, execState.Payloads, 1)

		data := execState.Payloads[0].(map[string]any)["data"].(FilePayload)
		assert.Equal(t, "file_1", data.ID)
		assert.Equal(t, "quarterly-report.pdf", data.Filename)
		assert.Equal(t, "application/pdf", data.MimeType)
		assert.Equal(t, int64(102400), data.SizeBytes)
		assert.True(t, data.Downloadable)
		assert.Equal(t, "https://api.anthropic.com/v1/files/file_1/content", data.DownloadURL)

		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.Path, "/files/file_1")
		assert.Equal(t, anthropicFilesBeta, httpContext.Requests[0].Header.Get("anthropic-beta"))
	})

	t.Run("API error fails the run", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"error":{"type":"not_found_error","message":"file not found"}}`))},
			},
		}
		ctx := core.ExecutionContext{
			Configuration:  map[string]any{"file": "file_missing"},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: make(map[string]string)},
		}
		require.ErrorContains(t, component.Execute(ctx), "failed to get file")
	})

	t.Run("missing file returns error", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration:  map[string]any{"file": ""},
			ExecutionState: &contexts.ExecutionStateContext{KVs: make(map[string]string)},
		}
		require.ErrorContains(t, component.Execute(ctx), "file is required")
	})
}
