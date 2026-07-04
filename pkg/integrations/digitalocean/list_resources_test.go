package digitalocean

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ListResources__Droplets(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("successful droplet listing -> returns resources", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"droplets": [
							{
								"id": 12345678,
								"name": "web-server-01",
								"status": "active"
							},
							{
								"id": 87654321,
								"name": "db-server-01",
								"status": "active"
							}
						],
						"links": {},
						"meta": {"total": 2}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		resources, err := integration.ListResources("droplet", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "droplet", resources[0].Type)
		assert.Equal(t, "web-server-01", resources[0].Name)
		assert.Equal(t, "12345678", resources[0].ID)
		assert.Equal(t, "db-server-01", resources[1].Name)
		assert.Equal(t, "87654321", resources[1].ID)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"id":"server_error","message":"Internal server error"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		resources, err := integration.ListResources("droplet", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error listing droplets")
		assert.Nil(t, resources)
	})

	t.Run("empty droplet list -> returns empty array", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"droplets": [],
						"links": {},
						"meta": {"total": 0}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		resources, err := integration.ListResources("droplet", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 0)
	})
}

func Test__ListResources__Regions(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("successful region listing -> returns only available regions", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"regions": [
							{
								"name": "New York 3",
								"slug": "nyc3",
								"available": true
							},
							{
								"name": "San Francisco 3",
								"slug": "sfo3",
								"available": true
							},
							{
								"name": "Amsterdam 1",
								"slug": "ams1",
								"available": false
							}
						],
						"links": {},
						"meta": {"total": 3}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		resources, err := integration.ListResources("region", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "region", resources[0].Type)
		assert.Equal(t, "New York 3", resources[0].Name)
		assert.Equal(t, "nyc3", resources[0].ID)
		assert.Equal(t, "San Francisco 3", resources[1].Name)
		assert.Equal(t, "sfo3", resources[1].ID)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"id":"unauthorized","message":"Unable to authenticate you"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "invalid-token",
			},
		}

		resources, err := integration.ListResources("region", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error listing regions")
		assert.Nil(t, resources)
	})
}

func Test__ListResources__Sizes(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("successful size listing -> returns only available sizes", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"sizes": [
							{
								"slug": "s-1vcpu-1gb",
								"available": true,
								"memory": 1024,
								"vcpus": 1
							},
							{
								"slug": "s-2vcpu-2gb",
								"available": true,
								"memory": 2048,
								"vcpus": 2
							},
							{
								"slug": "s-4vcpu-8gb",
								"available": false,
								"memory": 8192,
								"vcpus": 4
							}
						],
						"links": {},
						"meta": {"total": 3}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		resources, err := integration.ListResources("size", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "size", resources[0].Type)
		assert.Equal(t, "s-1vcpu-1gb", resources[0].Name)
		assert.Equal(t, "s-1vcpu-1gb", resources[0].ID)
		assert.Equal(t, "s-2vcpu-2gb", resources[1].Name)
		assert.Equal(t, "s-2vcpu-2gb", resources[1].ID)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusServiceUnavailable,
					Body:       io.NopCloser(strings.NewReader(`{"id":"service_unavailable","message":"Service temporarily unavailable"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		resources, err := integration.ListResources("size", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error listing sizes")
		assert.Nil(t, resources)
	})
}

func Test__ListResources__DatabaseClusters(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("successful cluster listing returns resources", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"databases": [
							{"id": "cluster-1", "name": "superplane-db", "engine": "pg"},
							{"id": "cluster-2", "name": "analytics-db", "engine": "mysql"}
						]
					}`)),
				},
			},
		}

		resources, err := integration.ListResources("database_cluster", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "database_cluster", resources[0].Type)
		assert.Equal(t, "superplane-db", resources[0].Name)
		assert.Equal(t, "cluster-1", resources[0].ID)
	})

	t.Run("null database list returns empty resources", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"databases": null
					}`)),
				},
			},
		}

		resources, err := integration.ListResources("database_cluster", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})
}

func Test__ListResources__Databases(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("missing cluster returns empty resources", func(t *testing.T) {
		resources, err := integration.ListResources("database", core.ListResourcesContext{
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
			Parameters:  map[string]string{},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("successful database listing returns resources", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"dbs": [
							{"name":"app_db"},
							{"name":"analytics_db"}
						]
					}`)),
				},
			},
		}

		resources, err := integration.ListResources("database", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
			Parameters: map[string]string{
				"databaseCluster": "cluster-1",
			},
		})

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "database", resources[0].Type)
		assert.Equal(t, "app_db", resources[0].Name)
		assert.Equal(t, "app_db", resources[0].ID)
	})
}

func Test__ListResources__DatabaseClusterVersions(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("missing engine returns empty resources", func(t *testing.T) {
		resources, err := integration.ListResources("database_cluster_version", core.ListResourcesContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("successful version listing returns versions for engine", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"options": {
							"pg": {
								"versions": ["14", "15", "16", "18"],
								"layouts": []
							},
							"mysql": {
								"versions": ["8"],
								"layouts": []
							}
						}
					}`)),
				},
			},
		}

		resources, err := integration.ListResources("database_cluster_version", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
			Parameters:  map[string]string{"engine": "pg"},
		})

		require.NoError(t, err)
		require.Len(t, resources, 4)
		assert.Equal(t, "14", resources[0].ID)
		assert.Equal(t, "18", resources[3].Name)
	})
}

func Test__ListResources__DatabaseClusterSizes(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("missing parameters returns empty resources", func(t *testing.T) {
		resources, err := integration.ListResources("database_cluster_size", core.ListResourcesContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
			Parameters:  map[string]string{"engine": "pg"},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("successful size listing returns sizes for engine and node count", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"options": {
							"pg": {
								"versions": ["18"],
								"layouts": [
									{"num_nodes": 1, "sizes": ["db-s-1vcpu-1gb", "db-s-1vcpu-2gb"]},
									{"num_nodes": 2, "sizes": ["db-s-1vcpu-2gb", "db-s-2vcpu-4gb"]}
								]
							}
						}
					}`)),
				},
			},
		}

		resources, err := integration.ListResources("database_cluster_size", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
			Parameters:  map[string]string{"engine": "pg", "numNodes": "2"},
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "db-s-1vcpu-2gb", resources[0].ID)
		assert.Equal(t, "db-s-2vcpu-4gb", resources[1].Name)
	})
}

func Test__ListResources__Images(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("successful image listing -> returns resources with formatted names", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"images": [
							{
								"id": 12345,
								"name": "24.04 (LTS) x64",
								"distribution": "Ubuntu",
								"slug": "ubuntu-24-04-x64"
							},
							{
								"id": 67890,
								"name": "12 x64",
								"distribution": "Debian",
								"slug": "debian-12-x64"
							},
							{
								"id": 11111,
								"name": "Custom Image",
								"distribution": "",
								"slug": "custom-image"
							}
						],
						"links": {},
						"meta": {"total": 3}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		resources, err := integration.ListResources("image", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 3)
		assert.Equal(t, "image", resources[0].Type)
		assert.Equal(t, "Ubuntu 24.04 (LTS) x64", resources[0].Name)
		assert.Equal(t, "ubuntu-24-04-x64", resources[0].ID)
		assert.Equal(t, "Debian 12 x64", resources[1].Name)
		assert.Equal(t, "debian-12-x64", resources[1].ID)
		assert.Equal(t, "Custom Image", resources[2].Name)
		assert.Equal(t, "custom-image", resources[2].ID)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusTooManyRequests,
					Body:       io.NopCloser(strings.NewReader(`{"id":"too_many_requests","message":"Rate limit exceeded"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		resources, err := integration.ListResources("image", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error listing images")
		assert.Nil(t, resources)
	})
}

func Test__ListResources__UnknownResourceType(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("unknown resource type -> returns empty array", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		resources, err := integration.ListResources("unknown", core.ListResourcesContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 0)
	})
}

func Test__ListResources__Snapshots(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("successful snapshot listing -> returns resources", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"snapshots": [
							{
								"id": "12345678",
								"name": "my-droplet-snapshot",
								"created_at": "2024-06-15T10:30:00Z",
								"resource_id": "98765432",
								"resource_type": "droplet",
								"regions": ["nyc3"],
								"min_disk_size": 25,
								"size_gigabytes": 2.36
							},
							{
								"id": "87654321",
								"name": "backup-snapshot",
								"created_at": "2024-06-14T08:00:00Z",
								"resource_id": "11111111",
								"resource_type": "droplet",
								"regions": ["sfo3"],
								"min_disk_size": 50,
								"size_gigabytes": 5.12
							}
						]
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		resources, err := integration.ListResources("snapshot", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "snapshot", resources[0].Type)
		assert.Equal(t, "my-droplet-snapshot", resources[0].Name)
		assert.Equal(t, "12345678", resources[0].ID)
		assert.Equal(t, "backup-snapshot", resources[1].Name)
		assert.Equal(t, "87654321", resources[1].ID)
	})

	t.Run("empty snapshot list -> returns empty array", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"snapshots": []}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		resources, err := integration.ListResources("snapshot", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 0)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"id":"server_error","message":"Internal server error"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		resources, err := integration.ListResources("snapshot", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error listing snapshots")
		assert.Nil(t, resources)
	})
}

func Test__ListResources__EmbeddingModels(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("successful listing -> returns embedding models", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"models": [
							{"uuid": "05700391-7aa8-11ef-bf8f-4e013e2ddde4", "name": "Multi QA MPNet Base Dot v1", "kb_min_chunk_size": 100, "kb_max_chunk_size": 512},
							{"uuid": "c4e7f3a1-9bb2-11ef-bf8f-4e013e2ddde4", "name": "GTE Large EN v1.5", "kb_min_chunk_size": 100, "kb_max_chunk_size": 1024},
							{"uuid": "d5f8a4b2-1cc3-11f0-bf8f-4e013e2ddde4", "name": "All MiniLM L6 v2", "kb_min_chunk_size": 0, "kb_max_chunk_size": 0}
						]
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		resources, err := integration.ListResources("embedding_model", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 3)
		assert.Equal(t, "embedding_model", resources[0].Type)
		assert.Equal(t, "Multi QA MPNet Base Dot v1 (100–512 tokens)", resources[0].Name)
		assert.Equal(t, "05700391-7aa8-11ef-bf8f-4e013e2ddde4", resources[0].ID)
		assert.Equal(t, "GTE Large EN v1.5 (100–1024 tokens)", resources[1].Name)
		// model without chunk size info -> name shown as-is
		assert.Equal(t, "All MiniLM L6 v2", resources[2].Name)
	})

	t.Run("verifies usecases=embedding query parameter is used", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"models": []}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		_, err := integration.ListResources("embedding_model", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.RawQuery, "usecases=MODEL_USECASE_KNOWLEDGEBASE")
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"id":"server_error","message":"Internal server error"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		resources, err := integration.ListResources("embedding_model", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error listing embedding models")
		assert.Nil(t, resources)
	})
}

func Test__ListResources__Projects(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("successful listing -> returns projects, default project marked", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"projects": [
							{"id": "37455431-84bd-4fa2-94cf-e8486f8f8c5e", "name": "My Project", "is_default": false},
							{"id": "aabbccdd-1234-5678-abcd-ef0123456789", "name": "Default Project", "is_default": true}
						]
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		resources, err := integration.ListResources("project", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "project", resources[0].Type)
		assert.Equal(t, "My Project", resources[0].Name)
		assert.Equal(t, "37455431-84bd-4fa2-94cf-e8486f8f8c5e", resources[0].ID)
		assert.Equal(t, "Default Project (default)", resources[1].Name)
		assert.Equal(t, "aabbccdd-1234-5678-abcd-ef0123456789", resources[1].ID)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"id":"unauthorized","message":"Unable to authenticate you"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		resources, err := integration.ListResources("project", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error listing projects")
		assert.Nil(t, resources)
	})
}

func Test__ListResources__OpenSearchDatabases(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("successful listing -> returns OpenSearch databases", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"databases": [
							{"id": "abf1055a-745d-4c24-a1db-1959ea819264", "name": "kb-search", "engine": "opensearch", "status": "online"},
							{"id": "ccdd1234-745d-4c24-a1db-1959ea819264", "name": "kb-search-2", "engine": "opensearch", "status": "online"}
						]
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		resources, err := integration.ListResources("opensearch_database", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "opensearch_database", resources[0].Type)
		assert.Equal(t, "kb-search", resources[0].Name)
		assert.Equal(t, "abf1055a-745d-4c24-a1db-1959ea819264", resources[0].ID)
		assert.Equal(t, "kb-search-2", resources[1].Name)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"id":"server_error","message":"Internal server error"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		resources, err := integration.ListResources("opensearch_database", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error listing OpenSearch databases")
		assert.Nil(t, resources)
	})
}

func Test__ListResources__AgentAvailableKnowledgeBases(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("missing agentId -> returns empty list", func(t *testing.T) {
		resources, err := integration.ListResources("agent_available_knowledge_base", core.ListResourcesContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
			Parameters:  map[string]string{},
		})

		require.NoError(t, err)
		assert.Len(t, resources, 0)
	})

	t.Run("filters out already attached knowledge bases", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					// GetAgent
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"agent": {
							"uuid": "agent-uuid",
							"name": "my-agent",
							"knowledge_bases": [
								{"uuid": "kb-attached-uuid", "name": "attached-kb"}
							]
						}
					}`)),
				},
				{
					// ListKnowledgeBases
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"knowledge_bases": [
							{"uuid": "kb-attached-uuid", "name": "attached-kb"},
							{"uuid": "kb-available-uuid", "name": "available-kb"}
						]
					}`)),
				},
			},
		}

		resources, err := integration.ListResources("agent_available_knowledge_base", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
			Parameters:  map[string]string{"agent": "agent-uuid"},
		})

		require.NoError(t, err)
		assert.Len(t, resources, 1)
		assert.Equal(t, "agent_available_knowledge_base", resources[0].Type)
		assert.Equal(t, "available-kb", resources[0].Name)
		assert.Equal(t, "kb-available-uuid", resources[0].ID)
	})

	t.Run("all knowledge bases already attached -> returns empty list", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					// GetAgent
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"agent": {
							"uuid": "agent-uuid",
							"name": "my-agent",
							"knowledge_bases": [
								{"uuid": "kb-uuid", "name": "my-kb"}
							]
						}
					}`)),
				},
				{
					// ListKnowledgeBases
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"knowledge_bases": [
							{"uuid": "kb-uuid", "name": "my-kb"}
						]
					}`)),
				},
			},
		}

		resources, err := integration.ListResources("agent_available_knowledge_base", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
			Parameters:  map[string]string{"agent": "agent-uuid"},
		})

		require.NoError(t, err)
		assert.Len(t, resources, 0)
	})
}

func Test__ListResources__ClientCreationError(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("missing API token -> returns error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{},
		}

		resources, err := integration.ListResources("droplet", core.ListResourcesContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create client")
		assert.Nil(t, resources)
	})
}

func Test__ListResources__SpacesBuckets(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("successful spaces bucket listing -> checks all regions", func(t *testing.T) {
		responses := make([]*http.Response, 0, len(allSpacesRegions))
		for _, region := range allSpacesRegions {
			body := `<?xml version="1.0" encoding="UTF-8"?><ListAllMyBucketsResult><Buckets></Buckets></ListAllMyBucketsResult>`
			if region == "nyc1" {
				body = `<?xml version="1.0" encoding="UTF-8"?><ListAllMyBucketsResult><Buckets><Bucket><Name>alpha-bucket</Name></Bucket></Buckets></ListAllMyBucketsResult>`
			}
			if region == "ric1" {
				body = `<?xml version="1.0" encoding="UTF-8"?><ListAllMyBucketsResult><Buckets><Bucket><Name>omega-bucket</Name></Bucket></Buckets></ListAllMyBucketsResult>`
			}

			responses = append(responses, &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
			})
		}

		httpContext := &contexts.HTTPContext{Responses: responses}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"spacesAccessKey": "test-access-key",
				"spacesSecretKey": "test-secret-key",
			},
		}

		resources, err := integration.ListResources("spaces_bucket", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, len(allSpacesRegions))
		assert.Len(t, resources, 2)
		assert.Equal(t, "spaces_bucket", resources[0].Type)
		assert.Equal(t, "alpha-bucket (nyc1)", resources[0].Name)
		assert.Equal(t, "nyc1/alpha-bucket", resources[0].ID)
		assert.Equal(t, "omega-bucket (ric1)", resources[1].Name)
		assert.Equal(t, "ric1/omega-bucket", resources[1].ID)
	})

	t.Run("spaces API error -> returns error", func(t *testing.T) {
		responses := make([]*http.Response, 0, len(allSpacesRegions))
		for range allSpacesRegions {
			responses = append(responses, &http.Response{
				StatusCode: http.StatusForbidden,
				Body:       io.NopCloser(strings.NewReader(`<Error><Code>SignatureDoesNotMatch</Code></Error>`)),
			})
		}

		httpContext := &contexts.HTTPContext{Responses: responses}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"spacesAccessKey": "invalid-access-key",
				"spacesSecretKey": "invalid-secret-key",
			},
		}

		resources, err := integration.ListResources("spaces_bucket", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error listing spaces buckets")
		assert.Contains(t, err.Error(), "region nyc1")
		assert.Nil(t, resources)
	})
}

func Test__ListResources__GPUDroplets(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("successful GPU droplet listing -> returns resources", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"droplets": [
							{"id": 11111111, "name": "gpu-node-1", "status": "active", "size_slug": "gpu-h100x1-80gb"},
							{"id": 22222222, "name": "gpu-node-2", "status": "active", "size_slug": "gpu-h100x1-80gb"}
						],
						"links": {},
						"meta": {"total": 2}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		resources, err := integration.ListResources("gpu_droplet", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "gpu_droplet", resources[0].Type)
		assert.Equal(t, "gpu-node-1", resources[0].Name)
		assert.Equal(t, "11111111", resources[0].ID)
		assert.Equal(t, "gpu-node-2", resources[1].Name)
		assert.Equal(t, "22222222", resources[1].ID)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"id":"unauthorized","message":"Unable to authenticate you"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "bad-token"},
		}

		resources, err := integration.ListResources("gpu_droplet", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.Error(t, err)
		assert.Nil(t, resources)
	})
}

func Test__ListResources__GPUSizes(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("successful GPU size listing -> returns only GPU sizes", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"sizes": [
							{"slug": "gpu-h100x1-80gb", "memory": 245760, "vcpus": 20, "disk": 480, "transfer": 10.0, "price_monthly": 4896.00, "available": true},
							{"slug": "gpu-h100x8-640gb", "memory": 1966080, "vcpus": 160, "disk": 3840, "transfer": 10.0, "price_monthly": 39168.00, "available": true},
							{"slug": "s-1vcpu-1gb", "memory": 1024, "vcpus": 1, "disk": 25, "transfer": 1.0, "price_monthly": 6.00, "available": true}
						],
						"links": {},
						"meta": {"total": 3}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		resources, err := integration.ListResources("gpu_size", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		// Only GPU sizes (slug prefixed with "gpu-") should be returned
		assert.Len(t, resources, 2)
		assert.Equal(t, "gpu_size", resources[0].Type)
		assert.Equal(t, "gpu-h100x1-80gb", resources[0].ID)
		assert.Equal(t, "gpu-h100x8-640gb", resources[1].ID)
	})
}

func Test__ListResources__GPURegions(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("successful GPU region listing -> returns available GPU-capable regions", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"regions": [
							{
								"name": "New York 3",
								"slug": "nyc3",
								"available": true,
								"sizes": ["gpu-h100x1-80gb", "s-1vcpu-1gb"]
							},
							{
								"name": "San Francisco 3",
								"slug": "sfo3",
								"available": true,
								"sizes": ["s-2vcpu-2gb"]
							},
							{
								"name": "Toronto",
								"slug": "tor1",
								"available": false,
								"sizes": ["gpu-h100x1-80gb"]
							}
						],
						"links": {},
						"meta": {"total": 3}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		resources, err := integration.ListResources("gpu_region", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		// Only available regions with GPU sizes should be returned
		for _, r := range resources {
			assert.Equal(t, "gpu_region", r.Type)
		}
	})
}

func Test__ListResources__GPUBaseImages(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("returns only GPU-supported distribution images", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"images": [
							{"id": 1, "name": "Ubuntu 22.04 (LTS) x64", "slug": "ubuntu-22-04-x64", "type": "distribution", "distribution": "Ubuntu"},
							{"id": 2, "name": "Ubuntu 24.04 (LTS) x64", "slug": "ubuntu-24-04-x64", "type": "distribution", "distribution": "Ubuntu"},
							{"id": 3, "name": "Ubuntu 20.04 (LTS) x64", "slug": "ubuntu-20-04-x64", "type": "distribution", "distribution": "Ubuntu"},
							{"id": 4, "name": "Debian 11 x64", "slug": "debian-11-x64", "type": "distribution", "distribution": "Debian"},
							{"id": 5, "name": "Debian 10 x64", "slug": "debian-10-x64", "type": "distribution", "distribution": "Debian"},
							{"id": 6, "name": "Rocky Linux 8 x64", "slug": "rockylinux-8-x64", "type": "distribution", "distribution": "Rocky Linux"},
							{"id": 7, "name": "Fedora 43 x64", "slug": "fedora-43-x64", "type": "distribution", "distribution": "Fedora"},
							{"id": 8, "name": "CentOS 7 x64", "slug": "centos-7-x64", "type": "distribution", "distribution": "CentOS"}
						],
						"links": {},
						"meta": {"total": 8}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		resources, err := integration.ListResources("base_gpu_image", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		// ubuntu-22-04, ubuntu-24-04, debian-11, rockylinux-8, fedora-43 pass; ubuntu-20-04, debian-10, centos-7 do not
		ids := make([]string, len(resources))
		for i, r := range resources {
			ids[i] = r.ID
			assert.Equal(t, "base_gpu_image", r.Type)
		}
		assert.Contains(t, ids, "ubuntu-22-04-x64")
		assert.Contains(t, ids, "ubuntu-24-04-x64")
		assert.Contains(t, ids, "debian-11-x64")
		assert.Contains(t, ids, "rockylinux-8-x64")
		assert.Contains(t, ids, "fedora-43-x64")
		assert.NotContains(t, ids, "ubuntu-20-04-x64")
		assert.NotContains(t, ids, "debian-10-x64")
		assert.NotContains(t, ids, "centos-7-x64")
	})
}

func Test__ListResources__GPUOneClickImages(t *testing.T) {
	integration := &DigitalOcean{}

	t.Run("returns only GPU-related application images", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"images": [
							{"id": 101, "name": "ML-in-a-Box", "slug": "ml-in-a-box", "type": "application", "distribution": ""},
							{"id": 102, "name": "NVIDIA CUDA Toolkit", "slug": "nvidia-cuda-toolkit", "type": "application", "distribution": ""},
							{"id": 103, "name": "PyTorch on Ubuntu", "slug": "pytorch-ubuntu", "type": "application", "distribution": ""},
							{"id": 104, "name": "WordPress on Ubuntu", "slug": "wordpress-ubuntu", "type": "application", "distribution": ""},
							{"id": 105, "name": "LAMP Stack", "slug": "lamp-stack", "type": "application", "distribution": ""}
						],
						"links": {},
						"meta": {"total": 5}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		resources, err := integration.ListResources("one_click_gpu_image", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		ids := make([]string, len(resources))
		for i, r := range resources {
			ids[i] = r.ID
			assert.Equal(t, "one_click_gpu_image", r.Type)
		}
		// GPU-related images should be included
		assert.Contains(t, ids, "ml-in-a-box")
		assert.Contains(t, ids, "nvidia-cuda-toolkit")
		assert.Contains(t, ids, "pytorch-ubuntu")
		// Non-GPU marketplace apps should be excluded
		assert.NotContains(t, ids, "wordpress-ubuntu")
		assert.NotContains(t, ids, "lamp-stack")
	})
}
