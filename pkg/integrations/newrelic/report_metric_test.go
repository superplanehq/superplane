package newrelic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestReportMetric_Name(t *testing.T) {
	component := &ReportMetric{}
	assert.Equal(t, "newrelic.reportMetric", component.Name())
}

func TestReportMetric_Label(t *testing.T) {
	component := &ReportMetric{}
	assert.Equal(t, "Report Metric", component.Label())
}

func TestReportMetric_Configuration(t *testing.T) {
	component := &ReportMetric{}
	config := component.Configuration()

	assert.NotEmpty(t, config)

	// Verify required fields
	var metricNameField, metricTypeField, valueField *bool
	var intervalFieldFound bool
	for _, field := range config {
		if field.Name == "metricName" {
			metricNameField = &field.Required
		}
		if field.Name == "metricType" {
			metricTypeField = &field.Required
		}
		if field.Name == "value" {
			valueField = &field.Required
		}
		if field.Name == "intervalMs" {
			intervalFieldFound = true
		}
	}

	require.NotNil(t, metricNameField)
	assert.True(t, *metricNameField)
	require.NotNil(t, metricTypeField)
	assert.True(t, *metricTypeField)
	require.NotNil(t, valueField)
	assert.True(t, *valueField)
	assert.True(t, intervalFieldFound, "intervalMs field should be present in configuration")
}

func TestReportMetric_Setup_IntervalValidation(t *testing.T) {
	component := &ReportMetric{}

	t.Run("gauge does not require intervalMs", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"metricName": "test.metric",
				"metricType": "gauge",
				"value":      1.0,
			},
		})

		require.NoError(t, err)
	})

	t.Run("count requires intervalMs", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"metricName": "test.metric",
				"metricType": "count",
				"value":      1.0,
				// intervalMs missing
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "intervalMs is required")
	})

	t.Run("summary requires intervalMs", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"metricName": "test.metric",
				"metricType": "summary",
				"value":      1.0,
				// intervalMs missing
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "intervalMs is required")
	})

	t.Run("count with intervalMs passes validation", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"metricName": "test.metric",
				"metricType": "count",
				"value":      1.0,
				"intervalMs": 60000,
			},
		})

		require.NoError(t, err)
	})
}

func TestClient_ReportMetric(t *testing.T) {
	t.Run("successful request -> reports metric", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"requestId":"123"}`)),
					Header:     make(http.Header),
				},
			},
		}

		client := &Client{
			APIKey:        "test-key",
			MetricBaseURL: "https://metric-api.newrelic.com/metric/v1",
			http:          httpCtx,
		}

		batch := []MetricBatch{
			{
				Metrics: []Metric{
					{
						Name:  "test.metric",
						Type:  MetricTypeGauge,
						Value: 42.5,
						Attributes: map[string]any{
							"host": "server1",
						},
					},
				},
			},
		}

		err := client.ReportMetric(context.Background(), batch)

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://metric-api.newrelic.com/metric/v1", httpCtx.Requests[0].URL.String())
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		// test-key is not an NRAK key, so it should use X-License-Key
		assert.Equal(t, "", httpCtx.Requests[0].Header.Get("Api-Key"))
		assert.Equal(t, "test-key", httpCtx.Requests[0].Header.Get("X-License-Key"))
		assert.Equal(t, "application/json", httpCtx.Requests[0].Header.Get("Content-Type"))

		// Verify request body
		bodyBytes, _ := io.ReadAll(httpCtx.Requests[0].Body)
		var sentBatch []MetricBatch
		err = json.Unmarshal(bodyBytes, &sentBatch)
		require.NoError(t, err)
		require.Len(t, sentBatch, 1)
		require.Len(t, sentBatch[0].Metrics, 1)
		assert.Equal(t, "test.metric", sentBatch[0].Metrics[0].Name)
		assert.Equal(t, MetricTypeGauge, sentBatch[0].Metrics[0].Type)
		assert.Equal(t, float64(42.5), sentBatch[0].Metrics[0].Value)
	})

	t.Run("User API Key (NRAK) request -> uses Api-Key header", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"requestId":"123"}`)),
					Header:     make(http.Header),
				},
			},
		}

		client := &Client{
			APIKey:        "NRAK-test-key",
			MetricBaseURL: "https://metric-api.newrelic.com/metric/v1",
			http:          httpCtx,
		}

		batch := []MetricBatch{
			{
				Metrics: []Metric{
					{
						Name:  "test.metric",
						Type:  MetricTypeGauge,
						Value: 42.5,
					},
				},
			},
		}

		err := client.ReportMetric(context.Background(), batch)

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://metric-api.newrelic.com/metric/v1", httpCtx.Requests[0].URL.String())
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Equal(t, "NRAK-test-key", httpCtx.Requests[0].Header.Get("Api-Key"))
		assert.Equal(t, "", httpCtx.Requests[0].Header.Get("X-License-Key"))
		assert.Equal(t, "application/json", httpCtx.Requests[0].Header.Get("Content-Type"))
	})

	t.Run("successful request with common attributes -> reports metric", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"requestId":"123"}`)),
					Header:     make(http.Header),
				},
			},
		}

		client := &Client{
			APIKey:        "test-key",
			MetricBaseURL: "https://metric-api.newrelic.com/metric/v1",
			http:          httpCtx,
		}

		common := map[string]any{"app": "test-app"}
		batch := []MetricBatch{
			{
				Common: &common,
				Metrics: []Metric{
					{
						Name:  "test.metric",
						Type:  MetricTypeGauge,
						Value: 42.5,
					},
				},
			},
		}

		err := client.ReportMetric(context.Background(), batch)

		require.NoError(t, err)

		// Verify request body contains common attributes
		bodyBytes, _ := io.ReadAll(httpCtx.Requests[0].Body)
		var sentBatch []MetricBatch
		err = json.Unmarshal(bodyBytes, &sentBatch)
		require.NoError(t, err)
		require.NotNil(t, sentBatch[0].Common)
		assert.Equal(t, "test-app", (*sentBatch[0].Common)["app"])
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"error":{"title":"Bad Request","message":"Invalid metric format"}}`)),
					Header:     make(http.Header),
				},
			},
		}

		client := &Client{
			APIKey:        "test-key",
			MetricBaseURL: "https://metric-api.newrelic.com/metric/v1",
			http:          httpCtx,
		}

		batch := []MetricBatch{
			{
				Metrics: []Metric{
					{
						Name:  "test.metric",
						Type:  MetricTypeGauge,
						Value: 42.5,
					},
				},
			},
		}

		err := client.ReportMetric(context.Background(), batch)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Bad Request")
	})

	t.Run("EU region -> uses EU endpoint", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"requestId":"456"}`)),
					Header:     make(http.Header),
				},
			},
		}

		client := &Client{
			APIKey:        "eu-test-key",
			MetricBaseURL: "https://metric-api.eu.newrelic.com/metric/v1",
			http:          httpCtx,
		}

		batch := []MetricBatch{
			{
				Metrics: []Metric{
					{
						Name:  "eu.test.metric",
						Type:  MetricTypeCount,
						Value: 100,
					},
				},
			},
		}

		err := client.ReportMetric(context.Background(), batch)

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://metric-api.eu.newrelic.com/metric/v1", httpCtx.Requests[0].URL.String())
	})
}

func TestReportMetric_Execute_SetsIntervalMsForCountAndSummary(t *testing.T) {
	component := &ReportMetric{}

	t.Run("count metric includes interval.ms in payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Body:       io.NopCloser(strings.NewReader(`{"requestId":"abc"}`)),
					Header:     make(http.Header),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-key",
				"site":   "US",
			},
		}

		execState := &contexts.ExecutionStateContext{}
		execCtx := core.ExecutionContext{
			Configuration: map[string]any{
				"metricName": "test.count",
				"metricType": "count",
				"value":      5.0,
				"intervalMs": 60000,
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(execCtx)

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)

		bodyBytes, _ := io.ReadAll(httpCtx.Requests[0].Body)
		var sentBatch []MetricBatch
		err = json.Unmarshal(bodyBytes, &sentBatch)
		require.NoError(t, err)
		require.Len(t, sentBatch, 1)
		require.Len(t, sentBatch[0].Metrics, 1)
		assert.Equal(t, int64(60000), sentBatch[0].Metrics[0].IntervalMs)
		assert.Equal(t, MetricTypeCount, sentBatch[0].Metrics[0].Type)
	})

	t.Run("summary metric requires intervalMs at execute time", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Body:       io.NopCloser(strings.NewReader(`{"requestId":"abc"}`)),
					Header:     make(http.Header),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-key",
				"site":   "US",
			},
		}

		execCtx := core.ExecutionContext{
			Configuration: map[string]any{
				"metricName": "test.summary",
				"metricType": "summary",
				"value":      5.0,
				// missing intervalMs
			},
			HTTP:        httpCtx,
			Integration: integrationCtx,
		}

		err := component.Execute(execCtx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "intervalMs is required")
	})
}

func TestNewClient_MetricBaseURL(t *testing.T) {
	t.Run("US region -> sets US metric URL", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-key",
				"site":   "US",
			},
		}

		client, err := NewClient(&contexts.HTTPContext{}, integrationCtx)

		require.NoError(t, err)
		assert.Equal(t, "https://metric-api.newrelic.com/metric/v1", client.MetricBaseURL)
	})

	t.Run("EU region -> sets EU metric URL", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-key",
				"site":   "EU",
			},
		}

		client, err := NewClient(&contexts.HTTPContext{}, integrationCtx)

		require.NoError(t, err)
		assert.Equal(t, "https://metric-api.eu.newrelic.com/metric/v1", client.MetricBaseURL)
	})
}
