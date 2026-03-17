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

func Test__IndexDocument__Setup(t *testing.T) {
	c := &IndexDocument{}

	t.Run("missing index -> error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "index is required")
	})

	t.Run("whitespace-only index -> error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{"index": "   "},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "index is required")
	})

	t.Run("missing document -> error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{"index": "my-index"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "document is required")
	})

	t.Run("valid config -> success", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"index":    "my-index",
				"document": map[string]any{"key": "value"},
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})
}

func Test__IndexDocument__Execute(t *testing.T) {
	integrationCtx := func(authType string) *contexts.IntegrationContext {
		cfg := map[string]any{
			"url":      "https://elastic.example.com",
			"authType": authType,
		}
		if authType == "apiKey" {
			cfg["apiKey"] = "test-api-key"
		} else {
			cfg["username"] = "elastic"
			cfg["password"] = "secret"
		}
		return &contexts.IntegrationContext{Configuration: cfg}
	}

	successResponse := func() *http.Response {
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body: io.NopCloser(strings.NewReader(`{
				"_id": "abc123",
				"_index": "workflow-audit",
				"result": "created",
				"_version": 1,
				"_shards": {"successful": 1, "failed": 0}
			}`)),
		}
	}

	t.Run("indexes document with auto-generated ID (POST)", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{successResponse()},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&IndexDocument{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"index":    "workflow-audit",
				"document": map[string]any{"event": "deploy", "version": "1.2.3"},
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx("apiKey"),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		require.Len(t, httpCtx.Requests, 1)
		req := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "https://elastic.example.com/workflow-audit/_doc", req.URL.String())
		assert.Equal(t, "ApiKey test-api-key", req.Header.Get("Authorization"))

		require.Len(t, state.Payloads, 1)
		wrapper := state.Payloads[0].(map[string]any)
		data := wrapper["data"].(map[string]any)
		assert.Equal(t, "abc123", data["id"])
		assert.Equal(t, "workflow-audit", data["index"])
		assert.Equal(t, "created", data["result"])
		assert.Equal(t, 1, data["version"])
	})

	t.Run("indexes document with explicit ID (PUT)", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{successResponse()},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&IndexDocument{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"index":      "workflow-audit",
				"document":   map[string]any{"event": "deploy"},
				"documentId": "run-42",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx("apiKey"),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		req := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPut, req.Method)
		assert.Equal(t, "https://elastic.example.com/workflow-audit/_doc/run-42", req.URL.String())
	})

	t.Run("uses basic auth header when authType is basic", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{successResponse()},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&IndexDocument{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"index":    "my-index",
				"document": map[string]any{"k": "v"},
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx("basic"),
			ExecutionState: state,
		})

		require.NoError(t, err)
		req := httpCtx.Requests[0]
		user, pass, ok := req.BasicAuth()
		require.True(t, ok)
		assert.Equal(t, "elastic", user)
		assert.Equal(t, "secret", pass)
	})

	t.Run("Elasticsearch error -> fails execution", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"error":{"type":"mapper_parsing_exception"}}`)),
				},
			},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&IndexDocument{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"index":    "my-index",
				"document": map[string]any{"k": "v"},
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx("apiKey"),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to index document")
	})

	t.Run("nil document -> fails execution", func(t *testing.T) {
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&IndexDocument{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"index": "my-index",
			},
			HTTP:           &contexts.HTTPContext{},
			Integration:    integrationCtx("apiKey"),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "document is required")
	})
}
