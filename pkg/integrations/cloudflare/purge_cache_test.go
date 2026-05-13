package cloudflare

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__PurgeCache__Setup(t *testing.T) {
	component := &PurgeCache{}

	t.Run("missing zone returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"mode": "files", "files": []any{"https://example.com/a.js"}},
		})
		require.ErrorContains(t, err, "zone is required")
	})

	t.Run("missing mode returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"zone": "zone123"},
		})
		require.ErrorContains(t, err, "mode is required")
	})

	t.Run("invalid mode returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"zone": "zone123", "mode": "unknown"},
		})
		require.ErrorContains(t, err, "mode must be one of")
	})

	t.Run("files mode without URLs returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"zone": "zone123", "mode": "files"},
		})
		require.ErrorContains(t, err, "at least one URL is required")
	})

	t.Run("tags mode without tags returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"zone": "zone123", "mode": "tags"},
		})
		require.ErrorContains(t, err, "at least one tag is required")
	})

	t.Run("hosts mode without hosts returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"zone": "zone123", "mode": "hosts"},
		})
		require.ErrorContains(t, err, "at least one hostname is required")
	})

	t.Run("prefixes mode without prefixes returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"zone": "zone123", "mode": "prefixes"},
		})
		require.ErrorContains(t, err, "at least one prefix is required")
	})

	t.Run("everything mode passes without additional fields", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"zone": "zone123", "mode": "everything"},
		})
		require.NoError(t, err)
	})

	t.Run("files mode with URLs passes", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":  "zone123",
				"mode":  "files",
				"files": []any{"https://example.com/a.js"},
			},
		})
		require.NoError(t, err)
	})

	t.Run("prefixes mode with prefixes passes", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":     "zone123",
				"mode":     "prefixes",
				"prefixes": []any{"www.example.com/foo"},
			},
		})
		require.NoError(t, err)
	})
}

func Test__PurgeCache__Execute(t *testing.T) {
	component := &PurgeCache{}

	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiToken": "token123"},
		Metadata:      Metadata{Zones: []Zone{{ID: "zone123", Name: "example.com"}}},
	}

	t.Run("purges by files and emits result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"success":true,"result":{"id":"purge-abc"}}`)),
				},
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"zone":  "zone123",
				"mode":  "files",
				"files": []any{"https://example.com/app.js", "https://example.com/style.css"},
			},
			HTTP:           httpContext,
			Integration:    integration,
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.Equal(t, PurgeCachePayloadType, execState.Type)
		assert.Len(t, execState.Payloads, 1)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/zones/zone123/purge_cache", httpContext.Requests[0].URL.String())
		assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)

		var body map[string]any
		require.NoError(t, json.NewDecoder(httpContext.Requests[0].Body).Decode(&body))
		assert.Nil(t, body["purge_everything"])
		files, ok := body["files"].([]any)
		require.True(t, ok)
		assert.Len(t, files, 2)
	})

	t.Run("purges everything and sends purge_everything flag", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"success":true,"result":{"id":"purge-xyz"}}`)),
				},
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"zone": "zone123",
				"mode": "everything",
			},
			HTTP:           httpContext,
			Integration:    integration,
			ExecutionState: execState,
		})

		require.NoError(t, err)

		var body map[string]any
		require.NoError(t, json.NewDecoder(httpContext.Requests[0].Body).Decode(&body))
		assert.Equal(t, true, body["purge_everything"])
		assert.Nil(t, body["files"])
	})

	t.Run("purges by prefixes and emits result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"success":true,"result":{"id":"purge-prefix"}}`)),
				},
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"zone":     "zone123",
				"mode":     "prefixes",
				"prefixes": []any{"www.example.com/foo", "www.example.com/bar"},
			},
			HTTP:           httpContext,
			Integration:    integration,
			ExecutionState: execState,
		})

		require.NoError(t, err)

		var body map[string]any
		require.NoError(t, json.NewDecoder(httpContext.Requests[0].Body).Decode(&body))
		prefixes, ok := body["prefixes"].([]any)
		require.True(t, ok)
		assert.Equal(t, []any{"www.example.com/foo", "www.example.com/bar"}, prefixes)

		wrapped := execState.Payloads[0].(map[string]any)
		payload := wrapped["data"].(map[string]any)
		assert.Equal(t, "prefixes", payload["mode"])
		assert.Equal(t, []string{"www.example.com/foo", "www.example.com/bar"}, payload["prefixes"])
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"success":false,"errors":[{"message":"You do not have permission"}]}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"zone":  "zone123",
				"mode":  "files",
				"files": []any{"https://example.com/app.js"},
			},
			HTTP:           httpContext,
			Integration:    integration,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to purge cache")
	})
}
