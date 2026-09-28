package datadog

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelemetryIDs_IncludesHexAndDecimalLower64Bits(t *testing.T) {
	ids := telemetryIDs("cf2c57cfc127be5a1f156480875acc0a")
	assert.Equal(t, []string{
		"cf2c57cfc127be5a1f156480875acc0a",
		"1f156480875acc0a",
		"2239806892876155914",
	}, ids)
}

func TestTelemetryIDs_ConvertsHexSpanToDecimal(t *testing.T) {
	ids := telemetryIDs("79c41ba11e216970")
	assert.Equal(t, []string{
		"79c41ba11e216970",
		"8774168352833759600",
	}, ids)
}

func TestTelemetryIDs_ConvertsDecimalToHex(t *testing.T) {
	ids := telemetryIDs("8774168352833759600")
	assert.Equal(t, []string{
		"8774168352833759600",
		"79c41ba11e216970",
	}, ids)
}

func TestRelatedLogsQuery_PrefersTraceID(t *testing.T) {
	query := relatedLogsQuery(&ErrorSample{
		TraceID: "cf2c57cfc127be5a1f156480875acc0a",
		SpanID:  "79c41ba11e216970",
	})
	assert.Equal(t,
		"trace_id:(cf2c57cfc127be5a1f156480875acc0a OR 1f156480875acc0a OR 2239806892876155914)",
		query,
	)
}

func TestRelatedLogsQuery_FallsBackToSpanID(t *testing.T) {
	query := relatedLogsQuery(&ErrorSample{SpanID: "79c41ba11e216970"})
	assert.Equal(t, "span_id:(79c41ba11e216970 OR 8774168352833759600)", query)
}

func TestParseErrorSample_FromSpanShape(t *testing.T) {
	var decoded any
	require.NoError(t, json.Unmarshal([]byte(`{
		"data": [{
			"attributes": {
				"custom": {
					"env": "development",
					"resource_name": "POST /api/debug/datadog-error",
					"http": {"method": "POST", "path": "/api/debug/datadog-error", "status_code": 500},
					"error": {
						"stack": "goroutine 1 [running]:\nmain.Charge(checkout/pay.go:22)",
						"fingerprint": "783228ea-d3e4-4a21-9ccd-477b80b24c19"
					},
					"usr": {"id": "user_01a0e233-4168-79cd-9a81-1609f1cd2d5e"},
					"request_id": "d8cfa4d5-b02b-4a92-a593-2be390df3358"
				},
				"start_timestamp": "2026-09-27T19:51:32.010Z",
				"service": "myt-home-api"
			}
		}]
	}`), &decoded))

	sample := parseErrorSample(decoded, "span")
	require.NotNil(t, sample)
	assert.Equal(t, "span", sample.Source)
	assert.Equal(t, "development", sample.Env)
	assert.Equal(t, "POST /api/debug/datadog-error", sample.Resource)
	assert.Equal(t, "POST", sample.HTTPMethod)
	assert.Equal(t, "/api/debug/datadog-error", sample.HTTPPath)
	assert.Equal(t, "500", sample.HTTPStatus)
	assert.Equal(t, "user_01a0e233-4168-79cd-9a81-1609f1cd2d5e", sample.UserID)
	assert.Equal(t, "d8cfa4d5-b02b-4a92-a593-2be390df3358", sample.RequestID)
	assert.Equal(t, "783228ea-d3e4-4a21-9ccd-477b80b24c19", sample.Fingerprint)
	assert.Contains(t, sample.Stack, "main.Charge")
	assert.Equal(t, time.Date(2026, 9, 27, 19, 51, 32, 10_000_000, time.UTC), sample.Timestamp)
}

func TestParseErrorSample_FromLogShape(t *testing.T) {
	var decoded any
	require.NoError(t, json.Unmarshal([]byte(`{
		"data": [{
			"attributes": {
				"timestamp": "2026-09-27T19:51:32.010Z",
				"message": "myt-home dogfood error 783228ea-d3e4-4a21-9ccd-477b80b24c19",
				"status": "error",
				"service": "myt-home-api",
				"host": "api-1",
				"attributes": {
					"http": {"method": "POST", "path": "/api/debug/datadog-error", "status": 500},
					"otel": {
						"span_id": "79c41ba11e216970",
						"trace_id": "cf2c57cfc127be5a1f156480875acc0a"
					},
					"request_id": "d8cfa4d5-b02b-4a92-a593-2be390df3358",
					"usr": {"id": "user_01a0e233-4168-79cd-9a81-1609f1cd2d5e"},
					"error": {
						"kind": "DogfoodError",
						"fingerprint": "783228ea-d3e4-4a21-9ccd-477b80b24c19"
					},
					"exception": {
						"stacktrace": "goroutine 121 [running]:\nruntime/debug.Stack()"
					},
					"env": "development"
				}
			}
		}]
	}`), &decoded))

	sample := parseErrorSample(decoded, "log")
	require.NotNil(t, sample)
	assert.Equal(t, "log", sample.Source)
	assert.Equal(t, "cf2c57cfc127be5a1f156480875acc0a", sample.TraceID)
	assert.Equal(t, "79c41ba11e216970", sample.SpanID)
	assert.Equal(t, "development", sample.Env)
	assert.Equal(t, "api-1", sample.Host)
	assert.Equal(t, "POST", sample.HTTPMethod)
	assert.Equal(t, "/api/debug/datadog-error", sample.HTTPPath)
	assert.Equal(t, "500", sample.HTTPStatus)
	assert.Equal(t, "user_01a0e233-4168-79cd-9a81-1609f1cd2d5e", sample.UserID)
	assert.Contains(t, sample.Stack, "runtime/debug.Stack")
}

func TestIssueSampleQuery_MatchesReservedAndAttributeIssueID(t *testing.T) {
	assert.Equal(t,
		"issue.id:af23f87c-bb36-11f1-b4a8-da7ad0900005 OR @issue.id:af23f87c-bb36-11f1-b4a8-da7ad0900005",
		issueSampleQuery(" af23f87c-bb36-11f1-b4a8-da7ad0900005 "),
	)
}

func TestParseErrorSample_FromFrontendLog(t *testing.T) {
	var decoded any
	require.NoError(t, json.Unmarshal([]byte(`{
		"data": [{
			"attributes": {
				"timestamp": "2026-09-28T12:18:14.312Z",
				"message": "✕ GET /budget-categories network failure (340ms): Failed to fetch",
				"service": "myt-home-api",
				"status": "error",
				"attributes": {
					"breadcrumb": {
						"00": "click IdeaUncategorized · receipt",
						"01": "navigation  /app/transactions/txn_01a0e329"
					},
					"env": "development",
					"fe": {
						"action": "GET /budget-categories",
						"route": "/app/transactions/txn_01a0e329",
						"request_id": "bd8a7101-8f3c-476f-acf2-0d2a3d84ef18"
					},
					"otel": {"span_id": "e718eaa2d795a307", "trace_id": "f4235b16fc304d9216431405eebf5cb3"},
					"request_id": "50bacc3e-4980-4434-aae9-e42ba300c0bd",
					"source": "frontend",
					"stack": "TypeError: Failed to fetch\n    at request (src/lib/api.ts:80:17)\n    at list (src/lib/api.ts:229:30)",
					"usr": {"id": "user_01a0e233-4168-79cd-9a81-1609f1cd2d5e"}
				}
			}
		}]
	}`), &decoded))

	sample := parseErrorSample(decoded, "log")
	require.NotNil(t, sample)
	assert.Equal(t, "f4235b16fc304d9216431405eebf5cb3", sample.TraceID)
	assert.Equal(t, "/app/transactions/txn_01a0e329", sample.Route)
	assert.Equal(t, "GET /budget-categories", sample.Action)
	assert.Equal(t, "frontend", sample.Origin)
	assert.Equal(t, "50bacc3e-4980-4434-aae9-e42ba300c0bd", sample.RequestID)
	assert.Equal(t, []string{
		"click IdeaUncategorized · receipt",
		"navigation  /app/transactions/txn_01a0e329",
	}, sample.Breadcrumbs)
	assert.Contains(t, sample.Stack, "src/lib/api.ts:80:17")
}

func TestParseRelatedLogs(t *testing.T) {
	var decoded any
	require.NoError(t, json.Unmarshal([]byte(`{
		"data": [
			{
				"attributes": {
					"timestamp": "2026-09-27T19:51:32.010Z",
					"status": "error",
					"service": "myt-home-api",
					"message": "POST /api/debug/datadog-error -> 500 (0.2ms)",
					"attributes": {
						"http": {"method": "POST", "path": "/api/debug/datadog-error", "status": 500}
					}
				}
			},
			{
				"attributes": {
					"timestamp": "2026-09-27T19:51:32.010Z",
					"status": "error",
					"service": "myt-home-api",
					"message": "myt-home dogfood error 783228ea-d3e4-4a21-9ccd-477b80b24c19",
					"attributes": {"error": {"kind": "DogfoodError"}}
				}
			}
		]
	}`), &decoded))

	logs := parseRelatedLogs(decoded)
	require.Len(t, logs, 2)
	assert.Equal(t, "error", logs[0].Status)
	assert.Equal(t, "myt-home-api", logs[0].Service)
	assert.Equal(t, "POST /api/debug/datadog-error -> 500 (0.2ms)", logs[0].Message)
	assert.Equal(t, "POST", logs[0].HTTPMethod)
	assert.Equal(t, "500", logs[0].HTTPStatus)
	assert.Equal(t, "DogfoodError", logs[1].ErrorKind)
}

func TestLoadErrorTrackingIssue_LoadsSampleAndRelatedLogs(t *testing.T) {
	var logsQuery string
	var logsFrom string
	var logsTo string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v2/error-tracking/issues/"):
			_, _ = w.Write([]byte(`{"data":{"id":"issue-1","type":"issue","attributes":{"error_type":"TimeoutError","error_message":"checkout timed out","service":"checkout","platform":"BACKEND"}}}`))
		case r.URL.Path == "/api/v2/error-tracking/issues/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.Contains(t, string(body), `"query":"issue.id:issue-1"`)
			_, _ = w.Write([]byte(`{"data":[{"id":"issue-1","attributes":{"total_count":2},"relationships":{"issue":{"data":{"id":"issue-1"}}}}]}`))
		case r.URL.Path == "/api/v2/spans/events/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.Contains(t, string(body), `"type":"search_request"`)
			assert.Contains(t, string(body), `issue.id:issue-1 OR @issue.id:issue-1`)
			_, _ = w.Write([]byte(`{"data":[{"attributes":{"custom":{"otel":{"trace_id":"cf2c57cfc127be5a1f156480875acc0a","span_id":"79c41ba11e216970"},"env":"development","resource_name":"POST /api/debug/datadog-error","error":{"stack":"goroutine 1 [running]:\nmain.Charge(checkout/pay.go:22)"}},"start_timestamp":"2026-09-27T19:51:32.010Z"}}]}`))
		case r.URL.Path == "/api/v2/logs/events/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var payload map[string]any
			require.NoError(t, json.Unmarshal(body, &payload))
			assert.NotContains(t, payload, "data")
			filter := payload["filter"].(map[string]any)
			logsQuery, _ = filter["query"].(string)
			logsFrom, _ = filter["from"].(string)
			logsTo, _ = filter["to"].(string)
			assert.Equal(t, "timestamp", payload["sort"])
			_, _ = w.Write([]byte(`{"data":[{"attributes":{"timestamp":"2026-09-27T19:51:32.010Z","status":"error","service":"checkout","message":"POST /api/debug/datadog-error -> 500 (0.2ms)"}}]}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, Site: "datadoghq.eu", http: server.Client()}
	issue, err := client.LoadErrorTrackingIssue("issue-1", nil)
	require.NoError(t, err)
	assert.True(t, issue.HasActivity)
	assert.Equal(t, int64(2), issue.TotalCount)
	require.NotNil(t, issue.Sample)
	assert.Equal(t, "cf2c57cfc127be5a1f156480875acc0a", issue.Sample.TraceID)
	assert.Contains(t, issue.Stack, "main.Charge")
	require.Len(t, issue.RelatedLogs, 1)
	assert.Contains(t, issue.RelatedLogs[0].Message, "POST /api/debug/datadog-error")
	assert.Contains(t, logsQuery, "trace_id:")
	assert.Contains(t, logsQuery, "2239806892876155914")
	assert.Contains(t, logsFrom, "2026-09-27T19:36:32")
	assert.Contains(t, logsTo, "2026-09-27T20:06:32")
}

func TestLoadErrorTrackingIssue_KeepsIssueWhenSampleIsForbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v2/error-tracking/issues/"):
			_, _ = w.Write([]byte(`{"data":{"id":"issue-1","type":"issue","attributes":{"error_type":"TimeoutError","error_message":"checkout timed out","service":"checkout"}}}`))
		case r.URL.Path == "/api/v2/error-tracking/issues/search":
			_, _ = w.Write([]byte(`{"data":[]}`))
		default:
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"errors":["Forbidden"]}`))
		}
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, Site: "datadoghq.com", http: server.Client()}
	var warnings []string
	issue, err := client.LoadErrorTrackingIssue("issue-1", func(format string, args ...any) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	})
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "error sample")
	assert.Equal(t, "issue-1", issue.ID)
	assert.Equal(t, "TimeoutError: checkout timed out", issue.IssueTitle())
	assert.Nil(t, issue.Sample)
	assert.Empty(t, issue.RelatedLogs)
}

func TestTraceURLAndLogsURL(t *testing.T) {
	client := &Client{Site: "datadoghq.eu"}
	assert.Equal(t, "https://app.datadoghq.eu/apm/trace/2239806892876155914", client.TraceURL("cf2c57cfc127be5a1f156480875acc0a"))

	from := time.Date(2026, 9, 27, 19, 36, 32, 10_000_000, time.UTC)
	to := time.Date(2026, 9, 27, 20, 6, 32, 10_000_000, time.UTC)
	logsURL := client.LogsURL("span_id:(79c41ba11e216970 OR 8774168352833759600)", from, to)
	assert.Contains(t, logsURL, "https://app.datadoghq.eu/logs?")
	assert.Contains(t, logsURL, "live=false")
	assert.Contains(t, logsURL, "from_ts=1790537792010")
	assert.Contains(t, logsURL, "to_ts=1790539592010")
	assert.Contains(t, logsURL, "query=")
}
