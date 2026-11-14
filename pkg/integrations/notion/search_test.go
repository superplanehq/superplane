package notion

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestSearchExecute(t *testing.T) {
	for _, objectType := range []string{"all", "page", "data_source"} {
		t.Run(objectType, func(t *testing.T) {
			h := &contexts.HTTPContext{Responses: []*http.Response{response(200, `{"object":"list","results":[{"object":"page","id":"page"},{"object":"data_source","id":"source"}],"has_more":true,"next_cursor":"opaque-next"}`)}}
			state := &contexts.ExecutionStateContext{}
			err := (&Search{}).Execute(core.ExecutionContext{
				HTTP: h, Integration: integration(), ExecutionState: state,
				Configuration: map[string]any{"query": "Runbook", "objectType": objectType, "pageSize": 10, "startCursor": "opaque:+/cursor", "sortDirection": "descending"},
			})
			require.NoError(t, err)
			require.Len(t, h.Requests, 1)
			req := h.Requests[0]
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, "https://api.notion.com/v1/search", req.URL.String())
			assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
			var body map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			assert.Equal(t, "Runbook", body["query"])
			assert.Equal(t, float64(10), body["page_size"])
			assert.Equal(t, "opaque:+/cursor", body["start_cursor"])
			assert.Equal(t, map[string]any{"timestamp": "last_edited_time", "direction": "descending"}, body["sort"])
			if objectType == "all" {
				assert.NotContains(t, body, "filter")
			} else {
				assert.Equal(t, map[string]any{"property": "object", "value": objectType}, body["filter"])
			}
			assert.Equal(t, "default", state.Channel)
			assert.Equal(t, "notion.search", state.Type)
			require.Len(t, state.Payloads, 1)
			data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
			assert.Equal(t, true, data["has_more"])
			assert.Equal(t, "opaque-next", data["next_cursor"])
			assert.Len(t, data["results"], 2)
		})
	}

	t.Run("defaults and empty results still emit one output", func(t *testing.T) {
		h := &contexts.HTTPContext{Responses: []*http.Response{response(200, `{"object":"list","results":[],"has_more":false,"next_cursor":null}`)}}
		state := &contexts.ExecutionStateContext{}
		err := (&Search{}).Execute(core.ExecutionContext{HTTP: h, Integration: integration(), ExecutionState: state, Configuration: map[string]any{}})
		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.NewDecoder(h.Requests[0].Body).Decode(&body))
		assert.Equal(t, map[string]any{"page_size": float64(100)}, body)
		require.Len(t, state.Payloads, 1)
		assert.Empty(t, state.Payloads[0].(map[string]any)["data"].(map[string]any)["results"])
	})
}

func TestSearchValidation(t *testing.T) {
	for _, config := range []map[string]any{
		{"pageSize": 0}, {"pageSize": 101}, {"pageSize": 1.5},
		{"objectType": "database"}, {"sortDirection": "newest"},
	} {
		err := (&Search{}).Setup(core.SetupContext{Configuration: config})
		require.Error(t, err)
		h := &contexts.HTTPContext{}
		err = (&Search{}).Execute(core.ExecutionContext{Configuration: config, HTTP: h, Integration: integration()})
		require.Error(t, err)
		assert.Empty(t, h.Requests)
	}

	require.NoError(t, (&Search{}).Setup(core.SetupContext{Configuration: map[string]any{
		"pageSize": "{{ $['Input'].data.size }}", "objectType": "{{ $['Input'].data.type }}",
		"sortDirection": "{{ $['Input'].data.sort }}",
	}}))
}
