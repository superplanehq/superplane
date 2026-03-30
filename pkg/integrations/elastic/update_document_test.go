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

func Test__UpdateDocument__Setup(t *testing.T) {
	c := &UpdateDocument{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"url":      "https://elastic.example.com",
			"authType": "apiKey",
			"apiKey":   "test-api-key",
		},
	}

	t.Run("missing index -> error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{},
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

	t.Run("missing fields -> error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"index":    "my-index",
				"document": "doc-1",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "fields is required")
	})

	t.Run("valid config -> success", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"index":"my-index"}]`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"_id": "doc-1",
						"_index": "my-index",
						"_version": 2,
						"found": true,
						"_source": {"status":"open"}
					}`)),
				},
			},
		}
		meta := &contexts.MetadataContext{}

		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"index":    "my-index",
				"document": "doc-1",
				"fields":   map[string]any{"status": "done"},
			},
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Metadata:    meta,
		})
		require.NoError(t, err)
		assert.Equal(t, UpdateDocumentSetupMetadata{Index: "my-index", Document: "doc-1"}, meta.Metadata)
	})

	t.Run("index does not exist -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"index":"other-index"}]`)),
				},
			},
		}

		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"index":    "my-index",
				"document": "doc-1",
				"fields":   map[string]any{"status": "done"},
			},
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, `index "my-index" was not found`)
	})

}

func Test__UpdateDocument__Configuration(t *testing.T) {
	c := &UpdateDocument{}

	fields := c.Configuration()
	require.Len(t, fields, 3)

	var documentIDField *configuration.Field
	for i := range fields {
		if fields[i].Name == "document" {
			documentIDField = &fields[i]
			break
		}
	}

	require.NotNil(t, documentIDField)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, documentIDField.Type)
	require.NotNil(t, documentIDField.TypeOptions)
	require.NotNil(t, documentIDField.TypeOptions.Resource)
	assert.Equal(t, ResourceTypeDocument, documentIDField.TypeOptions.Resource.Type)
	require.Len(t, documentIDField.TypeOptions.Resource.Parameters, 1)
	assert.Equal(t, "index", documentIDField.TypeOptions.Resource.Parameters[0].Name)
	require.NotNil(t, documentIDField.TypeOptions.Resource.Parameters[0].ValueFrom)
	assert.Equal(t, "index", documentIDField.TypeOptions.Resource.Parameters[0].ValueFrom.Field)

	updateFields := fields[2]
	assert.Equal(t, "fields", updateFields.Name)
	assert.Equal(t, map[string]any{
		onDocumentIndexedTimeField: defaultDocumentTimestampTemplate,
	}, updateFields.Default)
}

func Test__UpdateDocument__Execute(t *testing.T) {
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
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"_id": "doc-1",
				"_index": "workflow-audit",
				"result": "updated",
				"_version": 4
			}`)),
		}
	}

	t.Run("updates document and emits payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{successResponse()},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&UpdateDocument{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"index":    "workflow-audit",
				"document": "doc-1",
				"fields":   map[string]any{"status": "done"},
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
		assert.Equal(t, "https://elastic.example.com/workflow-audit/_update/doc-1", req.URL.String())
		assert.Equal(t, "ApiKey test-api-key", req.Header.Get("Authorization"))

		require.Len(t, state.Payloads, 1)
		wrapper := state.Payloads[0].(map[string]any)
		data := wrapper["data"].(map[string]any)
		assert.Equal(t, "doc-1", data["id"])
		assert.Equal(t, "workflow-audit", data["index"])
		assert.Equal(t, "updated", data["result"])
		assert.Equal(t, 4, data["version"])
	})

	t.Run("uses basic auth when authType is basic", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{successResponse()},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&UpdateDocument{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"index":    "my-index",
				"document": "doc-1",
				"fields":   map[string]any{"k": "v"},
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
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error":{"type":"document_missing_exception"}}`)),
				},
			},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&UpdateDocument{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"index":    "my-index",
				"document": "doc-1",
				"fields":   map[string]any{"k": "v"},
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx("apiKey"),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to update document")
	})

	t.Run("nil fields -> fails execution", func(t *testing.T) {
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&UpdateDocument{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"index":    "my-index",
				"document": "doc-1",
			},
			HTTP:           &contexts.HTTPContext{},
			Integration:    integrationCtx("apiKey"),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "fields is required")
	})

}
