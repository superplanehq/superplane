package grafana

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

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

func Test__dashboardURLPathSlug(t *testing.T) {
	require.Equal(t, "my-slug", dashboardURLPathSlug(&DashboardDetails{Slug: "my-slug", UID: "abc"}))
	require.Equal(t, "abc", dashboardURLPathSlug(&DashboardDetails{Slug: "", UID: "abc"}))
	require.Equal(t, "abc", dashboardURLPathSlug(&DashboardDetails{Slug: "   ", UID: "abc"}))
	require.Equal(t, "dashboard", dashboardURLPathSlug(&DashboardDetails{Slug: "", UID: ""}))
	require.Equal(t, "dashboard", dashboardURLPathSlug(nil))
}

func Test__Client__RenderPanelURL(t *testing.T) {
	client := &Client{BaseURL: "https://grafana.example.com/"}

	got := client.RenderPanelURL("cIBgcSjkk", "production-overview", 2, 1000, 500, "now-1h", "now")

	require.Equal(
		t,
		"https://grafana.example.com/render/d-solo/cIBgcSjkk/production-overview?from=now-1h&height=500&panelId=2&to=now&tz=UTC&width=1000",
		got,
	)
}

func Test__collectDashboardPanelSummaries__nestedUnderRows(t *testing.T) {
	raw := []json.RawMessage{
		json.RawMessage(`{"id":10,"title":"Resources","type":"row","panels":[{"id":1,"title":"CPU","type":"timeseries"},{"id":2,"title":"Memory","type":"timeseries"}]}`),
		json.RawMessage(`{"id":3,"title":"Standalone","type":"gauge"}`),
	}
	got := collectDashboardPanelSummaries(raw)
	require.Len(t, got, 3)
	require.Equal(t, 1, got[0].ID)
	require.Equal(t, "CPU", got[0].Title)
	require.Equal(t, "timeseries", got[0].Type)
	require.Equal(t, 2, got[1].ID)
	require.Equal(t, 3, got[2].ID)
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
