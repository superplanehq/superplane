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

func Test__CreateCase__Setup(t *testing.T) {
	c := &CreateCase{}

	t.Run("missing title -> error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "title is required")
	})

	t.Run("whitespace-only title -> error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{"title": "   "},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "title is required")
	})

	t.Run("valid config -> success", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{"title": "Incident 42"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})
}

func Test__CreateCase__Execute(t *testing.T) {
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
			Body: io.NopCloser(strings.NewReader(`{
				"id": "case-abc",
				"title": "Incident 42",
				"status": "open",
				"severity": "high",
				"version": "WzEsMV0=",
				"created_at": "2024-01-15T10:00:00.000Z",
				"updated_at": "2024-01-15T10:00:00.000Z",
				"tags": ["prod", "infra"]
			}`)),
		}
	}

	t.Run("creates case and emits payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{successResponse()},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&CreateCase{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"title":       "Incident 42",
				"description": "Something went wrong",
				"severity":    "high",
				"owner":       "cases",
				"tags":        []string{"prod", "infra"},
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx(),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		require.Len(t, httpCtx.Requests, 1)
		req := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "https://kibana.example.com/api/cases", req.URL.String())
		assert.Equal(t, "ApiKey test-api-key", req.Header.Get("Authorization"))

		require.Len(t, state.Payloads, 1)
		wrapper := state.Payloads[0].(map[string]any)
		data := wrapper["data"].(map[string]any)
		assert.Equal(t, "case-abc", data["id"])
		assert.Equal(t, "Incident 42", data["title"])
		assert.Equal(t, "open", data["status"])
		assert.Equal(t, "high", data["severity"])
		assert.Equal(t, "WzEsMV0=", data["version"])
		assert.Equal(t, "2024-01-15T10:00:00.000Z", data["createdAt"])
	})

	t.Run("defaults severity to low and owner to cases when omitted", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"id": "case-xyz",
						"title": "Quick case",
						"status": "open",
						"severity": "low",
						"version": "WzEsMV0=",
						"created_at": "2024-01-15T10:00:00.000Z",
						"updated_at": "2024-01-15T10:00:00.000Z"
					}`)),
				},
			},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&CreateCase{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"title": "Quick case",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx(),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
	})

	t.Run("Kibana error -> fails execution", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"statusCode":400,"error":"Bad Request","message":"Invalid value"}`)),
				},
			},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&CreateCase{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"title": "Incident 42",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx(),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to create case")
	})
}
