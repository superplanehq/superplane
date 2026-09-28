package datadog

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDescribeErrorTrackingIssue_IncludesSampleAndRelatedLogs(t *testing.T) {
	stamp := time.Date(2026, 9, 27, 19, 51, 32, 10_000_000, time.UTC)
	text := DescribeErrorTrackingIssue(ErrorTrackingIssue{
		ID:           "issue-1",
		ErrorType:    "*observability.trackedError",
		ErrorMessage: "myt-home dogfood error",
		Service:      "myt-home-api",
		FilePath:     "observability/datadog.go",
		FunctionName: "dogfoodHandler",
		URL:          "https://app.datadoghq.eu/error-tracking/issue/issue-1",
		Stack:        "goroutine 121 [running]:\nruntime/debug.Stack()",
		Sample: &ErrorSample{
			Timestamp:  stamp,
			Env:        "development",
			Resource:   "POST /api/debug/datadog-error",
			HTTPMethod: "POST",
			HTTPPath:   "/api/debug/datadog-error",
			HTTPStatus: "500",
			UserID:     "user_01a0e233-4168-79cd-9a81-1609f1cd2d5e",
			RequestID:  "d8cfa4d5-b02b-4a92-a593-2be390df3358",
			TraceID:    "cf2c57cfc127be5a1f156480875acc0a",
			SpanID:     "79c41ba11e216970",
			TraceURL:   "https://app.datadoghq.eu/apm/trace/2239806892876155914",
			LogsURL:    "https://app.datadoghq.eu/logs?query=trace_id%3A%28cf2c57cfc127be5a1f156480875acc0a%29&live=false",
		},
		RelatedLogs: []LogLine{
			{Timestamp: stamp, Status: "error", Service: "myt-home-api", Message: "POST /api/debug/datadog-error -> 500 (0.2ms)"},
			{Timestamp: stamp, Status: "error", Service: "myt-home-api", Message: "myt-home dogfood error 783228ea-d3e4-4a21-9ccd-477b80b24c19"},
		},
	}, AlertDetails{})

	assert.Contains(t, text, "## Error")
	assert.Contains(t, text, "## Location")
	assert.Contains(t, text, "## Error sample")
	assert.Contains(t, text, "## Stack trace")
	assert.Contains(t, text, "## Related logs")
	assert.True(t, strings.Index(text, "## Error sample") < strings.Index(text, "## Stack trace"))
	assert.True(t, strings.Index(text, "## Stack trace") < strings.Index(text, "## Related logs"))
	assert.Contains(t, text, "development")
	assert.Contains(t, text, "POST /api/debug/datadog-error -> 500")
	assert.Contains(t, text, "user_01a0e233-4168-79cd-9a81-1609f1cd2d5e")
	assert.Contains(t, text, "d8cfa4d5-b02b-4a92-a593-2be390df3358")
	assert.Contains(t, text, "View the trace")
	assert.Contains(t, text, "View logs in Datadog")
	assert.Contains(t, text, "POST /api/debug/datadog-error -> 500 (0.2ms)")
	assert.Contains(t, text, "myt-home dogfood error 783228ea")
}

func TestDescribeErrorTrackingIssue_IncludesFrontendContext(t *testing.T) {
	text := DescribeErrorTrackingIssue(ErrorTrackingIssue{
		ErrorType:    "Error",
		ErrorMessage: "GET /budget-categories network failure",
		Platform:     "UNKNOWN",
		Sample: &ErrorSample{
			Origin:      "frontend",
			Route:       "/app/transactions/txn_01a0e329",
			Action:      "GET /budget-categories",
			Breadcrumbs: []string{"click IdeaUncategorized", "navigation /app/transactions/txn_01a0e329"},
		},
	}, AlertDetails{})

	assert.Contains(t, text, "**Origin:** frontend")
	assert.Contains(t, text, "**Route:** /app/transactions/txn_01a0e329")
	assert.Contains(t, text, "**Action:** GET /budget-categories")
	assert.Contains(t, text, "**Breadcrumbs:**")
	assert.Contains(t, text, "  1. click IdeaUncategorized")
	assert.Contains(t, text, "  2. navigation /app/transactions/txn_01a0e329")
	assert.NotContains(t, text, "Platform:** Unknown")
}

func TestDescribeErrorTrackingIssue_OmitsEmptySampleAndLogs(t *testing.T) {
	text := DescribeErrorTrackingIssue(ErrorTrackingIssue{
		ErrorType:    "TimeoutError",
		ErrorMessage: "checkout timed out",
		URL:          "https://app.datadoghq.com/error-tracking/issue/issue-1",
	}, AlertDetails{})

	assert.Contains(t, text, "TimeoutError")
	assert.NotContains(t, text, "## Error sample")
	assert.NotContains(t, text, "## Related logs")
	assert.NotContains(t, text, "## Stack trace")
}
