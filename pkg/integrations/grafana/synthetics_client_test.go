package grafana

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__readSyntheticsBaseURL__AcceptsAbsoluteURL(t *testing.T) {
	baseURL, err := readSyntheticsBaseURL(&contexts.IntegrationContext{
		Configuration: map[string]any{
			"syntheticsBaseURL": "https://synthetic-monitoring-api.grafana.net/",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "https://synthetic-monitoring-api.grafana.net", baseURL)
}

func Test__SyntheticsClient__ListChecks__UsesBearerToken(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[]`)),
			},
		},
	}

	client := &SyntheticsClient{
		BaseURL:     "https://synthetic-monitoring-api.grafana.net",
		AccessToken: "sm-token",
		http:        httpContext,
	}

	checks, err := client.ListChecks()
	require.NoError(t, err)
	assert.Empty(t, checks)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, "Bearer sm-token", httpContext.Requests[0].Header.Get("Authorization"))
	assert.Equal(t, "/api/v1/check", httpContext.Requests[0].URL.Path)
}

func Test__NewSyntheticsClient__FallsBackToGrafanaDatasourceProxy(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[{"uid":"sm-ds","name":"Synthetic Monitoring","type":"synthetic-monitoring-datasource","jsonData":{"metrics":{"uid":"prom-ds"}}}]`)),
			},
		},
	}

	client, err := NewSyntheticsClient(httpContext, &contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseURL":  "https://grafana.example.com",
			"apiToken": "grafana-token",
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "sm-ds", client.DataSourceUID)
	require.NotNil(t, client.GrafanaClient)
}

func Test__SyntheticsClient__ListProbes__UsesGrafanaDatasourceProxy(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[]`)),
			},
		},
	}

	client := &SyntheticsClient{
		DataSourceUID: "sm-ds",
		GrafanaClient: &Client{
			BaseURL:  "https://grafana.example.com",
			APIToken: "grafana-token",
			http:     httpContext,
		},
		http: httpContext,
	}

	probes, err := client.ListProbes()
	require.NoError(t, err)
	assert.Empty(t, probes)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, "/api/datasources/proxy/uid/sm-ds/sm/probe/list", httpContext.Requests[0].URL.Path)
	assert.Equal(t, "Bearer grafana-token", httpContext.Requests[0].Header.Get("Authorization"))
}

func Test__SyntheticsClient__UpdateCheckAlerts__UsesSmPathViaProxy(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"msg":"alerts updated, sync in process"}`)),
			},
		},
	}

	client := &SyntheticsClient{
		DataSourceUID: "sm-ds",
		GrafanaClient: &Client{
			BaseURL:  "https://grafana.example.com",
			APIToken: "grafana-token",
			http:     httpContext,
		},
		http: httpContext,
	}

	err := client.UpdateCheckAlerts("101", []SyntheticCheckAlert{{Name: "ProbeFailedExecutionsTooHigh", Threshold: 1, Period: "5m"}})
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, "/api/datasources/proxy/uid/sm-ds/sm/check/101/alerts", httpContext.Requests[0].URL.Path)
}

func Test__SyntheticsClient__ListCheckAlerts__UsesSmPathViaProxy(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"alerts":[{"name":"ProbeFailedExecutionsTooHigh","threshold":1,"period":"5m"}]}`)),
			},
		},
	}

	client := &SyntheticsClient{
		DataSourceUID: "sm-ds",
		GrafanaClient: &Client{
			BaseURL:  "https://grafana.example.com",
			APIToken: "grafana-token",
			http:     httpContext,
		},
		http: httpContext,
	}

	alerts, err := client.ListCheckAlerts("101")
	require.NoError(t, err)
	require.Len(t, alerts, 1)
	assert.Equal(t, "ProbeFailedExecutionsTooHigh", alerts[0].Name)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, "/api/datasources/proxy/uid/sm-ds/sm/check/101/alerts", httpContext.Requests[0].URL.Path)
}
