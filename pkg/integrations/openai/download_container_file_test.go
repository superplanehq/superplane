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

func Test__DownloadContainerFile__Setup(t *testing.T) {
	component := &DownloadContainerFile{}

	t.Run("missing containerId returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"containerId": "", "fileId": "cfile_1"},
		}
		require.ErrorContains(t, component.Setup(ctx), "containerId is required")
	})

	t.Run("missing fileId returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"containerId": "cntr_1", "fileId": ""},
		}
		require.ErrorContains(t, component.Setup(ctx), "fileId is required")
	})

	t.Run("expressions pass validation", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"containerId": "{{ $.textPrompt.data.artifacts[0].containerId }}",
				"fileId":      "{{ $.textPrompt.data.artifacts[0].fileId }}",
			},
		}
		require.NoError(t, component.Setup(ctx))
	})
}

func Test__DownloadContainerFile__Execute(t *testing.T) {
	component := &DownloadContainerFile{}

	t.Run("downloads container file and derives filename", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
					"id": "cfile_1",
					"object": "container.file",
					"container_id": "cntr_1",
					"created_at": 1747848842,
					"bytes": 8,
					"path": "/mnt/data/report.csv",
					"source": "assistant"
				}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("a,b\n1,2\n"))},
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: make(map[string]string)}
		ctx := core.ExecutionContext{
			Configuration:  map[string]any{"containerId": "cntr_1", "fileId": "cfile_1"},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}},
			ExecutionState: execState,
		}

		require.NoError(t, component.Execute(ctx))
		assert.Equal(t, ContainerFileDownloadedPayloadType, execState.Type)

		data := execState.Payloads[0].(map[string]any)["data"].(ContainerFilePayload)
		assert.Equal(t, "cfile_1", data.FileID)
		assert.Equal(t, "cntr_1", data.ContainerID)
		assert.Equal(t, "/mnt/data/report.csv", data.Path)
		assert.Equal(t, "report.csv", data.Filename)
		assert.Equal(t, "text", data.Encoding)
		assert.Equal(t, "a,b\n1,2\n", data.Content)

		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[0].URL.Path, "/containers/cntr_1/files/cfile_1")
		assert.Contains(t, httpContext.Requests[1].URL.Path, "/containers/cntr_1/files/cfile_1/content")
	})

	t.Run("expired container surfaces API error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"Container is expired"}}`))},
			},
		}
		ctx := core.ExecutionContext{
			Configuration:  map[string]any{"containerId": "cntr_1", "fileId": "cfile_1"},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: make(map[string]string)},
		}
		require.ErrorContains(t, component.Execute(ctx), "failed to get container file")
	})

	t.Run("oversized container file is rejected before download", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"cfile_1","object":"container.file","container_id":"cntr_1","bytes":99999999999,"path":"/mnt/data/big.bin","source":"assistant"}`))},
			},
		}
		ctx := core.ExecutionContext{
			Configuration:  map[string]any{"containerId": "cntr_1", "fileId": "cfile_1"},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: make(map[string]string)},
		}
		require.ErrorContains(t, component.Execute(ctx), "exceeds")
		require.Len(t, httpContext.Requests, 1)
	})
}
