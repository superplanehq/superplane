package grafana

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeSyntheticFrequency_treatsInputAsSeconds(t *testing.T) {
	assert.Equal(t, int64(1000000), normalizeSyntheticFrequency(1000))
	assert.Equal(t, int64(2000000), normalizeSyntheticFrequency(2000))
}

func TestSyntheticCheckToSpecBase_preservesExactGrafanaFrequency(t *testing.T) {
	base, err := syntheticCheckToSpecBase(&SyntheticCheck{
		Job:       "API health",
		Target:    "https://api.example.com/health",
		Frequency: 1500,
		Timeout:   3000,
		Enabled:   true,
		Probes:    []int64{1},
		Settings: SyntheticCheckSettings{
			HTTP: &SyntheticCheckHTTPSettings{Method: "GET"},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, base.FrequencyMilliseconds)
	assert.Equal(t, int64(2), base.Frequency)
	assert.Equal(t, int64(1500), *base.FrequencyMilliseconds)

	payload, err := buildSyntheticCheckPayload(base)
	require.NoError(t, err)
	assert.Equal(t, int64(1500), payload.Frequency)

	base.Frequency = 1000
	base.FrequencyMilliseconds = nil
	payload, err = buildSyntheticCheckPayload(base)
	require.NoError(t, err)
	assert.Equal(t, int64(1000000), payload.Frequency)
}

func grafanaSyntheticHTTPResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func grafanaSyntheticDataSourceResponse() *http.Response {
	return grafanaSyntheticHTTPResponse(`[{"uid":"sm-ds","name":"Synthetic Monitoring","type":"synthetic-monitoring-datasource","jsonData":{"metrics":{"uid":"prom-ds"}}}]`)
}

func grafanaSyntheticCheckGetResponse(checkJSON string) *http.Response {
	return grafanaSyntheticHTTPResponse(checkJSON)
}

func grafanaGetCheckResponses(reachability string) []*http.Response {
	checkJSON := `{
		"id": 101,
		"job": "API health",
		"target": "https://api.example.com/health",
		"frequency": 60000,
		"timeout": 3000,
		"enabled": true,
		"basicMetricsOnly": true,
		"settings": {"http": {"method": "GET"}},
		"probes": [1]
	}`

	return []*http.Response{
		grafanaSyntheticDataSourceResponse(),
		grafanaSyntheticCheckGetResponse(checkJSON),
		grafanaSyntheticHTTPResponse(`{"alerts":[{"name":"ProbeFailedExecutionsTooHigh","threshold":1,"period":"5m"}]}`),
		grafanaSyntheticHTTPResponse(`{"results":{"A":{"frames":[{"data":{"values":[[1],[1438]]}}]}}}`),
		grafanaSyntheticHTTPResponse(`{"results":{"A":{"frames":[{"data":{"values":[[1],[1440]]}}]}}}`),
		grafanaSyntheticHTTPResponse(`{"results":{"A":{"frames":[{"data":{"values":[[1],[0.142]]}}]}}}`),
		grafanaSyntheticHTTPResponse(`{"results":{"A":{"frames":[{"data":{"values":[[1],[0.999]]}}]}}}`),
		grafanaSyntheticHTTPResponse(`{"results":{"A":{"frames":[{"data":{"values":[[1],[` + reachability + `]]}}]}}}`),
		grafanaSyntheticHTTPResponse(`{"results":{"A":{"frames":[{"data":{"values":[[1],[1713176700]]}}]}}}`),
		grafanaSyntheticHTTPResponse(`{"results":{"A":{"frames":[{"data":{"values":[[1],[1715768700]]}}]}}}`),
	}
}
