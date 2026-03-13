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
