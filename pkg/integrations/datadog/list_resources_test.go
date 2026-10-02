package datadog

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

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
		httpContext := serviceListHTTP(
			[]*http.Response{httpJSON(http.StatusOK, spanAggregateBody("service", "myt-home-api"))},
			[]*http.Response{httpJSON(http.StatusOK, logAggregateBody("service"))},
			[]*http.Response{httpJSON(http.StatusOK, issueSearchBody())},
		)

		resources, err := (&Datadog{}).ListResources(ResourceTypeService, datadogListContext(httpContext))
		require.NoError(t, err)
		assert.Equal(t, []core.IntegrationResource{
			{Type: ResourceTypeService, ID: "myt-home-api", Name: "myt-home-api"},
		}, resources)
		assertServiceListPaths(t, httpContext.Requests)
		assertAggregateQuery(t, requestByPath(t, httpContext.Requests, spansAggregatePath), serviceFacet, serviceAggregatePageSize, `"query":"*"`)
	})

	t.Run("lists a service present only in logs when spans return other names", func(t *testing.T) {
		httpContext := serviceListHTTP(
			[]*http.Response{httpJSON(http.StatusOK, spanAggregateBody("service", "checkout"))},
			[]*http.Response{httpJSON(http.StatusOK, logAggregateBody("service", "myt-home-api"))},
			[]*http.Response{httpJSON(http.StatusOK, issueSearchBody())},
		)

		resources, err := (&Datadog{}).ListResources(ResourceTypeService, datadogListContext(httpContext))
		require.NoError(t, err)
		assert.Equal(t, []string{"checkout", "myt-home-api"}, resourceIDs(resources))
	})

	t.Run("lists a service present only on an open issue when telemetry is refused", func(t *testing.T) {
		httpContext := serviceListHTTP(
			[]*http.Response{httpJSON(http.StatusForbidden, `{"errors":["forbidden"]}`)},
			[]*http.Response{httpJSON(http.StatusForbidden, `{"errors":["forbidden"]}`)},
			[]*http.Response{httpJSON(http.StatusOK, issueSearchBody("billing"))},
		)

		resources, err := (&Datadog{}).ListResources(ResourceTypeService, datadogListContext(httpContext))
		require.NoError(t, err)
		assert.Equal(t, []string{"billing"}, resourceIDs(resources))
		assertServiceListPaths(t, httpContext.Requests)
		assert.Equal(t, "issue", requestByPath(t, httpContext.Requests, errorTrackingSearchPath).URL.Query().Get("include"))
	})

	t.Run("lists each service once and sorts the names", func(t *testing.T) {
		httpContext := serviceListHTTP(
			[]*http.Response{httpJSON(http.StatusOK, spanAggregateBody("service", "zeta", "alpha", "  ", "alpha"))},
			[]*http.Response{httpJSON(http.StatusOK, logAggregateBody("service", "alpha", "middle"))},
			[]*http.Response{httpJSON(http.StatusOK, issueSearchBody("middle", "  ", "beta"))},
		)

		resources, err := (&Datadog{}).ListResources(ResourceTypeService, datadogListContext(httpContext))
		require.NoError(t, err)
		assert.Equal(t, []string{"alpha", "beta", "middle", "zeta"}, resourceIDs(resources))
	})

	t.Run("keeps telemetry services when the issue search is refused", func(t *testing.T) {
		httpContext := serviceListHTTP(
			[]*http.Response{httpJSON(http.StatusOK, spanAggregateBody("service", "checkout"))},
			[]*http.Response{httpJSON(http.StatusOK, logAggregateBody("service"))},
			[]*http.Response{httpJSON(http.StatusForbidden, `{"errors":["forbidden"]}`)},
		)

		resources, err := (&Datadog{}).ListResources(ResourceTypeService, datadogListContext(httpContext))
		require.NoError(t, err)
		assert.Equal(t, []string{"checkout"}, resourceIDs(resources))
		assert.Empty(t, listNoticeID(resources))
	})

	t.Run("keeps telemetry services when the issue search fails", func(t *testing.T) {
		httpContext := serviceListHTTP(
			[]*http.Response{httpJSON(http.StatusOK, spanAggregateBody("service", "checkout"))},
			[]*http.Response{httpJSON(http.StatusOK, logAggregateBody("service", "billing"))},
			[]*http.Response{httpJSON(http.StatusInternalServerError, `{"errors":["unavailable"]}`)},
		)

		resources, err := (&Datadog{}).ListResources(ResourceTypeService, datadogListContext(httpContext))
		require.NoError(t, err)
		assert.Equal(t, []string{"billing", "checkout"}, serviceResourceIDs(resources))
		assert.Equal(t, []core.IntegrationResource{
			{Type: ResourceTypeService, ID: "billing", Name: "billing"},
			{Type: ResourceTypeService, ID: "checkout", Name: "checkout"},
			{
				Type: ResourceTypeListNotice,
				ID:   ListNoticeIssueSearchFailed,
				Name: listNoticeIssueSearchFailedName,
			},
		}, resources)
	})

	t.Run("fails the list when the issue search fails and telemetry has no names", func(t *testing.T) {
		httpContext := serviceListHTTP(
			[]*http.Response{httpJSON(http.StatusOK, spanAggregateBody("service"))},
			[]*http.Response{httpJSON(http.StatusForbidden, `{"errors":["forbidden"]}`)},
			[]*http.Response{httpJSON(http.StatusInternalServerError, `{"errors":["unavailable"]}`)},
		)

		_, err := (&Datadog{}).ListResources(ResourceTypeService, datadogListContext(httpContext))
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrErrorTrackingForbidden)
	})

	t.Run("keeps the list when a telemetry query fails or is refused", func(t *testing.T) {
		httpContext := serviceListHTTP(
			[]*http.Response{httpJSON(http.StatusInternalServerError, `{"errors":["unavailable"]}`)},
			[]*http.Response{httpJSON(http.StatusForbidden, `{"errors":["forbidden"]}`)},
			[]*http.Response{httpJSON(http.StatusOK, issueSearchBody("checkout"))},
		)

		resources, err := (&Datadog{}).ListResources(ResourceTypeService, datadogListContext(httpContext))
		require.NoError(t, err)
		assert.Equal(t, []string{"checkout"}, resourceIDs(resources))
		assertServiceListPaths(t, httpContext.Requests)
	})

	t.Run("returns the Error Tracking permission error when every source is refused", func(t *testing.T) {
		httpContext := serviceListHTTP(
			[]*http.Response{httpJSON(http.StatusForbidden, `{"errors":["forbidden"]}`)},
			[]*http.Response{httpJSON(http.StatusForbidden, `{"errors":["forbidden"]}`)},
			[]*http.Response{httpJSON(http.StatusForbidden, `{"errors":["forbidden"]}`)},
		)

		_, err := (&Datadog{}).ListResources(ResourceTypeService, datadogListContext(httpContext))
		require.ErrorIs(t, err, ErrErrorTrackingForbidden)
		assertServiceListPaths(t, httpContext.Requests)
	})

	t.Run("asks for more names when one telemetry page is full", func(t *testing.T) {
		firstPage := serviceNames(serviceAggregatePageSize)
		httpContext := serviceListHTTP(
			[]*http.Response{
				httpJSON(http.StatusOK, spanAggregateBody("service", firstPage...)),
				httpJSON(http.StatusOK, spanAggregateBody("service", "svc-extra")),
			},
			[]*http.Response{httpJSON(http.StatusOK, logAggregateBody("service"))},
			[]*http.Response{httpJSON(http.StatusOK, issueSearchBody())},
		)

		resources, err := (&Datadog{}).ListResources(ResourceTypeService, datadogListContext(httpContext))
		require.NoError(t, err)
		assert.Contains(t, resourceIDs(resources), "svc-extra")
		assert.Contains(t, resourceIDs(resources), "svc-0000")
		spanRequests := requestsByPath(httpContext.Requests, spansAggregatePath)
		require.Len(t, spanRequests, 2)
		assertAggregateQuery(t, spanRequests[0], serviceFacet, serviceAggregatePageSize, `"query":"*"`)
		assertAggregateQuery(t, spanRequests[1], serviceFacet, serviceAggregateMax, `"query":"*"`)
	})

	t.Run("reads the next names when a larger page is refused", func(t *testing.T) {
		firstPage := serviceNames(serviceAggregatePageSize)
		httpContext := serviceListHTTP(
			[]*http.Response{
				httpJSON(http.StatusOK, spanAggregateBody("service", firstPage...)),
				httpJSON(http.StatusBadRequest, `{"errors":["limit"]}`),
				httpJSON(http.StatusOK, spanAggregateBody("service", "svc-extra")),
			},
			[]*http.Response{httpJSON(http.StatusOK, logAggregateBody("service"))},
			[]*http.Response{httpJSON(http.StatusOK, issueSearchBody())},
		)

		resources, err := (&Datadog{}).ListResources(ResourceTypeService, datadogListContext(httpContext))
		require.NoError(t, err)
		assert.Contains(t, resourceIDs(resources), "svc-0999")
		assert.Contains(t, resourceIDs(resources), "svc-extra")
		spanRequests := requestsByPath(httpContext.Requests, spansAggregatePath)
		require.Len(t, spanRequests, 3)
		assert.Equal(t, `service:>"svc-0999"`, aggregateQuery(t, spanRequests[2]))
	})

	t.Run("stops when the next page does not add names", func(t *testing.T) {
		firstPage := serviceNames(serviceAggregatePageSize)
		httpContext := serviceListHTTP(
			[]*http.Response{
				httpJSON(http.StatusOK, spanAggregateBody("service", firstPage...)),
				httpJSON(http.StatusBadRequest, `{"errors":["limit"]}`),
				httpJSON(http.StatusOK, spanAggregateBody("service", firstPage...)),
			},
			[]*http.Response{httpJSON(http.StatusOK, logAggregateBody("service"))},
			[]*http.Response{httpJSON(http.StatusOK, issueSearchBody())},
		)

		resources, err := (&Datadog{}).ListResources(ResourceTypeService, datadogListContext(httpContext))
		require.NoError(t, err)
		assert.Len(t, resources, serviceAggregatePageSize)
		assert.Len(t, requestsByPath(httpContext.Requests, spansAggregatePath), 3)
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

func TestListServicesStartsIssueSearchWhileTelemetryIsOpen(t *testing.T) {
	gate := &gatedHTTP{
		allEntered: make(chan struct{}),
		release:    make(chan struct{}),
		byPath: map[string]*http.Response{
			logsAggregatePath:       httpJSON(http.StatusOK, logAggregateBody("service", "from-logs")),
			errorTrackingSearchPath: httpJSON(http.StatusOK, issueSearchBody("from-issues")),
		},
	}
	client := &Client{
		APIKey:  "test-api-key",
		AppKey:  "test-app-key",
		Site:    "datadoghq.com",
		BaseURL: "https://api.datadoghq.com",
		http:    gate,
	}
	client.SetRequestDeadline(time.Now().Add(time.Second))

	done := make(chan struct{})
	var names []string
	var err error
	started := time.Now()
	go func() {
		defer close(done)
		names, err = client.ListServices()
	}()

	select {
	case <-gate.allEntered:
	case <-time.After(time.Second):
		t.Fatal("issue search did not start while the span read was still open")
	}
	close(gate.release)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("stalled span read held the service list")
	}
	require.NoError(t, err)
	assert.Less(t, time.Since(started), 2*time.Second)
	assert.Equal(t, []string{"from-issues", "from-logs"}, names)
}

func TestListServicesKeepsTelemetryNamesWhenTheIssueSearchMissesTheDeadline(t *testing.T) {
	gate := &deadlineHTTP{
		started: make(chan struct{}),
	}
	client := &Client{
		APIKey:  "test-api-key",
		AppKey:  "test-app-key",
		Site:    "datadoghq.com",
		BaseURL: "https://api.datadoghq.com",
		http:    gate,
	}
	client.SetRequestDeadline(time.Now().Add(50 * time.Millisecond))

	resources, err := client.listServiceResources()
	require.NoError(t, err)
	assert.Equal(t, []string{"from-logs", "from-spans"}, serviceResourceIDs(resources))
	assert.Equal(t, ListNoticeIssueSearchFailed, listNoticeID(resources))
	select {
	case <-gate.started:
	default:
		t.Fatal("issue search did not start")
	}
}

type deadlineHTTP struct {
	started chan struct{}
	once    sync.Once
}

func (h *deadlineHTTP) Do(request *http.Request) (*http.Response, error) {
	switch request.URL.Path {
	case spansAggregatePath:
		return httpJSON(http.StatusOK, spanAggregateBody("service", "from-spans")), nil
	case logsAggregatePath:
		return httpJSON(http.StatusOK, logAggregateBody("service", "from-logs")), nil
	case errorTrackingSearchPath:
		h.once.Do(func() { close(h.started) })
		<-request.Context().Done()
		return nil, request.Context().Err()
	default:
		return nil, fmt.Errorf("unexpected path %s", request.URL.Path)
	}
}

const errorTrackingSearchPath = "/api/v2/error-tracking/issues/search"

func serviceListHTTP(spans, logs, issues []*http.Response) *contexts.HTTPContext {
	return &contexts.HTTPContext{
		ResponsesByPath: map[string][]*http.Response{
			spansAggregatePath:      spans,
			logsAggregatePath:       logs,
			errorTrackingSearchPath: issues,
		},
	}
}

func serviceNames(count int) []string {
	names := make([]string, count)
	for i := range names {
		names[i] = fmt.Sprintf("svc-%04d", i)
	}
	return names
}

func assertServiceListPaths(t *testing.T, requests []*http.Request) {
	t.Helper()
	require.Len(t, requests, 3)
	assert.NotNil(t, requestByPath(t, requests, spansAggregatePath))
	assert.NotNil(t, requestByPath(t, requests, logsAggregatePath))
	assert.NotNil(t, requestByPath(t, requests, errorTrackingSearchPath))
}

func requestByPath(t *testing.T, requests []*http.Request, path string) *http.Request {
	t.Helper()
	matches := requestsByPath(requests, path)
	require.NotEmpty(t, matches)
	return matches[0]
}

func requestsByPath(requests []*http.Request, path string) []*http.Request {
	matches := make([]*http.Request, 0)
	for _, request := range requests {
		if request.URL.Path == path {
			matches = append(matches, request)
		}
	}
	return matches
}

type gatedHTTP struct {
	mu         sync.Mutex
	entered    int
	allEntered chan struct{}
	release    chan struct{}
	byPath     map[string]*http.Response
}

func (h *gatedHTTP) Do(request *http.Request) (*http.Response, error) {
	h.mu.Lock()
	h.entered++
	if h.entered == 3 {
		close(h.allEntered)
	}
	h.mu.Unlock()

	if request.URL.Path == spansAggregatePath {
		<-request.Context().Done()
		return nil, request.Context().Err()
	}
	<-h.release
	response := h.byPath[request.URL.Path]
	if response == nil {
		return nil, fmt.Errorf("no response for %s", request.URL.Path)
	}
	return response, nil
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
	return serviceResourceIDs(resources)
}

func serviceResourceIDs(resources []core.IntegrationResource) []string {
	ids := make([]string, 0, len(resources))
	for _, resource := range resources {
		if resource.Type != ResourceTypeService {
			continue
		}
		ids = append(ids, resource.ID)
	}
	return ids
}

func listNoticeID(resources []core.IntegrationResource) string {
	for _, resource := range resources {
		if resource.Type == ResourceTypeListNotice {
			return resource.ID
		}
	}
	return ""
}

func aggregateQuery(t *testing.T, request *http.Request) string {
	t.Helper()
	body, err := io.ReadAll(request.Body)
	require.NoError(t, err)
	var payload struct {
		Data struct {
			Attributes struct {
				Filter struct {
					Query string `json:"query"`
				} `json:"filter"`
			} `json:"attributes"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &payload))
	return payload.Data.Attributes.Filter.Query
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
