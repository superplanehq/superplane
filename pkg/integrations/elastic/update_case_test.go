package elastic

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__UpdateCase__Setup(t *testing.T) {
	c := &UpdateCase{}

	t.Run("missing case -> error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "caseId is required")
	})

	t.Run("missing version -> allowed", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{"case": "case-abc"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "at least one field to update is required")
	})

	t.Run("no update fields -> error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"case": "case-abc",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "at least one field to update is required")
	})

	t.Run("valid config -> success", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"case-abc","title":"Production incident","status":"open","severity":"high","version":"WzEsMV0="}`)),
				},
			},
		}
		meta := &contexts.MetadataContext{}
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"case":   "case-abc",
				"status": "closed",
			},
			Metadata:    meta,
			HTTP:        httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"url": "https://elastic.example.com", "kibanaUrl": "https://kibana.example.com", "authType": "apiKey", "apiKey": "test-api-key"}},
		})
		require.NoError(t, err)
		saved, ok := meta.Metadata.(UpdateCaseNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "Production incident", saved.CaseName)
	})
}

func Test__UpdateCase__Execute(t *testing.T) {
	integrationCtx := func() *contexts.IntegrationContext {
		return &contexts.IntegrationContext{Configuration: map[string]any{
			"url":       "https://elastic.example.com",
			"kibanaUrl": "https://kibana.example.com",
			"authType":  "apiKey",
			"apiKey":    "test-api-key",
		}}
	}

	successResponse := func() *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`[{
				"id": "case-abc",
				"title": "Incident 42 - Resolved",
				"status": "closed",
				"severity": "high",
				"version": "WzIsMV0=",
				"updated_at": "2024-01-16T09:00:00.000Z"
			}]`)),
		}
	}

	t.Run("updates case and emits payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"case-abc","title":"Incident 42","status":"open","severity":"high","version":"WzEsMV0="}`)),
				},
				successResponse(),
			},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&UpdateCase{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"case":   "case-abc",
				"status": "closed",
				"title":  "Incident 42 - Resolved",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx(),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		req := httpCtx.Requests[1]
		assert.Equal(t, http.MethodPatch, req.Method)
		assert.Equal(t, "https://kibana.example.com/api/cases", req.URL.String())
		assert.Equal(t, "ApiKey test-api-key", req.Header.Get("Authorization"))

		require.Len(t, state.Payloads, 1)
		wrapper := state.Payloads[0].(map[string]any)
		data := wrapper["data"].(map[string]any)
		assert.Equal(t, "case-abc", data["id"])
		assert.Equal(t, "Incident 42 - Resolved", data["title"])
		assert.Equal(t, "closed", data["status"])
		assert.Equal(t, "high", data["severity"])
		assert.Equal(t, "WzIsMV0=", data["version"])
		assert.Equal(t, "2024-01-16T09:00:00.000Z", data["updatedAt"])
	})

	t.Run("updates only provided fields", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"case-abc","title":"Incident 42","status":"open","severity":"low","version":"WzEsMV0="}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[{
						"id": "case-abc",
						"title": "Incident 42",
						"status": "in-progress",
						"severity": "low",
						"version": "WzIsMV0=",
						"updated_at": "2024-01-16T09:00:00.000Z"
					}]`)),
				},
			},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&UpdateCase{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"case":   "case-abc",
				"status": "in-progress",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx(),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "in-progress", data["status"])
	})

	t.Run("always fetches latest version before update", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"case-abc","title":"Incident 42","status":"open","severity":"high","version":"WzEsMV0="}`)),
				},
				successResponse(),
			},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&UpdateCase{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"case":   "case-abc",
				"status": "closed",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx(),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Equal(t, "https://kibana.example.com/api/cases/case-abc", httpCtx.Requests[0].URL.String())
		assert.Equal(t, http.MethodPatch, httpCtx.Requests[1].Method)
	})

	t.Run("provided version is ignored in favor of latest case version", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"case-abc","title":"Incident 42","status":"open","severity":"high","version":"LATEST-VERSION"}`)),
				},
				successResponse(),
			},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&UpdateCase{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"case":    "case-abc",
				"version": "STALE-VERSION",
				"status":  "closed",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx(),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		require.Len(t, httpCtx.Requests, 2)
		body, err := io.ReadAll(httpCtx.Requests[1].Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), `"version":"LATEST-VERSION"`)
		assert.NotContains(t, string(body), "STALE-VERSION")
	})

	t.Run("no update fields -> fails execution early", func(t *testing.T) {
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		httpCtx := &contexts.HTTPContext{}

		err := (&UpdateCase{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"case": "case-abc",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx(),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "at least one field to update is required")
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("Kibana error -> fails execution", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"case-abc","title":"Incident 42","status":"open","severity":"high","version":"WzEsMV0="}`)),
				},
				{
					StatusCode: http.StatusConflict,
					Body:       io.NopCloser(strings.NewReader(`{"statusCode":409,"error":"Conflict","message":"Version mismatch"}`)),
				},
			},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&UpdateCase{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"case":   "case-abc",
				"status": "closed",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx(),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to update case")
	})

	t.Run("get case version fails -> fails execution", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"statusCode":404,"error":"Not Found"}`)),
				},
			},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&UpdateCase{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"case":   "case-abc",
				"status": "closed",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx(),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to get case version")
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
	})
}
