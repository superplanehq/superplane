package datadog

import (
	"encoding/json"
	"fmt"
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

func Test__Datadog__ListResources(t *testing.T) {
	t.Run("lists a service present only in spans", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				httpJSON(http.StatusOK, spanAggregateBody("service", "myt-home-api")),
				httpJSON(http.StatusOK, logAggregateBody("service")),
				httpJSON(http.StatusOK, issueSearchBody()),
			},
		}

		resources, err := (&Datadog{}).ListResources(ResourceTypeService, datadogListContext(httpContext))
		require.NoError(t, err)
		assert.Equal(t, []core.IntegrationResource{
			{Type: ResourceTypeService, ID: "myt-home-api", Name: "myt-home-api"},
		}, resources)
		require.Len(t, httpContext.Requests, 3)
		assert.Equal(t, spansAggregatePath, httpContext.Requests[0].URL.Path)
		assert.Equal(t, logsAggregatePath, httpContext.Requests[1].URL.Path)
		assert.Equal(t, "/api/v2/error-tracking/issues/search", httpContext.Requests[2].URL.Path)
		assertAggregateQuery(t, httpContext.Requests[0], serviceFacet, serviceAggregateMax, `"query":"*"`)
	})

	t.Run("lists a service present only in logs when spans return other names", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				httpJSON(http.StatusOK, spanAggregateBody("service", "checkout")),
				httpJSON(http.StatusOK, logAggregateBody("service", "myt-home-api")),
				httpJSON(http.StatusOK, issueSearchBody()),
			},
		}

		resources, err := (&Datadog{}).ListResources(ResourceTypeService, datadogListContext(httpContext))
		require.NoError(t, err)
		assert.Equal(t, []string{"checkout", "myt-home-api"}, resourceIDs(resources))
	})

	t.Run("lists a service present only on an open issue when telemetry is refused", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				httpJSON(http.StatusForbidden, `{"errors":["forbidden"]}`),
				httpJSON(http.StatusForbidden, `{"errors":["forbidden"]}`),
				httpJSON(http.StatusOK, issueSearchBody("billing")),
			},
		}

		resources, err := (&Datadog{}).ListResources(ResourceTypeService, datadogListContext(httpContext))
		require.NoError(t, err)
		assert.Equal(t, []string{"billing"}, resourceIDs(resources))
		require.Len(t, httpContext.Requests, 3)
		assert.Equal(t, "issue", httpContext.Requests[2].URL.Query().Get("include"))
	})

	t.Run("lists each service once and sorts the names", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				httpJSON(http.StatusOK, spanAggregateBody("service", "zeta", "alpha", "  ", "alpha")),
				httpJSON(http.StatusOK, logAggregateBody("service", "alpha", "middle")),
				httpJSON(http.StatusOK, issueSearchBody("middle", "  ", "beta")),
			},
		}

		resources, err := (&Datadog{}).ListResources(ResourceTypeService, datadogListContext(httpContext))
		require.NoError(t, err)
		assert.Equal(t, []string{"alpha", "beta", "middle", "zeta"}, resourceIDs(resources))
	})

	t.Run("keeps the list when a telemetry query fails or is refused", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				httpJSON(http.StatusInternalServerError, `{"errors":["unavailable"]}`),
				httpJSON(http.StatusForbidden, `{"errors":["forbidden"]}`),
				httpJSON(http.StatusOK, issueSearchBody("checkout")),
			},
		}

		resources, err := (&Datadog{}).ListResources(ResourceTypeService, datadogListContext(httpContext))
		require.NoError(t, err)
		assert.Equal(t, []string{"checkout"}, resourceIDs(resources))
		require.Len(t, httpContext.Requests, 3)
	})

	t.Run("returns the Error Tracking permission error when every source is refused", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				httpJSON(http.StatusForbidden, `{"errors":["forbidden"]}`),
				httpJSON(http.StatusForbidden, `{"errors":["forbidden"]}`),
				httpJSON(http.StatusForbidden, `{"errors":["forbidden"]}`),
			},
		}

		_, err := (&Datadog{}).ListResources(ResourceTypeService, datadogListContext(httpContext))
		require.ErrorIs(t, err, ErrErrorTrackingForbidden)
		require.Len(t, httpContext.Requests, 3)
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
		assertAggregateQuery(t, httpContext.Requests[0], environmentFacet, environmentAggregateMax, `service:\"checkout\"`)
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

func datadogListContext(httpContext *contexts.HTTPContext) core.ListResourcesContext {
	return core.ListResourcesContext{
		HTTP: httpContext,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
				"appKey": "test-app-key",
				"site":   "datadoghq.com",
			},
		},
	}
}

func httpJSON(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func spanAggregateBody(facet string, names ...string) string {
	buckets := make([]string, 0, len(names))
	for _, name := range names {
		buckets = append(buckets, fmt.Sprintf(`{"type":"bucket","attributes":{"by":{"%s":"%s"}}}`, facet, name))
	}
	return `{"data":[` + strings.Join(buckets, ",") + `]}`
}

func logAggregateBody(facet string, names ...string) string {
	buckets := make([]string, 0, len(names))
	for _, name := range names {
		buckets = append(buckets, fmt.Sprintf(`{"by":{"%s":"%s"}}`, facet, name))
	}
	return `{"data":{"buckets":[` + strings.Join(buckets, ",") + `]}}`
}

func issueSearchBody(services ...string) string {
	data := make([]string, 0, len(services))
	included := make([]string, 0, len(services))
	for i, service := range services {
		id := fmt.Sprintf("issue-%d", i+1)
		data = append(data, fmt.Sprintf(
			`{"id":"%s","type":"error_tracking_search_result","relationships":{"issue":{"data":{"id":"%s","type":"issue"}}}}`,
			id,
			id,
		))
		included = append(included, fmt.Sprintf(
			`{"id":"%s","type":"issue","attributes":{"service":"%s"}}`,
			id,
			service,
		))
	}
	return `{"data":[` + strings.Join(data, ",") + `],"included":[` + strings.Join(included, ",") + `]}`
}

func resourceIDs(resources []core.IntegrationResource) []string {
	ids := make([]string, 0, len(resources))
	for _, resource := range resources {
		ids = append(ids, resource.ID)
	}
	return ids
}

func assertAggregateQuery(t *testing.T, request *http.Request, facet string, limit int, queryPart string) {
	t.Helper()
	body, err := io.ReadAll(request.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), queryPart)

	var payload struct {
		Data struct {
			Attributes struct {
				Filter struct {
					From string `json:"from"`
				} `json:"filter"`
				GroupBy []struct {
					Facet string `json:"facet"`
					Limit int    `json:"limit"`
				} `json:"group_by"`
			} `json:"attributes"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Len(t, payload.Data.Attributes.GroupBy, 1)
	assert.Equal(t, facet, payload.Data.Attributes.GroupBy[0].Facet)
	assert.Equal(t, limit, payload.Data.Attributes.GroupBy[0].Limit)
	assert.Equal(t, "now-30d", payload.Data.Attributes.Filter.From)
}
