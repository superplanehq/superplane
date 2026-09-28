package datadog

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

func TestFacetValues(t *testing.T) {
	t.Run("reads span aggregate buckets", func(t *testing.T) {
		names, err := facetValues([]byte(`{
			"data": [
				{"type": "bucket", "attributes": {"by": {"env": "prod"}}},
				{"type": "bucket", "attributes": {"by": {"env": "staging"}}},
				{"type": "bucket", "attributes": {"by": {"env": "prod"}}}
			]
		}`), "env")
		require.NoError(t, err)
		assert.Equal(t, []string{"prod", "staging"}, names)
	})

	t.Run("reads log aggregate buckets", func(t *testing.T) {
		names, err := facetValues([]byte(`{
			"data": {
				"buckets": [
					{"by": {"env": "dev"}},
					{"by": {"env": ""}}
				]
			}
		}`), "env")
		require.NoError(t, err)
		assert.Equal(t, []string{"dev"}, names)
	})
}

func TestServiceResources_DeduplicatesAndSorts(t *testing.T) {
	resources := serviceResources([]ErrorTrackingIssue{
		{ID: "1", Service: "checkout"},
		{ID: "2", Service: "billing"},
		{ID: "3", Service: "checkout"},
		{ID: "4", Service: "  "},
		{ID: "5", Service: "api"},
	})

	require.Len(t, resources, 3)
	assert.Equal(t, []core.IntegrationResource{
		{Type: ResourceTypeService, ID: "api", Name: "api"},
		{Type: ResourceTypeService, ID: "billing", Name: "billing"},
		{Type: ResourceTypeService, ID: "checkout", Name: "checkout"},
	}, resources)
}

func Test__Datadog__ListResources(t *testing.T) {
	t.Run("lists distinct services from recent issues", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": [
							{"id": "issue-1", "type": "error_tracking_search_result", "relationships": {"issue": {"data": {"id": "issue-1", "type": "issue"}}}},
							{"id": "issue-2", "type": "error_tracking_search_result", "relationships": {"issue": {"data": {"id": "issue-2", "type": "issue"}}}},
							{"id": "issue-3", "type": "error_tracking_search_result", "relationships": {"issue": {"data": {"id": "issue-3", "type": "issue"}}}}
						],
						"included": [
							{"id": "issue-1", "type": "issue", "attributes": {"service": "checkout"}},
							{"id": "issue-2", "type": "issue", "attributes": {"service": "billing"}},
							{"id": "issue-3", "type": "issue", "attributes": {"service": "checkout"}}
						]
					}`)),
				},
			},
		}

		resources, err := (&Datadog{}).ListResources(ResourceTypeService, core.ListResourcesContext{
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiKey": "test-api-key",
					"appKey": "test-app-key",
					"site":   "datadoghq.com",
				},
			},
		})
		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "billing", resources[0].ID)
		assert.Equal(t, "checkout", resources[1].ID)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "issue", httpContext.Requests[0].URL.Query().Get("include"))
	})

	t.Run("lists environments from span aggregates", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": [
							{"type": "bucket", "attributes": {"by": {"env": "prod"}}},
							{"type": "bucket", "attributes": {"by": {"env": "staging"}}},
							{"type": "bucket", "attributes": {"by": {"env": "prod"}}}
						]
					}`)),
				},
			},
		}

		resources, err := (&Datadog{}).ListResources(ResourceTypeEnvironment, core.ListResourcesContext{
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiKey": "test-api-key",
					"appKey": "test-app-key",
					"site":   "datadoghq.com",
				},
			},
			Parameters: map[string]string{"service": "checkout"},
		})
		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "prod", resources[0].ID)
		assert.Equal(t, "staging", resources[1].ID)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, spansAggregatePath, httpContext.Requests[0].URL.Path)
	})

	t.Run("lists environments for the selected service", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"errors":["forbidden"]}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"buckets": [
								{"by": {"env": "staging"}},
								{"by": {"env": "prod"}},
								{"by": {"env": "prod"}}
							]
						}
					}`)),
				},
			},
		}

		resources, err := (&Datadog{}).ListResources(ResourceTypeEnvironment, core.ListResourcesContext{
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiKey": "test-api-key",
					"appKey": "test-app-key",
					"site":   "datadoghq.com",
				},
			},
			Parameters: map[string]string{"service": "checkout"},
		})
		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "prod", resources[0].ID)
		assert.Equal(t, "staging", resources[1].ID)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, spansAggregatePath, httpContext.Requests[0].URL.Path)
		assert.Equal(t, logsAggregatePath, httpContext.Requests[1].URL.Path)
		body, err := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), `service:\"checkout\"`)
	})

	t.Run("unknown resource types return an empty list", func(t *testing.T) {
		resources, err := (&Datadog{}).ListResources("project", core.ListResourcesContext{})
		require.NoError(t, err)
		assert.Empty(t, resources)
	})
}
