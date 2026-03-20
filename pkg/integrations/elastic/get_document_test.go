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

func Test__GetDocument__Configuration(t *testing.T) {
	fields := (&GetDocument{}).Configuration()

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
}

func Test__GetDocument__Setup(t *testing.T) {
	c := &GetDocument{}
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
						"_id": "abc123",
						"_index": "my-index",
						"_version": 1,
						"found": true,
						"_source": {"k":"v"}
					}`)),
				},
			},
		}
		meta := &contexts.MetadataContext{}

		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"index":    "my-index",
				"document": "abc123",
			},
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Metadata:    meta,
		})
		require.NoError(t, err)
		assert.Equal(t, GetDocumentSetupMetadata{Index: "my-index", Document: "abc123"}, meta.Metadata)
	})

	t.Run("document does not exist -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"index":"my-index"}]`)),
				},
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error":{"type":"document_missing_exception"}}`)),
				},
			},
		}

		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"index":    "my-index",
				"document": "missing-doc",
			},
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, `failed to verify document "missing-doc" in index "my-index"`)
	})

}

func Test__GetDocument__Execute(t *testing.T) {
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
				"_version": 3,
				"found": true,
				"_source": {"event": "deploy", "version": "1.2.3"}
			}`)),
		}
	}

	t.Run("retrieves document and emits payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{successResponse()},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&GetDocument{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"index":    "workflow-audit",
				"document": "doc-1",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx("apiKey"),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		require.Len(t, httpCtx.Requests, 1)
		req := httpCtx.Requests[0]
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, "https://elastic.example.com/workflow-audit/_doc/doc-1", req.URL.String())
		assert.Equal(t, "ApiKey test-api-key", req.Header.Get("Authorization"))

		require.Len(t, state.Payloads, 1)
		wrapper := state.Payloads[0].(map[string]any)
		data := wrapper["data"].(map[string]any)
		assert.Equal(t, "doc-1", data["id"])
		assert.Equal(t, "workflow-audit", data["index"])
		assert.Equal(t, 3, data["version"])
		assert.NotNil(t, data["source"])
	})

	t.Run("uses basic auth when authType is basic", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{successResponse()},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&GetDocument{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"index":    "my-index",
				"document": "doc-1",
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
					Body:       io.NopCloser(strings.NewReader(`{"error":{"type":"index_not_found_exception"}}`)),
				},
			},
		}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := (&GetDocument{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"index":    "my-index",
				"document": "doc-1",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx("apiKey"),
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to get document")
	})

}

func Test__Elastic__ListResources__Document(t *testing.T) {
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"url":      "https://elastic.example.com",
			"authType": "apiKey",
			"apiKey":   "test-api-key",
		},
	}

	t.Run("lists documents for selected index", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"hits": {
							"hits": [
								{"_id": "doc-1", "_index": "workflow-audit"},
								{"_id": "doc-2", "_index": "workflow-audit"}
							]
						}
					}`)),
				},
			},
		}

		resources, err := (&Elastic{}).ListResources(ResourceTypeDocument, core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Parameters:  map[string]string{"index": "workflow-audit"},
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, ResourceTypeDocument, resources[0].Type)
		assert.Equal(t, "doc-1", resources[0].Name)
		assert.Equal(t, "doc-1", resources[0].ID)
		assert.Equal(t, "doc-2", resources[1].Name)
		assert.Equal(t, "https://elastic.example.com/workflow-audit/_search", httpCtx.Requests[0].URL.String())
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
	})

	t.Run("returns empty resources when index is not selected", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}

		resources, err := (&Elastic{}).ListResources(ResourceTypeDocument, core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Parameters:  map[string]string{},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("returns empty resources when index is an expression", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}

		resources, err := (&Elastic{}).ListResources(ResourceTypeDocument, core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Parameters:  map[string]string{"index": "{{ previous().data.index }}"},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
		assert.Empty(t, httpCtx.Requests)
	})
}
