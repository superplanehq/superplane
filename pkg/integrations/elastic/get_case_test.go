package elastic

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetCase__Configuration(t *testing.T) {
	fields := (&GetCase{}).Configuration()

	require.Len(t, fields, 1)
	assert.Equal(t, "caseId", fields[0].Name)
	assert.Equal(t, "Case", fields[0].Label)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, fields[0].Type)
	require.NotNil(t, fields[0].TypeOptions)
	require.NotNil(t, fields[0].TypeOptions.Resource)
	assert.Equal(t, ResourceTypeCase, fields[0].TypeOptions.Resource.Type)
}

func Test__GetCase__Setup(t *testing.T) {
	c := &GetCase{}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
		"url":       "https://elastic.example.com",
		"kibanaUrl": "https://kibana.example.com",
		"authType":  "apiKey",
		"apiKey":    "test-api-key",
	}}

	t.Run("missing caseId -> error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "caseId is required")
	})

	t.Run("whitespace-only caseId -> error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{"caseId": "   "},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "caseId is required")
	})

	t.Run("valid config -> success", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"id": "case-abc",
						"title": "Incident 42"
					}`)),
				},
			},
		}
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{"caseId": "case-abc"},
			HTTP:          httpCtx,
			Integration:   integrationCtx,
			Metadata:      metadata,
		})
		require.NoError(t, err)
		assert.Equal(t, GetCaseNodeMetadata{CaseName: "Incident 42"}, metadata.Metadata)
	})

	t.Run("expression caseId -> stores expression as metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{"caseId": "{{ previous().data.id }}"},
			Metadata:      metadata,
		})
		require.NoError(t, err)
		assert.Equal(t, GetCaseNodeMetadata{CaseName: "{{ previous().data.id }}"}, metadata.Metadata)
	})
}

func Test__GetCase__Execute(t *testing.T) {
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
				"description": "Something went wrong in prod",
				"status": "open",
				"severity": "high",
				"tags": ["prod", "infra"],
				"version": "WzEsMV0=",
				"created_at": "2024-01-15T10:00:00.000Z",
				"updated_at": "2024-01-16T08:30:00.000Z"
			}`)),
		}
	}

	t.Run("retrieves case and emits full payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{successResponse()},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&GetCase{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"caseId": "case-abc",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx(),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		require.Len(t, httpCtx.Requests, 1)
		req := httpCtx.Requests[0]
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, "https://kibana.example.com/api/cases/case-abc", req.URL.String())
		assert.Equal(t, "ApiKey test-api-key", req.Header.Get("Authorization"))

		require.Len(t, state.Payloads, 1)
		wrapper := state.Payloads[0].(map[string]any)
		data := wrapper["data"].(map[string]any)
		assert.Equal(t, "case-abc", data["id"])
		assert.Equal(t, "Incident 42", data["title"])
		assert.Equal(t, "Something went wrong in prod", data["description"])
		assert.Equal(t, "open", data["status"])
		assert.Equal(t, "high", data["severity"])
		assert.Equal(t, "WzEsMV0=", data["version"])
		assert.Equal(t, "2024-01-15T10:00:00.000Z", data["createdAt"])
		assert.Equal(t, "2024-01-16T08:30:00.000Z", data["updatedAt"])
		assert.NotNil(t, data["tags"])
	})

	t.Run("Kibana error -> fails execution", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"statusCode":404,"error":"Not Found","message":"Case case-abc not found"}`)),
				},
			},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&GetCase{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"caseId": "case-abc",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx(),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to get case")
	})
}
