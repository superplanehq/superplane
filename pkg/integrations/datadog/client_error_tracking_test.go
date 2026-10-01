package datadog

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__Client__SearchErrorTrackingIssues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v2/error-tracking/issues/search", r.URL.Path)
		assert.Equal(t, "issue", r.URL.Query().Get("include"))
		assert.Equal(t, "test-api-key", r.Header.Get("DD-API-KEY"))
		assert.Equal(t, "test-app-key", r.Header.Get("DD-APPLICATION-KEY"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var payload errorTrackingSearchRequest
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "search_request", payload.Data.Type)
		assert.Equal(t, "timeout", payload.Data.Attributes.Query)
		assert.Equal(t, "ALL", payload.Data.Attributes.Persona)
		assert.Equal(t, []string{"OPEN", "ACKNOWLEDGED"}, payload.Data.Attributes.States)
		assert.Greater(t, payload.Data.Attributes.To, payload.Data.Attributes.From)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [{
				"id": "issue-1",
				"type": "error_tracking_search_result",
				"relationships": {"issue": {"data": {"id": "issue-1", "type": "issue"}}}
			}],
			"included": [{
				"id": "issue-1",
				"type": "issue",
				"attributes": {
					"error_type": "TimeoutError",
					"error_message": "checkout timed out",
					"service": "checkout",
					"first_seen": 1671612804001
				}
			}]
		}`))
	}))
	defer server.Close()

	client := &Client{
		APIKey:  "test-api-key",
		AppKey:  "test-app-key",
		Site:    "datadoghq.eu",
		BaseURL: server.URL,
		http:    server.Client(),
	}

	issues, err := client.SearchErrorTrackingIssues("timeout", 10)
	require.NoError(t, err)
	require.Len(t, issues, 1)
	assert.Equal(t, "issue-1", issues[0].ID)
	assert.Equal(t, "TimeoutError", issues[0].ErrorType)
	assert.Equal(t, "checkout timed out", issues[0].ErrorMessage)
	assert.Equal(t, "checkout", issues[0].Service)
	assert.Equal(t, time.UnixMilli(1671612804001).UTC(), issues[0].FirstSeen)
	assert.Equal(t, "TimeoutError: checkout timed out", issues[0].IssueTitle())
	assert.Equal(t, "https://app.datadoghq.eu/error-tracking/issue/issue-1", client.IssueURL(issues[0].ID))
}

func Test__Client__GetErrorTrackingIssue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v2/error-tracking/issues/issue-1", r.URL.Path)
		assert.Equal(t, "assignee,case,team_owners", r.URL.Query().Get("include"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"id": "issue-1",
				"type": "issue",
				"attributes": {
					"error_type": "TimeoutError",
					"error_message": "checkout timed out",
					"service": "checkout",
					"file_path": "checkout/pay.go",
					"function_name": "Charge",
					"platform": "BACKEND",
					"state": "OPEN",
					"is_crash": true,
					"languages": ["GO"],
					"first_seen": 1671612804001,
					"last_seen_version": "b6199f80"
				}
			},
			"included": [{
				"id": "user-1",
				"type": "user",
				"attributes": {"name": "Ada Lovelace"}
			}]
		}`))
	}))
	defer server.Close()

	client := &Client{
		APIKey:  "test-api-key",
		AppKey:  "test-app-key",
		Site:    "datadoghq.com",
		BaseURL: server.URL,
		http:    server.Client(),
	}

	issue, err := client.GetErrorTrackingIssue("issue-1")
	require.NoError(t, err)
	require.NotNil(t, issue)
	assert.Equal(t, "issue-1", issue.ID)
	assert.Equal(t, "TimeoutError: checkout timed out", issue.IssueTitle())
	assert.Equal(t, "checkout timed out", issue.ErrorMessage)
	assert.Equal(t, "checkout/pay.go", issue.FilePath)
	assert.Equal(t, "Charge", issue.FunctionName)
	assert.Equal(t, "BACKEND", issue.Platform)
	assert.Equal(t, "OPEN", issue.State)
	assert.True(t, issue.IsCrash)
	assert.Equal(t, []string{"GO"}, issue.Languages)
	assert.False(t, issue.FirstSeen.IsZero())
	assert.Equal(t, "b6199f80", issue.LastSeenVersion)
	assert.Equal(t, "Ada Lovelace", issue.Assignee)
	assert.Equal(t, "https://app.datadoghq.com/error-tracking/issue/issue-1", issue.URL)
}

func Test__Client__SearchErrorTrackingIssues__Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errors":["Forbidden"]}`))
	}))
	defer server.Close()

	client := &Client{
		APIKey:  "test-api-key",
		AppKey:  "test-app-key",
		Site:    "datadoghq.com",
		BaseURL: server.URL,
		http:    server.Client(),
	}

	_, err := client.SearchErrorTrackingIssues("*", 5)
	require.ErrorIs(t, err, ErrErrorTrackingForbidden)
}

func Test__Client__SearchErrorTrackingIssues__EmptyQueryUsesStar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var payload errorTrackingSearchRequest
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "*", payload.Data.Attributes.Query)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[],"included":[]}`))
	}))
	defer server.Close()

	client := &Client{
		BaseURL: server.URL,
		http:    server.Client(),
	}

	issues, err := client.SearchErrorTrackingIssues("  ", 5)
	require.NoError(t, err)
	assert.Empty(t, issues)
}

func Test__Datadog__Instructions__NamesApplicationKeyScopes(t *testing.T) {
	instructions := (&Datadog{}).Instructions()
	assert.Contains(t, instructions, "The API key has no scopes.")
	assert.Contains(t, instructions, "Grant the minimum permissions")
	assert.Contains(t, instructions, "create_webhooks")
	assert.Contains(t, instructions, "manage_integrations")
	assert.Contains(t, instructions, "error_tracking_read")
	assert.Contains(t, instructions, "monitors_read")
	assert.Contains(t, instructions, "monitors_write")
	assert.NotContains(t, instructions, "Add the webhook to a monitor")
	assert.Contains(t, instructions, "apm_read")
	assert.Contains(t, instructions, "logs_read_data")
	assert.Contains(t, instructions, "rum_apps_read")
	assert.NotContains(t, instructions, "error_tracking_write")
	assert.NotContains(t, instructions, "Webhooks Write")

	var appKeyDescription string
	for _, field := range (&Datadog{}).Configuration() {
		if field.Name == "appKey" {
			appKeyDescription = field.Description
		}
	}
	assert.Equal(t, "A restricted key needs create_webhooks, manage_integrations, error_tracking_read, monitors_read, and monitors_write. Add apm_read, logs_read_data, and rum_apps_read to include the error sample and related logs.", appKeyDescription)
}
