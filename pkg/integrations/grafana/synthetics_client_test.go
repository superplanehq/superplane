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

func Test__SyntheticsClient__ListChecks__UsesGrafanaDatasourceProxy(t *testing.T) {
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
	}

	checks, err := client.ListChecks()
	require.NoError(t, err)
	assert.Empty(t, checks)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, "Bearer grafana-token", httpContext.Requests[0].Header.Get("Authorization"))
	assert.Equal(t, "/api/datasources/proxy/uid/sm-ds/sm/check/list", httpContext.Requests[0].URL.Path)
	assert.Equal(t, "includeAlerts=true", httpContext.Requests[0].URL.RawQuery)
}

func Test__NewSyntheticsClient__UsesGrafanaDatasourceProxy(t *testing.T) {
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
	assert.Equal(t, "prom-ds", client.MetricsDataSourceUID)
	require.NotNil(t, client.GrafanaClient)
}

func Test__SyntheticsClient__GetCheck__UsesSingleCheckPath(t *testing.T) {
	checkJSON := `{"id":101,"job":"API health","target":"https://api.example.com/health","frequency":60000,"timeout":3000,"enabled":true,"basicMetricsOnly":true,"settings":{"http":{"method":"GET"}},"probes":[1]}`
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(checkJSON)),
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
	}

	check, err := client.GetCheck("101")
	require.NoError(t, err)
	require.NotNil(t, check)
	assert.Equal(t, int64(101), check.ID)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, "/api/datasources/proxy/uid/sm-ds/sm/check/101", httpContext.Requests[0].URL.Path)
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
	}

	err := client.UpdateCheckAlerts("101", []SyntheticCheckAlert{{Name: "ProbeFailedExecutionsTooHigh", Threshold: 1, Period: "5m"}})
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, "/api/datasources/proxy/uid/sm-ds/sm/check/101/alerts", httpContext.Requests[0].URL.Path)
}

func Test__SyntheticsClient__UpdateCheckAlerts__SendsEmptyArrayForNilAlerts(t *testing.T) {
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
	}

	err := client.UpdateCheckAlerts("101", nil)
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)

	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"alerts":[]}`, string(body))
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
	}

	alerts, err := client.ListCheckAlerts("101")
	require.NoError(t, err)
	require.Len(t, alerts, 1)
	assert.Equal(t, "ProbeFailedExecutionsTooHigh", alerts[0].Name)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, "/api/datasources/proxy/uid/sm-ds/sm/check/101/alerts", httpContext.Requests[0].URL.Path)
}
