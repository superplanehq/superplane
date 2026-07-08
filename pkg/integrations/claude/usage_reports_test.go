package claude

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestClient_GetMessagesUsageReport_Pagination(t *testing.T) {
	page1 := `{
		"data": [{"starting_at": "2024-03-18T00:00:00Z", "ending_at": "2024-03-19T00:00:00Z", "results": [{"model": "claude-sonnet-5", "uncached_input_tokens": 10, "output_tokens": 5}]}],
		"has_more": true,
		"next_page": "cursor-1"
	}`
	page2 := `{
		"data": [{"starting_at": "2024-03-19T00:00:00Z", "ending_at": "2024-03-20T00:00:00Z", "results": [{"model": "claude-sonnet-5", "uncached_input_tokens": 20, "output_tokens": 10}]}],
		"has_more": false,
		"next_page": null
	}`

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(page1))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(page2))},
		},
	}

	client := &Client{AdminKey: "admin-key", BaseURL: defaultBaseURL, http: httpContext}

	buckets, err := client.GetMessagesUsageReport("2024-03-18T00:00:00Z", "2024-03-20T00:00:00Z", 2)
	require.NoError(t, err)
	require.Len(t, buckets, 2)

	require.Len(t, httpContext.Requests, 2)
	assert.NotContains(t, httpContext.Requests[0].URL.String(), "page=")
	assert.Contains(t, httpContext.Requests[1].URL.String(), "page=cursor-1")
	assert.Equal(t, "admin-key", httpContext.Requests[0].Header.Get("x-api-key"))
}

func TestClient_GetMessagesUsageReport_MissingAdminKey(t *testing.T) {
	client := &Client{BaseURL: defaultBaseURL, http: &contexts.HTTPContext{}}

	_, err := client.GetMessagesUsageReport("2024-03-18T00:00:00Z", "2024-03-19T00:00:00Z", 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "admin API key is not configured")
}

func TestClient_GetClaudeCodeUsageReport_LoopsPerDay(t *testing.T) {
	dayResponse := func(date string) string {
		return `{"data": [{"actor": {"type": "user_actor", "email_address": "dev@company.com"}, "date": "` + date + `", "core_metrics": {"num_sessions": 1}}], "has_more": false, "next_page": null}`
	}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(dayResponse("2024-03-18")))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(dayResponse("2024-03-19")))},
		},
	}

	client := &Client{AdminKey: "admin-key", BaseURL: defaultBaseURL, http: httpContext}

	start := time.Date(2024, 3, 18, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 3, 19, 0, 0, 0, 0, time.UTC)

	records, err := client.GetClaudeCodeUsageReport(start, end)
	require.NoError(t, err)
	require.Len(t, records, 2)

	require.Len(t, httpContext.Requests, 2)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "starting_at=2024-03-18")
	assert.Contains(t, httpContext.Requests[1].URL.String(), "starting_at=2024-03-19")
}

func TestClient_GetClaudeCodeUsageReport_MissingAdminKey(t *testing.T) {
	client := &Client{BaseURL: defaultBaseURL, http: &contexts.HTTPContext{}}

	_, err := client.GetClaudeCodeUsageReport(time.Now(), time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "admin API key is not configured")
}
