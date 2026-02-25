package daytona

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

func Test__Daytona__Sync(t *testing.T) {
	d := &Daytona{}

	t.Run("no apiKey -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "apiKey is required")
	})

	t.Run("successful connection test -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/sandbox")
		assert.Equal(t, "Bearer test-api-key", httpContext.Requests[0].Header.Get("Authorization"))
	})

	t.Run("connection test failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"message":"unauthorized"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "invalid-key",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", appCtx.State)
	})

	t.Run("custom baseURL is used", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":  "test-api-key",
				"baseURL": "https://custom.daytona.io/api",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "https://custom.daytona.io/api/sandbox")
	})
}

func Test__Daytona__Metadata(t *testing.T) {
	t.Run("metadata is set on successful sync", func(t *testing.T) {
		d := &Daytona{}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.NoError(t, err)
		assert.NotNil(t, appCtx.Metadata)
		_, ok := appCtx.Metadata.(Metadata)
		assert.True(t, ok, "metadata should be of type Metadata")
	})
}

func Test__Daytona__ListResources(t *testing.T) {
	d := &Daytona{}

	t.Run("list snapshot resources", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"items":[{"id":"snap-1","name":"default"},{"id":"snap-2","name":"daytona-small"}]}`,
					)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		resources, err := d.ListResources("snapshot", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: appCtx,
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "snapshot", resources[0].Type)
		assert.Equal(t, "default", resources[0].Name)
		assert.Equal(t, "snap-1", resources[0].ID)
	})

	t.Run("list sandbox resources", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"id":"sandbox-123","state":"started"},{"id":"sandbox-456","state":"stopped"}]`,
					)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		resources, err := d.ListResources("sandbox", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: appCtx,
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "sandbox", resources[0].Type)
		assert.Equal(t, "sandbox-123", resources[0].Name)
		assert.Equal(t, "sandbox-123", resources[0].ID)
	})

	t.Run("unknown resource type returns empty list", func(t *testing.T) {
		resources, err := d.ListResources("unknown", core.ListResourcesContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})
}
