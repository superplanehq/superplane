package grafana

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__readBaseURL__RejectsRelativeURL(t *testing.T) {
	_, err := readBaseURL(&contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseURL": "grafana.local",
		},
	})
	require.ErrorContains(t, err, "must include scheme and host")
}

func Test__readBaseURL__AcceptsAbsoluteHTTPURL(t *testing.T) {
	baseURL, err := readBaseURL(&contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseURL": "https://grafana.example.com/",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "https://grafana.example.com", baseURL)
}

func Test__Client__ExecRequest__AllowsExactMaxSize(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(bytes.Repeat([]byte("a"), maxResponseSize))),
			},
		},
	}

	client := &Client{
		BaseURL: "https://grafana.example.com",
		http:    httpContext,
	}

	body, status, err := client.execRequest(http.MethodGet, "/api/health", nil, "")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, body, maxResponseSize)
}

func Test__Client__ExecRequest__RejectsOverMaxSize(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(bytes.Repeat([]byte("a"), maxResponseSize+1))),
			},
		},
	}

	client := &Client{
		BaseURL: "https://grafana.example.com",
		http:    httpContext,
	}

	_, status, err := client.execRequest(http.MethodGet, "/api/health", nil, "")
	require.ErrorContains(t, err, "response too large")
	require.Equal(t, http.StatusOK, status)
}

func Test__Grafana__Sync__RejectsRelativeBaseURL(t *testing.T) {
	err := (&Grafana{}).Sync(core.SyncContext{
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL": "grafana.local",
			},
			Metadata: map[string]any{},
		},
	})
	require.ErrorContains(t, err, "must include scheme and host")
}

func Test__Client__UpsertWebhookContactPoint__UpdatesExisting(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[{"uid":"cp_1","name":"superplane-123"}]`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"uid":"cp_1"}`)),
			},
		},
	}

	client := &Client{
		BaseURL:  "https://grafana.example.com",
		APIToken: "token",
		http:     httpContext,
	}

	uid, err := client.UpsertWebhookContactPoint("superplane-123", "https://example.com/webhook", "secret")
	require.NoError(t, err)
	require.Equal(t, "cp_1", uid)
	require.Len(t, httpContext.Requests, 2)
	require.Equal(t, http.MethodPut, httpContext.Requests[1].Method)
	require.Equal(t, "true", httpContext.Requests[1].Header.Get("X-Disable-Provenance"))
}

func Test__Client__UpsertWebhookContactPoint__CreatesAndFindsUID(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[]`)),
			},
			{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[{"uid":"cp_2","name":"superplane-abc"}]`)),
			},
		},
	}

	client := &Client{
		BaseURL:  "https://grafana.example.com",
		APIToken: "token",
		http:     httpContext,
	}

	uid, err := client.UpsertWebhookContactPoint("superplane-abc", "https://example.com/webhook", "")
	require.NoError(t, err)
	require.Equal(t, "cp_2", uid)
	require.Len(t, httpContext.Requests, 3)
	require.Equal(t, "true", httpContext.Requests[1].Header.Get("X-Disable-Provenance"))

	body, err := io.ReadAll(httpContext.Requests[1].Body)
	require.NoError(t, err)
	payload := map[string]any{}
	require.NoError(t, json.Unmarshal(body, &payload))
	settings := payload["settings"].(map[string]any)
	_, hasAuthScheme := settings["authorization_scheme"]
	require.False(t, hasAuthScheme)
}

func Test__Client__DeleteContactPoint__NotFoundIsIgnored(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader(`not found`)),
			},
		},
	}

	client := &Client{
		BaseURL:  "https://grafana.example.com",
		APIToken: "token",
		http:     httpContext,
	}

	err := client.DeleteContactPoint("cp_missing")
	require.NoError(t, err)
}

func Test__Client__listContactPoints__AcceptsWrappedItemsFormat(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"items":[{"uid":"cp_1","name":"superplane-1"}]}`)),
			},
		},
	}

	client := &Client{
		BaseURL:  "https://grafana.example.com",
		APIToken: "token",
		http:     httpContext,
	}

	points, err := client.listContactPoints()
	require.NoError(t, err)
	require.Len(t, points, 1)
	require.Equal(t, "cp_1", points[0].UID)
}

func Test__Client__listContactPoints__ErrorsWhenWrappedItemsFieldMissing(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"contactPoints":[{"uid":"cp_1","name":"superplane-1"}]}`)),
			},
		},
	}

	client := &Client{
		BaseURL:  "https://grafana.example.com",
		APIToken: "token",
		http:     httpContext,
	}

	_, err := client.listContactPoints()
	require.ErrorContains(t, err, "error parsing contact points response")
}

func Test__Client__ListDataSources(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[{"uid":"prom","name":"Prometheus"},{"uid":"loki","name":"Loki"}]`)),
			},
		},
	}

	client := &Client{
		BaseURL:  "https://grafana.example.com",
		APIToken: "token",
		http:     httpContext,
	}

	dataSources, err := client.ListDataSources()
	require.NoError(t, err)
	require.Len(t, dataSources, 2)
	require.Equal(t, "prom", dataSources[0].UID)
	require.Equal(t, "Prometheus", dataSources[0].Name)
}

func Test__Grafana__ListResources(t *testing.T) {
	g := &Grafana{}

	t.Run("unknown resource type returns empty", func(t *testing.T) {
		resources, err := g.ListResources("unknown", core.ListResourcesContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"baseURL":  "https://grafana.example.com",
					"apiToken": "token",
				},
			},
		})
		require.NoError(t, err)
		require.Empty(t, resources)
	})

	t.Run("data-source returns grafana datasources", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{"uid":"prom","name":"Prometheus"},
						{"uid":"loki","name":"Loki"},
						{"uid":"","name":"Missing UID"}
					]`)),
				},
			},
		}

		resources, err := g.ListResources(resourceTypeDataSource, core.ListResourcesContext{
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"baseURL":  "https://grafana.example.com",
					"apiToken": "token",
				},
			},
		})
		require.NoError(t, err)
		require.Len(t, resources, 2)
		require.Equal(t, "Prometheus", resources[0].Name)
		require.Equal(t, "prom", resources[0].ID)
		require.Equal(t, resourceTypeDataSource, resources[0].Type)
	})

	t.Run("dashboard returns search hits", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{"uid":"dash1","title":"Overview","type":"dash-db"},
						{"uid":"","title":"No UID","type":"dash-db"}
					]`)),
				},
			},
		}

		resources, err := g.ListResources(resourceTypeDashboard, core.ListResourcesContext{
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"baseURL":  "https://grafana.example.com",
					"apiToken": "token",
				},
			},
		})
		require.NoError(t, err)
		require.Len(t, resources, 1)
		require.Equal(t, "Overview", resources[0].Name)
		require.Equal(t, "dash1", resources[0].ID)
		require.Equal(t, resourceTypeDashboard, resources[0].Type)
	})

	t.Run("panel returns dashboard panels scoped by dashboard uid", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"dashboard": {
							"panels": [
								{"id": 3, "title": "Latency"},
								{"type": "row", "panels": [
									{"id": 7, "title": "Errors"},
									{"id": 9, "title": ""}
								]}
							]
						}
					}`)),
				},
			},
		}

		resources, err := g.ListResources(resourceTypePanel, core.ListResourcesContext{
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"baseURL":  "https://grafana.example.com",
					"apiToken": "token",
				},
			},
			Parameters: map[string]string{
				"dashboardUID": "dash-1",
			},
		})
		require.NoError(t, err)
		require.Len(t, resources, 3)
		require.Equal(t, "Latency", resources[0].Name)
		require.Equal(t, "3", resources[0].ID)
		require.Equal(t, "Errors", resources[1].Name)
		require.Equal(t, "7", resources[1].ID)
		require.Equal(t, "Panel 9", resources[2].Name)
		require.Equal(t, resourceTypePanel, resources[2].Type)
		require.Contains(t, httpContext.Requests[0].URL.Path, "/api/dashboards/uid/dash-1")
	})
}

func Test__Client__SearchDashboards__ParsesResponse(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[{"uid":"a","title":"A"},{"uid":"b","title":""}]`)),
			},
		},
	}

	client := &Client{
		BaseURL:  "https://grafana.example.com",
		APIToken: "token",
		http:     httpContext,
	}

	hits, err := client.SearchDashboards()
	require.NoError(t, err)
	require.Len(t, hits, 2)
	require.Equal(t, "a", hits[0].UID)
	require.Equal(t, "A", hits[0].Title)
	require.Equal(t, "b", hits[1].UID)
	require.Equal(t, "", hits[1].Title)
	require.Contains(t, httpContext.Requests[0].URL.Path, "/api/search")
}

func Test__Client__ListDashboardPanels(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"dashboard": {
						"panels": [
							{"id": 2, "title": "CPU"},
							{"id": 3, "type": "row", "title": "Row", "collapsed": false, "panels": [
								{"id": 4, "title": "Memory"},
								{"id": 6, "title": ""}
							]}
						]
					}
				}`)),
			},
		},
	}

	client := &Client{
		BaseURL:  "https://grafana.example.com",
		APIToken: "token",
		http:     httpContext,
	}

	panels, err := client.ListDashboardPanels("dash-1")
	require.NoError(t, err)
	require.Len(t, panels, 3)
	require.Equal(t, DashboardPanel{ID: 2, Title: "CPU"}, panels[0])
	require.Equal(t, DashboardPanel{ID: 4, Title: "Memory"}, panels[1])
	require.Equal(t, DashboardPanel{ID: 6, Title: ""}, panels[2])
	require.Contains(t, httpContext.Requests[0].URL.Path, "/api/dashboards/uid/dash-1")
}

func Test__Client__GetDashboardTitle(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"dashboard": {"title": "Production Overview", "uid": "abc"}
				}`)),
			},
		},
	}

	client := &Client{
		BaseURL:  "https://grafana.example.com",
		APIToken: "token",
		http:     httpContext,
	}

	title, err := client.GetDashboardTitle("abc")
	require.NoError(t, err)
	require.Equal(t, "Production Overview", title)
	require.Contains(t, httpContext.Requests[0].URL.Path, "/api/dashboards/uid/abc")
}

func Test__Client__GetAnnotation(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"id":42,"text":"deploy","tags":["prod"],"time":1,"timeEnd":2,"dashboardUID":"d1","panelId":3}`,
				)),
			},
		},
	}

	client := &Client{
		BaseURL:  "https://grafana.example.com",
		APIToken: "token",
		http:     httpContext,
	}

	annotation, err := client.GetAnnotation(42)
	require.NoError(t, err)
	require.Equal(t, int64(42), annotation.ID)
	require.Equal(t, "deploy", annotation.Text)
	require.Contains(t, httpContext.Requests[0].URL.Path, "/api/annotations/42")
}

func Test__Client__CreateAnnotation__RetriesRateLimit(t *testing.T) {
	originalRetryDelays := createAnnotationRetryDelays
	createAnnotationRetryDelays = []time.Duration{0}
	defer func() {
		createAnnotationRetryDelays = originalRetryDelays
	}()

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(strings.NewReader(`Too Many Requests`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":42,"message":"Annotation added"}`)),
			},
		},
	}

	client := &Client{
		BaseURL:  "https://grafana.example.com",
		APIToken: "token",
		http:     httpContext,
	}

	panelID := int64(7)
	id, err := client.CreateAnnotation("deploy", []string{"prod"}, "dashboard-1", &panelID, 1, 2)
	require.NoError(t, err)
	require.Equal(t, int64(42), id)
	require.Len(t, httpContext.Requests, 2)
}

func Test__Client__CreateAnnotation__ReturnsRateLimitAfterRetries(t *testing.T) {
	originalRetryDelays := createAnnotationRetryDelays
	createAnnotationRetryDelays = []time.Duration{0, 0}
	defer func() {
		createAnnotationRetryDelays = originalRetryDelays
	}()

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(strings.NewReader(`Too Many Requests`)),
			},
			{
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(strings.NewReader(`Too Many Requests`)),
			},
			{
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(strings.NewReader(`Too Many Requests`)),
			},
		},
	}

	client := &Client{
		BaseURL:  "https://grafana.example.com",
		APIToken: "token",
		http:     httpContext,
	}

	panelID := int64(7)
	_, err := client.CreateAnnotation("deploy", nil, "dashboard-1", &panelID, 1, 2)
	require.ErrorContains(t, err, "status 429")
	require.Len(t, httpContext.Requests, 3)
}

func Test__Grafana__ListResources__Annotations(t *testing.T) {
	g := &Grafana{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`[
					{"id":1,"text":"a","time":0},
					{"id":2,"text":"","time":0}
				]`)),
			},
		},
	}

	resources, err := g.ListResources(resourceTypeAnnotation, core.ListResourcesContext{
		HTTP: httpContext,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://grafana.example.com",
				"apiToken": "token",
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, resources, 2)
	require.Equal(t, "1", resources[0].ID)
	require.Equal(t, resourceTypeAnnotation, resources[0].Type)
	require.Contains(t, resources[0].Name, "#1")
	require.Equal(t, "2", resources[1].ID)
	require.Equal(t, "#2", resources[1].Name)
}

func Test__notificationPolicyRoot__PreservesUnknownRootAndRouteFields(t *testing.T) {
	raw := []byte(`{"receiver":"default","mute_time_intervals":["offhours"],"routes":[{"receiver":"keep-me","matchers":[{"type":"a"}]}]}`)
	root, err := parseNotificationPolicyRoot(raw)
	require.NoError(t, err)
	require.Contains(t, root, "mute_time_intervals")

	routes, err := getChildRoutes(root)
	require.NoError(t, err)
	require.Len(t, routes, 1)

	filtered, err := removeRoutesForReceiverRaw(routes, "other")
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	require.NoError(t, setChildRoutes(root, filtered))

	out, err := marshalNotificationPolicyRoot(root)
	require.NoError(t, err)

	var check map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(out, &check))
	require.Contains(t, check, "mute_time_intervals")

	var rt []json.RawMessage
	require.NoError(t, json.Unmarshal(check["routes"], &rt))
	require.Len(t, rt, 1)
	require.Contains(t, string(rt[0]), "matchers")
}
