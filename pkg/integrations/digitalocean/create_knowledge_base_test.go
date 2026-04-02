package digitalocean

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// readCapturedBody reads the body of the first captured HTTP request.
// The first request is always the create KB POST — subsequent requests are lookup calls with no body.
func readCapturedBody(t *testing.T, httpCtx *contexts.HTTPContext) []byte {
	t.Helper()
	require.NotEmpty(t, httpCtx.Requests)
	body, err := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, err)
	return body
}

// toStringSlice converts []any to []string.
func toStringSlice(in []any) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = v.(string)
	}
	return out
}

func Test__CreateKnowledgeBase__Setup(t *testing.T) {
	component := &CreateKnowledgeBase{}

	validSpacesSource := map[string]any{
		"type":         "spaces",
		"spacesBucket": "tor1/my-bucket",
	}

	validSeedSource := map[string]any{
		"type":           "web",
		"crawlType":      "seed",
		"webURL":         "https://docs.example.com",
		"crawlingOption": "SCOPED",
	}

	t.Run("missing name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"region":             "tor1",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "new",
				"dataSources":        []any{validSpacesSource},
			},
		})

		require.ErrorContains(t, err, "name is required")
	})

	t.Run("missing embeddingModelUUID returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":           "my-kb",
				"region":         "tor1",
				"projectId":      "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption": "new",
				"dataSources":    []any{validSpacesSource},
			},
		})

		require.ErrorContains(t, err, "embeddingModelUUID is required")
	})

		t.Run("missing region with new database returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "new",
				"dataSources":        []any{validSpacesSource},
			},
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing region with existing database -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "existing",
				"databaseId":         "abf1055a-745d-4c24-a1db-1959ea819264",
				"dataSources":        []any{validSpacesSource},
			},
		})

		require.NoError(t, err)
	})

	t.Run("missing projectId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"region":             "tor1",
				"databaseOption":     "new",
				"dataSources":        []any{validSpacesSource},
			},
		})

		require.ErrorContains(t, err, "projectId is required")
	})

	t.Run("existing database without databaseId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"region":             "tor1",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "existing",
				"dataSources":        []any{validSpacesSource},
			},
		})

		require.ErrorContains(t, err, "databaseId is required when using an existing database")
	})

	t.Run("empty dataSources returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"region":             "tor1",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "new",
				"dataSources":        []any{},
			},
		})

		require.ErrorContains(t, err, "at least one data source is required")
	})

	t.Run("missing dataSources field returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"region":             "tor1",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "new",
			},
		})

		require.ErrorContains(t, err, "at least one data source is required")
	})

	// ── Data source type validation ────────────────────────────────────────

	t.Run("data source with missing type returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"region":             "tor1",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "new",
				"dataSources": []any{
					map[string]any{"spacesBucket": "my-bucket"},
				},
			},
		})

		require.ErrorContains(t, err, "type is required")
	})

	t.Run("data source with unknown type returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"region":             "tor1",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "new",
				"dataSources": []any{
					map[string]any{"type": "dropbox"},
				},
			},
		})

		require.ErrorContains(t, err, "unsupported type")
	})

	// ── Spaces validation ──────────────────────────────────────────────────

	t.Run("spaces source without spacesBucket returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"region":             "tor1",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "new",
				"dataSources": []any{
					map[string]any{"type": "spaces"},
				},
			},
		})

		require.ErrorContains(t, err, "spacesBucket is required")
	})

	t.Run("spaces source with invalid bucket ID format returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"region":             "tor1",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "new",
				"dataSources": []any{
					map[string]any{
						"type":         "spaces",
						"spacesBucket": "just-a-bucket-name",
					},
				},
			},
		})

		require.ErrorContains(t, err, "invalid spacesBucket value")
	})

	// ── Web validation ─────────────────────────────────────────────────────

	t.Run("web source without webURL returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"region":             "tor1",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "new",
				"dataSources": []any{
					map[string]any{
						"type":           "web",
						"crawlType":      "seed",
						"crawlingOption": "SCOPED",
					},
				},
			},
		})

		require.ErrorContains(t, err, "webURL is required")
	})

	t.Run("web source without crawlType returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"region":             "tor1",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "new",
				"dataSources": []any{
					map[string]any{
						"type":   "web",
						"webURL": "https://example.com",
					},
				},
			},
		})

		require.ErrorContains(t, err, "crawlType is required")
	})

	t.Run("web source with invalid crawlType returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"region":             "tor1",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "new",
				"dataSources": []any{
					map[string]any{
						"type":      "web",
						"webURL":    "https://example.com",
						"crawlType": "rss",
					},
				},
			},
		})

		require.ErrorContains(t, err, "crawlType must be 'seed' or 'sitemap'")
	})

	t.Run("seed URL without crawlingOption returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"region":             "tor1",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "new",
				"dataSources": []any{
					map[string]any{
						"type":      "web",
						"webURL":    "https://example.com",
						"crawlType": "seed",
					},
				},
			},
		})

		require.ErrorContains(t, err, "crawlingOption is required for seed URLs")
	})

	// ── Chunking validation ────────────────────────────────────────────────

	t.Run("invalid chunking algorithm returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"region":             "tor1",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "new",
				"dataSources": []any{
					map[string]any{
						"type":              "spaces",
						"spacesBucket":      "tor1/my-bucket",
						"chunkingAlgorithm": "CHUNKING_ALGORITHM_UNKNOWN",
					},
				},
			},
		})

		require.ErrorContains(t, err, "unsupported chunking algorithm")
	})

	t.Run("hierarchical chunking with childChunkSize >= parentChunkSize returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"region":             "tor1",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "new",
				"dataSources": []any{
					map[string]any{
						"type":              "spaces",
						"spacesBucket":      "tor1/my-bucket",
						"chunkingAlgorithm": chunkingHierarchical,
						"parentChunkSize":   500,
						"childChunkSize":    500,
					},
				},
			},
		})

		require.ErrorContains(t, err, "childChunkSize")
		require.ErrorContains(t, err, "parentChunkSize")
	})

	// ── Valid configurations ───────────────────────────────────────────────

	t.Run("valid spaces source with new database -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"region":             "tor1",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "new",
				"dataSources":        []any{validSpacesSource},
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid spaces source with existing database -> no error (no region needed)", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "existing",
				"databaseId":         "abf1055a-745d-4c24-a1db-1959ea819264",
				"dataSources":        []any{validSpacesSource},
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid web seed source -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"region":             "tor1",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "new",
				"dataSources":        []any{validSeedSource},
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid web sitemap source -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"region":             "tor1",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "new",
				"dataSources": []any{
					map[string]any{
						"type":      "web",
						"crawlType": "sitemap",
						"webURL":    "https://example.com/sitemap.xml",
					},
				},
			},
		})

		require.NoError(t, err)
	})

	t.Run("multiple data sources of mixed types -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"region":             "tor1",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "new",
				"dataSources": []any{
					validSpacesSource,
					validSeedSource,
					map[string]any{
						"type":      "web",
						"crawlType": "sitemap",
						"webURL":    "https://example.com/sitemap.xml",
					},
				},
			},
		})

		require.NoError(t, err)
	})

	t.Run("hierarchical chunking with valid sizes -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-kb",
				"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"region":             "tor1",
				"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"databaseOption":     "new",
				"dataSources": []any{
					map[string]any{
						"type":              "spaces",
						"spacesBucket":      "tor1/my-bucket",
						"chunkingAlgorithm": chunkingHierarchical,
						"parentChunkSize":   1000,
						"childChunkSize":    350,
					},
				},
			},
		})

		require.NoError(t, err)
	})
}

func Test__CreateKnowledgeBase__Execute(t *testing.T) {
	component := &CreateKnowledgeBase{}

	kbResponse := `{
		"knowledge_base": {
			"uuid": "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
			"name": "my-kb",
			"region": "tor1",
			"embedding_model_uuid": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
			"project_id": "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
			"database_id": "",
			"tags": [],
			"created_at": "2025-01-01T00:00:00Z",
			"updated_at": "2025-01-01T00:00:00Z"
		}
	}`

	kbResponseWithDB := `{
		"knowledge_base": {
			"uuid": "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
			"name": "my-kb",
			"region": "tor1",
			"embedding_model_uuid": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
			"project_id": "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
			"database_id": "abf1055a-745d-4c24-a1db-1959ea819264",
			"tags": [],
			"created_at": "2025-01-01T00:00:00Z",
			"updated_at": "2025-01-01T00:00:00Z"
		}
	}`

	modelsResponse := `{"models": [{"uuid": "05700391-7aa8-11ef-bf8f-4e013e2ddde4", "name": "Multi QA MPNet Base Dot v1", "kb_min_chunk_size": 100, "kb_max_chunk_size": 512}]}`
	projectsResponse := `{"projects": [{"id": "37455431-84bd-4fa2-94cf-e8486f8f8c5e", "name": "My Project", "is_default": false}]}`
	databasesResponse := `{"databases": [{"id": "abf1055a-745d-4c24-a1db-1959ea819264", "name": "kb-search", "engine": "opensearch", "status": "online"}]}`

	// lookupResponses returns mock responses for the model and project name lookups
	// that resolveDisplayNames makes after a successful create.
	// For the "existing database" case, pass withDB=true to also mock the databases lookup.
	lookupResponses := func(withDB bool) []*http.Response {
		resps := []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(modelsResponse))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(projectsResponse))},
		}
		if withDB {
			resps = append(resps, &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(databasesResponse)),
			})
		}
		return resps
	}

	// executeKB runs Execute and returns the metadata context so tests can inspect stored state.
	executeKB := func(t *testing.T, httpContext *contexts.HTTPContext, config map[string]any) (*contexts.MetadataContext, *contexts.RequestContext, error) {
		t.Helper()
		metaCtx := &contexts.MetadataContext{}
		reqCtx := &contexts.RequestContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration:  config,
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Metadata:       metaCtx,
			Requests:       reqCtx,
		})
		return metaCtx, reqCtx, err
	}

	baseConfig := map[string]any{
		"name":               "my-kb",
		"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
		"region":             "tor1",
		"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
		"databaseOption":     "new",
		"dataSources": []any{
			map[string]any{
				"type":         "spaces",
				"spacesBucket": "tor1/my-bucket",
			},
		},
	}

	t.Run("creates KB and schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: append([]*http.Response{
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(kbResponse))},
			}, lookupResponses(false)...),
		}

		meta, req, err := executeKB(t, httpContext, baseConfig)

		require.NoError(t, err)
		assert.Equal(t, "poll", req.Action)
		assert.Equal(t, kbPollInterval, req.Duration)

		// KB UUID and output stored in metadata
		var stored kbMetadata
		require.NoError(t, mapstructure.Decode(meta.Get(), &stored))
		assert.Equal(t, "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4", stored.KBUUID)
		assert.Equal(t, "my-kb", stored.KBOutput["name"])
	})

	t.Run("existing database -> databaseId included, region omitted from request", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: append([]*http.Response{
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(kbResponseWithDB))},
			}, lookupResponses(true)...),
		}

		_, _, err := executeKB(t, httpContext, map[string]any{
			"name":               "my-kb",
			"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
			"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
			"databaseOption":     "existing",
			"databaseId":         "abf1055a-745d-4c24-a1db-1959ea819264",
			"dataSources": []any{
				map[string]any{"type": "spaces", "spacesBucket": "tor1/my-bucket"},
			},
		})

		require.NoError(t, err)
		var reqBody map[string]any
		require.NoError(t, json.Unmarshal(readCapturedBody(t, httpContext), &reqBody))
		assert.Equal(t, "abf1055a-745d-4c24-a1db-1959ea819264", reqBody["database_id"])
		assert.Empty(t, reqBody["region"], "region must not be sent when using an existing database")
	})

	t.Run("new database -> database_id omitted from request", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: append([]*http.Response{
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(kbResponse))},
			}, lookupResponses(false)...),
		}

		_, _, err := executeKB(t, httpContext, baseConfig)

		require.NoError(t, err)
		var reqBody map[string]any
		require.NoError(t, json.Unmarshal(readCapturedBody(t, httpContext), &reqBody))
		assert.Empty(t, reqBody["database_id"])
	})

	t.Run("multiple data sources -> all sent in request", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: append([]*http.Response{
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(kbResponse))},
			}, lookupResponses(false)...),
		}

		_, _, err := executeKB(t, httpContext, map[string]any{
			"name":               "my-kb",
			"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
			"region":             "tor1",
			"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
			"databaseOption":     "new",
			"dataSources": []any{
				map[string]any{"type": "spaces", "spacesBucket": "tor1/my-bucket"},
				map[string]any{"type": "web", "crawlType": "seed", "webURL": "https://docs.example.com", "crawlingOption": "SCOPED"},
			},
		})

		require.NoError(t, err)
		var reqBody map[string]any
		require.NoError(t, json.Unmarshal(readCapturedBody(t, httpContext), &reqBody))
		sources, ok := reqBody["datasources"].([]any)
		require.True(t, ok)
		assert.Len(t, sources, 2)
	})

	t.Run("chunking options are included in request", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: append([]*http.Response{
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(kbResponse))},
			}, lookupResponses(false)...),
		}

		_, _, err := executeKB(t, httpContext, map[string]any{
			"name":               "my-kb",
			"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
			"region":             "tor1",
			"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
			"databaseOption":     "new",
			"dataSources": []any{
				map[string]any{
					"type":              "spaces",
					"spacesBucket":      "tor1/my-bucket",
					"chunkingAlgorithm": chunkingHierarchical,
					"parentChunkSize":   1000,
					"childChunkSize":    350,
				},
			},
		})

		require.NoError(t, err)
		var reqBody map[string]any
		require.NoError(t, json.Unmarshal(readCapturedBody(t, httpContext), &reqBody))
		sources := reqBody["datasources"].([]any)
		require.Len(t, sources, 1)
		source := sources[0].(map[string]any)
		assert.Equal(t, chunkingHierarchical, source["chunking_algorithm"])
		opts := source["chunking_options"].(map[string]any)
		assert.Equal(t, float64(1000), opts["parent_chunk_size"])
		assert.Equal(t, float64(350), opts["child_chunk_size"])
	})

	t.Run("webIncludeNavLinks=false -> nav/header/footer are in exclude_tags", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: append([]*http.Response{
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(kbResponse))},
			}, lookupResponses(false)...),
		}

		_, _, err := executeKB(t, httpContext, map[string]any{
			"name":               "my-kb",
			"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
			"region":             "tor1",
			"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
			"databaseOption":     "new",
			"dataSources": []any{
				map[string]any{"type": "web", "crawlType": "seed", "webURL": "https://docs.example.com", "crawlingOption": "SCOPED", "webIncludeNavLinks": false},
			},
		})

		require.NoError(t, err)
		var reqBody map[string]any
		require.NoError(t, json.Unmarshal(readCapturedBody(t, httpContext), &reqBody))
		crawler := reqBody["datasources"].([]any)[0].(map[string]any)["web_crawler_data_source"].(map[string]any)
		excludeTags := toStringSlice(crawler["exclude_tags"].([]any))
		assert.Contains(t, excludeTags, "nav")
		assert.Contains(t, excludeTags, "header")
		assert.Contains(t, excludeTags, "footer")
		assert.Contains(t, excludeTags, "aside")
	})

	t.Run("webIncludeNavLinks=true -> nav/header/footer are NOT in exclude_tags", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: append([]*http.Response{
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(kbResponse))},
			}, lookupResponses(false)...),
		}

		_, _, err := executeKB(t, httpContext, map[string]any{
			"name":               "my-kb",
			"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
			"region":             "tor1",
			"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
			"databaseOption":     "new",
			"dataSources": []any{
				map[string]any{"type": "web", "crawlType": "seed", "webURL": "https://docs.example.com", "crawlingOption": "SCOPED", "webIncludeNavLinks": true},
			},
		})

		require.NoError(t, err)
		var reqBody map[string]any
		require.NoError(t, json.Unmarshal(readCapturedBody(t, httpContext), &reqBody))
		crawler := reqBody["datasources"].([]any)[0].(map[string]any)["web_crawler_data_source"].(map[string]any)
		excludeTags := toStringSlice(crawler["exclude_tags"].([]any))
		assert.NotContains(t, excludeTags, "nav")
		assert.NotContains(t, excludeTags, "header")
		assert.NotContains(t, excludeTags, "footer")
		assert.Contains(t, excludeTags, "script")
		assert.Contains(t, excludeTags, "style")
	})

	t.Run("sitemap URL sets crawling_option to SITEMAP", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: append([]*http.Response{
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(kbResponse))},
			}, lookupResponses(false)...),
		}

		_, _, err := executeKB(t, httpContext, map[string]any{
			"name":               "my-kb",
			"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
			"region":             "tor1",
			"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
			"databaseOption":     "new",
			"dataSources": []any{
				map[string]any{"type": "web", "crawlType": "sitemap", "webURL": "https://example.com/sitemap.xml"},
			},
		})

		require.NoError(t, err)
		var reqBody map[string]any
		require.NoError(t, json.Unmarshal(readCapturedBody(t, httpContext), &reqBody))
		crawler := reqBody["datasources"].([]any)[0].(map[string]any)["web_crawler_data_source"].(map[string]any)
		assert.Equal(t, "SITEMAP", crawler["crawling_option"])
		assert.Equal(t, "https://example.com/sitemap.xml", crawler["base_url"])
	})

	t.Run("display names are resolved and stored in metadata output", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: append([]*http.Response{
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(kbResponseWithDB))},
			}, lookupResponses(true)...),
		}

		meta, _, err := executeKB(t, httpContext, map[string]any{
			"name":               "my-kb",
			"embeddingModelUUID": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
			"region":             "tor1",
			"projectId":          "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
			"databaseOption":     "existing",
			"databaseId":         "abf1055a-745d-4c24-a1db-1959ea819264",
			"dataSources": []any{
				map[string]any{"type": "spaces", "spacesBucket": "tor1/my-bucket"},
			},
		})

		require.NoError(t, err)
		var stored kbMetadata
		require.NoError(t, mapstructure.Decode(meta.Get(), &stored))
		assert.Equal(t, "Multi QA MPNet Base Dot v1", stored.KBOutput["embeddingModelName"])
		assert.Equal(t, "My Project", stored.KBOutput["projectName"])
		assert.Equal(t, "kb-search", stored.KBOutput["databaseName"])
		assert.Equal(t, "online", stored.KBOutput["databaseStatus"])
	})

	t.Run("lookup failures do not block execution", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(kbResponse))},
				{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"id":"unauthorized","message":"forbidden"}`))},
				{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"id":"unauthorized","message":"forbidden"}`))},
			},
		}

		meta, _, err := executeKB(t, httpContext, baseConfig)

		require.NoError(t, err)
		var stored kbMetadata
		require.NoError(t, mapstructure.Decode(meta.Get(), &stored))
		assert.Nil(t, stored.KBOutput["embeddingModelName"])
		assert.Nil(t, stored.KBOutput["projectName"])
		assert.Equal(t, "05700391-7aa8-11ef-bf8f-4e013e2ddde4", stored.KBOutput["embeddingModelUUID"])
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnprocessableEntity,
					Body:       io.NopCloser(strings.NewReader(`{"id":"unprocessable_entity","message":"Name already in use"}`)),
				},
			},
		}

		_, _, err := executeKB(t, httpContext, baseConfig)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create knowledge base")
	})
}

func Test__CreateKnowledgeBase__Poll(t *testing.T) {
	component := &CreateKnowledgeBase{}

	storedOutput := map[string]any{
		"uuid": "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
		"name": "my-kb",
		"region": "tor1",
	}

	// buildPollCtx builds an ActionContext for the poll action with pre-set metadata.
	buildPollCtx := func(httpContext *contexts.HTTPContext) core.ActionContext {
		return core.ActionContext{
			Name: "poll",
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Metadata: &contexts.MetadataContext{Metadata: map[string]any{
				"kbUUID":   "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
				"kbOutput": storedOutput,
			}},
			Requests: &contexts.RequestContext{},
		}
	}

	// DO embeds the latest indexing job directly in the KB response as "last_indexing_job".
	// Status values use the "INDEX_JOB_STATUS_" prefix (confirmed from the real API).
	kbNoJob := `{"knowledge_base": {"uuid": "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4", "status": "active"}}`
	kbPendingJob := `{"knowledge_base": {"uuid": "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4", "status": "active", "last_indexing_job": {"uuid": "job-1", "status": "INDEX_JOB_STATUS_PENDING"}}}`
	kbRunningJob := `{"knowledge_base": {"uuid": "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4", "status": "active", "last_indexing_job": {"uuid": "job-1", "status": "INDEX_JOB_STATUS_RUNNING"}}}`
	kbCompletedJob := `{"knowledge_base": {"uuid": "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4", "status": "active", "last_indexing_job": {"uuid": "job-1", "status": "INDEX_JOB_STATUS_COMPLETED"}}}`
	kbFailedJob := `{"knowledge_base": {"uuid": "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4", "status": "active", "last_indexing_job": {"uuid": "job-1", "status": "INDEX_JOB_STATUS_FAILED"}}}`
	startJobResponse := `{"index_job": {"uuid": "job-1", "status": "INDEX_JOB_STATUS_PENDING"}}`

	kbProvisioningResponse := `{"knowledge_base": {"uuid": "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4", "status": "provisioning"}}`

	t.Run("no last_indexing_job and KB ready -> starts one and reschedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(kbNoJob))},
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(startJobResponse))},
			},
		}

		ctx := buildPollCtx(httpContext)
		err := component.HandleAction(ctx)

		require.NoError(t, err)
		assert.Equal(t, "poll", ctx.Requests.(*contexts.RequestContext).Action)
		// two requests: get KB, start job
		assert.Len(t, httpContext.Requests, 2)
	})

	t.Run("no last_indexing_job and KB still provisioning -> waits without starting a job", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(kbProvisioningResponse))},
			},
		}

		ctx := buildPollCtx(httpContext)
		err := component.HandleAction(ctx)

		require.NoError(t, err)
		assert.Equal(t, "poll", ctx.Requests.(*contexts.RequestContext).Action)
		// only one request: get KB — no start job call
		assert.Len(t, httpContext.Requests, 1)
	})

	t.Run("no last_indexing_job, KB ready, but start job fails -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(kbNoJob))},
				{StatusCode: http.StatusUnprocessableEntity, Body: io.NopCloser(strings.NewReader(`{"message":"quota exceeded"}`))},
			},
		}

		ctx := buildPollCtx(httpContext)
		err := component.HandleAction(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to start indexing job")
	})

	t.Run("last_indexing_job is pending -> reschedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(kbPendingJob))},
			},
		}

		ctx := buildPollCtx(httpContext)
		err := component.HandleAction(ctx)

		require.NoError(t, err)
		assert.Equal(t, "poll", ctx.Requests.(*contexts.RequestContext).Action)
		// only one request: get KB
		assert.Len(t, httpContext.Requests, 1)
	})

	t.Run("last_indexing_job is running -> reschedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(kbRunningJob))},
			},
		}

		ctx := buildPollCtx(httpContext)
		err := component.HandleAction(ctx)

		require.NoError(t, err)
		assert.Equal(t, "poll", ctx.Requests.(*contexts.RequestContext).Action)
		assert.Len(t, httpContext.Requests, 1)
	})

	t.Run("last_indexing_job is INDEX_JOB_STATUS_COMPLETED -> emits output", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(kbCompletedJob))},
			},
		}

		ctx := buildPollCtx(httpContext)
		err := component.HandleAction(ctx)

		require.NoError(t, err)
		execState := ctx.ExecutionState.(*contexts.ExecutionStateContext)
		assert.True(t, execState.Passed)
		assert.Equal(t, "digitalocean.knowledge_base.created", execState.Type)
		require.Len(t, execState.Payloads, 1)
		wrapped := execState.Payloads[0].(map[string]any)
		data := wrapped["data"].(map[string]any)
		assert.Equal(t, "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4", data["uuid"])
		assert.Equal(t, "my-kb", data["name"])
	})

	t.Run("last_indexing_job is failed -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(kbFailedJob))},
			},
		}

		ctx := buildPollCtx(httpContext)
		err := component.HandleAction(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "indexing job")
	})

	t.Run("already finished -> returns immediately without API calls", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		ctx := buildPollCtx(httpContext)
		ctx.ExecutionState = &contexts.ExecutionStateContext{Finished: true, KVs: map[string]string{}}

		err := component.HandleAction(ctx)

		require.NoError(t, err)
		assert.Empty(t, httpContext.Requests)
	})
}

func Test__CreateKnowledgeBase__Configuration(t *testing.T) {
	component := &CreateKnowledgeBase{}
	fields := component.Configuration()

	findField := func(name string) *configuration.Field {
		for _, f := range fields {
			if f.Name == name {
				return &f
			}
		}
		return nil
	}

	t.Run("has all top-level required fields", func(t *testing.T) {
		for _, name := range []string{"name", "embeddingModelUUID", "projectId"} {
			field := findField(name)
			require.NotNil(t, field, "%s field must exist", name)
			assert.True(t, field.Required, "%s must be required", name)
		}
	})

	t.Run("region is a select with tor1 as default, visible and required only for new database", func(t *testing.T) {
		field := findField("region")
		require.NotNil(t, field)
		assert.Equal(t, "select", field.Type)
		assert.Equal(t, "tor1", field.Default)
		assert.False(t, field.Required, "region must not be unconditionally required")
		require.NotNil(t, field.TypeOptions)
		require.NotNil(t, field.TypeOptions.Select)
		assert.NotEmpty(t, field.TypeOptions.Select.Options)
		require.Len(t, field.VisibilityConditions, 1)
		assert.Equal(t, "databaseOption", field.VisibilityConditions[0].Field)
		assert.Equal(t, []string{"new"}, field.VisibilityConditions[0].Values)
		require.Len(t, field.RequiredConditions, 1)
		assert.Equal(t, "databaseOption", field.RequiredConditions[0].Field)
		assert.Equal(t, []string{"new"}, field.RequiredConditions[0].Values)
	})

	t.Run("region appears after databaseId in configuration order", func(t *testing.T) {
		regionIdx, databaseIdIdx := -1, -1
		for i, f := range fields {
			switch f.Name {
			case "region":
				regionIdx = i
			case "databaseId":
				databaseIdIdx = i
			}
		}
		require.NotEqual(t, -1, regionIdx, "region field must exist")
		require.NotEqual(t, -1, databaseIdIdx, "databaseId field must exist")
		assert.Greater(t, regionIdx, databaseIdIdx, "region must appear after databaseId")
	})

	t.Run("embeddingModelUUID is an integration resource field", func(t *testing.T) {
		field := findField("embeddingModelUUID")
		require.NotNil(t, field)
		assert.Equal(t, "integration-resource", field.Type)
		assert.True(t, field.Required)
		require.NotNil(t, field.TypeOptions)
		require.NotNil(t, field.TypeOptions.Resource)
		assert.Equal(t, "embedding_model", field.TypeOptions.Resource.Type)
	})

	t.Run("projectId is an integration resource field", func(t *testing.T) {
		field := findField("projectId")
		require.NotNil(t, field)
		assert.Equal(t, "integration-resource", field.Type)
		assert.True(t, field.Required)
		require.NotNil(t, field.TypeOptions)
		require.NotNil(t, field.TypeOptions.Resource)
		assert.Equal(t, "project", field.TypeOptions.Resource.Type)
	})

	t.Run("databaseId is an integration resource field with visibility condition", func(t *testing.T) {
		field := findField("databaseId")
		require.NotNil(t, field)
		assert.Equal(t, "integration-resource", field.Type)
		require.NotNil(t, field.TypeOptions)
		require.NotNil(t, field.TypeOptions.Resource)
		assert.Equal(t, "opensearch_database", field.TypeOptions.Resource.Type)
		require.Len(t, field.VisibilityConditions, 1)
		assert.Equal(t, "databaseOption", field.VisibilityConditions[0].Field)
		assert.Equal(t, []string{"existing"}, field.VisibilityConditions[0].Values)
	})

	t.Run("tags is a togglable list of strings", func(t *testing.T) {
		field := findField("tags")
		require.NotNil(t, field)
		assert.Equal(t, "list", field.Type)
		assert.True(t, field.Togglable)
		require.NotNil(t, field.TypeOptions)
		require.NotNil(t, field.TypeOptions.List)
		require.NotNil(t, field.TypeOptions.List.ItemDefinition)
		assert.Equal(t, "string", field.TypeOptions.List.ItemDefinition.Type)
	})

	t.Run("databaseOption is a select with 'new' as default", func(t *testing.T) {
		field := findField("databaseOption")
		require.NotNil(t, field)
		assert.Equal(t, "select", field.Type)
		assert.Equal(t, "new", field.Default)
		require.NotNil(t, field.TypeOptions)
		require.NotNil(t, field.TypeOptions.Select)
		assert.Len(t, field.TypeOptions.Select.Options, 2)

		values := make([]string, len(field.TypeOptions.Select.Options))
		for i, opt := range field.TypeOptions.Select.Options {
			values[i] = opt.Value
		}
		assert.Contains(t, values, "new")
		assert.Contains(t, values, "existing")
	})

	t.Run("databaseId is hidden unless databaseOption is 'existing'", func(t *testing.T) {
		field := findField("databaseId")
		require.NotNil(t, field)
		require.Len(t, field.VisibilityConditions, 1)
		assert.Equal(t, "databaseOption", field.VisibilityConditions[0].Field)
		assert.Equal(t, []string{"existing"}, field.VisibilityConditions[0].Values)
		require.Len(t, field.RequiredConditions, 1)
		assert.Equal(t, "databaseOption", field.RequiredConditions[0].Field)
		assert.Equal(t, []string{"existing"}, field.RequiredConditions[0].Values)
	})

	t.Run("dataSources is a required list of objects", func(t *testing.T) {
		field := findField("dataSources")
		require.NotNil(t, field)
		assert.Equal(t, "list", field.Type)
		assert.True(t, field.Required)
		require.NotNil(t, field.TypeOptions)
		require.NotNil(t, field.TypeOptions.List)
		require.NotNil(t, field.TypeOptions.List.ItemDefinition)
		assert.Equal(t, "object", field.TypeOptions.List.ItemDefinition.Type)
		assert.NotEmpty(t, field.TypeOptions.List.ItemDefinition.Schema)
	})

	t.Run("data source schema has type selector with spaces and web options", func(t *testing.T) {
		dataSourcesField := findField("dataSources")
		require.NotNil(t, dataSourcesField)

		schema := dataSourcesField.TypeOptions.List.ItemDefinition.Schema

		findSchemaField := func(name string) *configuration.Field {
			for _, f := range schema {
				if f.Name == name {
					return &f
				}
			}
			return nil
		}

		typeField := findSchemaField("type")
		require.NotNil(t, typeField, "type field must exist in schema")
		assert.Equal(t, "select", typeField.Type)
		assert.True(t, typeField.Required)

		values := make([]string, len(typeField.TypeOptions.Select.Options))
		for i, opt := range typeField.TypeOptions.Select.Options {
			values[i] = opt.Value
		}
		assert.Contains(t, values, "spaces")
		assert.Contains(t, values, "web")
	})

	t.Run("spacesBucket is an integration resource field conditional on type=spaces", func(t *testing.T) {
		dataSourcesField := findField("dataSources")
		require.NotNil(t, dataSourcesField)
		schema := dataSourcesField.TypeOptions.List.ItemDefinition.Schema

		var f *configuration.Field
		for _, s := range schema {
			if s.Name == "spacesBucket" {
				f = &s
				break
			}
		}

		require.NotNil(t, f)
		assert.Equal(t, "integration-resource", f.Type)
		require.NotNil(t, f.TypeOptions)
		require.NotNil(t, f.TypeOptions.Resource)
		assert.Equal(t, "spaces_bucket", f.TypeOptions.Resource.Type)
		require.Len(t, f.VisibilityConditions, 1)
		assert.Equal(t, "type", f.VisibilityConditions[0].Field)
		assert.Equal(t, []string{"spaces"}, f.VisibilityConditions[0].Values)
		require.Len(t, f.RequiredConditions, 1)
		assert.Equal(t, "type", f.RequiredConditions[0].Field)
	})

	t.Run("webIncludeNavLinks is a bool field visible only for web sources", func(t *testing.T) {
		dataSourcesField := findField("dataSources")
		require.NotNil(t, dataSourcesField)
		schema := dataSourcesField.TypeOptions.List.ItemDefinition.Schema

		var navLinksField *configuration.Field
		for _, f := range schema {
			if f.Name == "webIncludeNavLinks" {
				navLinksField = &f
				break
			}
		}

		require.NotNil(t, navLinksField)
		assert.Equal(t, "boolean", navLinksField.Type)
		assert.Equal(t, false, navLinksField.Default)
		assert.False(t, navLinksField.Required)
		require.Len(t, navLinksField.VisibilityConditions, 1)
		assert.Equal(t, "type", navLinksField.VisibilityConditions[0].Field)
		assert.Equal(t, []string{"web"}, navLinksField.VisibilityConditions[0].Values)
	})

	t.Run("web fields are conditional on type=web", func(t *testing.T) {
		dataSourcesField := findField("dataSources")
		require.NotNil(t, dataSourcesField)
		schema := dataSourcesField.TypeOptions.List.ItemDefinition.Schema

		findSchemaField := func(name string) *configuration.Field {
			for _, f := range schema {
				if f.Name == name {
					return &f
				}
			}
			return nil
		}

		for _, name := range []string{"crawlType", "webURL"} {
			f := findSchemaField(name)
			require.NotNil(t, f, "%s must exist in schema", name)
			require.NotEmpty(t, f.VisibilityConditions)
			hasWebCondition := false
			for _, vc := range f.VisibilityConditions {
				if vc.Field == "type" && len(vc.Values) == 1 && vc.Values[0] == "web" {
					hasWebCondition = true
				}
			}
			assert.True(t, hasWebCondition, "%s should be conditional on type=web", name)
		}
	})

	t.Run("crawlingOption is conditional on type=web AND crawlType=seed", func(t *testing.T) {
		dataSourcesField := findField("dataSources")
		require.NotNil(t, dataSourcesField)
		schema := dataSourcesField.TypeOptions.List.ItemDefinition.Schema

		var crawlingOptionField *configuration.Field
		for _, f := range schema {
			if f.Name == "crawlingOption" {
				crawlingOptionField = &f
				break
			}
		}

		require.NotNil(t, crawlingOptionField)
		require.Len(t, crawlingOptionField.VisibilityConditions, 2)

		fieldNames := make(map[string][]string)
		for _, vc := range crawlingOptionField.VisibilityConditions {
			fieldNames[vc.Field] = vc.Values
		}
		assert.Contains(t, fieldNames, "type")
		assert.Equal(t, []string{"web"}, fieldNames["type"])
		assert.Contains(t, fieldNames, "crawlType")
		assert.Equal(t, []string{"seed"}, fieldNames["crawlType"])
	})

	t.Run("chunking algorithm has four options with section-based as default", func(t *testing.T) {
		dataSourcesField := findField("dataSources")
		require.NotNil(t, dataSourcesField)
		schema := dataSourcesField.TypeOptions.List.ItemDefinition.Schema

		var chunkingField *configuration.Field
		for _, f := range schema {
			if f.Name == "chunkingAlgorithm" {
				chunkingField = &f
				break
			}
		}

		require.NotNil(t, chunkingField)
		assert.Equal(t, chunkingSectionBased, chunkingField.Default)
		require.NotNil(t, chunkingField.TypeOptions)
		require.NotNil(t, chunkingField.TypeOptions.Select)
		assert.Len(t, chunkingField.TypeOptions.Select.Options, 4)
	})

	t.Run("hierarchical chunk size fields are conditional on chunkingAlgorithm=hierarchical", func(t *testing.T) {
		dataSourcesField := findField("dataSources")
		require.NotNil(t, dataSourcesField)
		schema := dataSourcesField.TypeOptions.List.ItemDefinition.Schema

		findSchemaField := func(name string) *configuration.Field {
			for _, f := range schema {
				if f.Name == name {
					return &f
				}
			}
			return nil
		}

		for _, name := range []string{"parentChunkSize", "childChunkSize"} {
			f := findSchemaField(name)
			require.NotNil(t, f, "%s must exist in schema", name)
			assert.True(t, f.Togglable)
			require.Len(t, f.VisibilityConditions, 1)
			assert.Equal(t, "chunkingAlgorithm", f.VisibilityConditions[0].Field)
			assert.Equal(t, []string{chunkingHierarchical}, f.VisibilityConditions[0].Values)
		}
	})

	t.Run("semanticThreshold is conditional on chunkingAlgorithm=semantic", func(t *testing.T) {
		dataSourcesField := findField("dataSources")
		require.NotNil(t, dataSourcesField)
		schema := dataSourcesField.TypeOptions.List.ItemDefinition.Schema

		var semanticField *configuration.Field
		for _, f := range schema {
			if f.Name == "semanticThreshold" {
				semanticField = &f
				break
			}
		}

		require.NotNil(t, semanticField)
		assert.True(t, semanticField.Togglable)
		require.Len(t, semanticField.VisibilityConditions, 1)
		assert.Equal(t, "chunkingAlgorithm", semanticField.VisibilityConditions[0].Field)
		assert.Equal(t, []string{chunkingSemantic}, semanticField.VisibilityConditions[0].Values)
	})
}
