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

	t.Run("missing caseId -> error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "caseId is required")
	})

	t.Run("missing version -> error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{"caseId": "case-abc"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "version is required")
	})

	t.Run("no update fields -> error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"caseId":  "case-abc",
				"version": "WzEsMV0=",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "at least one field to update is required")
	})

	t.Run("valid config -> success", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"caseId":  "case-abc",
				"version": "WzEsMV0=",
				"status":  "closed",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.NoError(t, err)
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
			Responses: []*http.Response{successResponse()},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&UpdateCase{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"caseId":  "case-abc",
				"version": "WzEsMV0=",
				"status":  "closed",
				"title":   "Incident 42 - Resolved",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx(),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		require.Len(t, httpCtx.Requests, 1)
		req := httpCtx.Requests[0]
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
				"caseId":  "case-abc",
				"version": "WzEsMV0=",
				"status":  "in-progress",
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

	t.Run("no update fields -> fails execution early", func(t *testing.T) {
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		httpCtx := &contexts.HTTPContext{}

		err := (&UpdateCase{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"caseId":  "case-abc",
				"version": "WzEsMV0=",
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
					StatusCode: http.StatusConflict,
					Body:       io.NopCloser(strings.NewReader(`{"statusCode":409,"error":"Conflict","message":"Version mismatch"}`)),
				},
			},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&UpdateCase{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"caseId":  "case-abc",
				"version": "WzEsMV0=",
				"status":  "closed",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx(),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to update case")
	})
}
