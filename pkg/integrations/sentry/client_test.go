package sentry

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Client__UpdateIssue__FallbackToOrgScopedEndpointOn404(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("{}"))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"123","status":"resolved"}`))},
		},
	}
	integration := &contexts.IntegrationContext{Configuration: map[string]any{"authToken": "token", "baseURL": "https://sentry.io"}}

	client, err := NewClient(httpCtx, integration)
	assert.NoError(t, err)

	out, err := client.UpdateIssue("my-org", "123", UpdateIssueRequest{Status: "resolved"})
	assert.NoError(t, err)
	assert.Equal(t, "123", out["id"])

	if assert.Len(t, httpCtx.Requests, 2) {
		assert.Equal(t, "/api/0/issues/123/", httpCtx.Requests[0].URL.Path)
		assert.Equal(t, "/api/0/organizations/my-org/issues/123/", httpCtx.Requests[1].URL.Path)
	}
}

func Test__Client__UpdateIssue__NoFallbackWhenOrgMissing(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("{}"))},
		},
	}
	integration := &contexts.IntegrationContext{Configuration: map[string]any{"authToken": "token", "baseURL": "https://sentry.io"}}

	client, err := NewClient(httpCtx, integration)
	assert.NoError(t, err)

	_, err = client.UpdateIssue("", "123", UpdateIssueRequest{Status: "resolved"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
	assert.Len(t, httpCtx.Requests, 1)
}
