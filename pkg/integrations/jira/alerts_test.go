package jira

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Client__OpsAlertsAPI(t *testing.T) {
	cloudID := "35273b54-3f06-40d2-880f-dd28cf6daafa"
	appCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseUrl":  "https://test.atlassian.net",
			"email":    "test@example.com",
			"apiToken": "test-token",
		},
	}

	t.Run("create alert", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"result":"Request will be processed","requestId":"r1","took":0.1}`)),
				},
			},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)
		out, err := client.CreateOpsAlert(cloudID, &OpsCreateAlertRequest{Message: "Hi"})
		require.NoError(t, err)
		assert.Equal(t, "r1", out.RequestID)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "api.atlassian.com/jsm/ops/api/"+cloudID+"/v1/alerts")
	})

	t.Run("get alert", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"a1","message":"m"}`)),
				},
			},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)
		m, err := client.GetOpsAlert(cloudID, "a1")
		require.NoError(t, err)
		assert.Equal(t, "m", m["message"])
	})

	t.Run("delete alert", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Body:       io.NopCloser(strings.NewReader(`{"requestId":"d1"}`)),
				},
			},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)
		out, err := client.DeleteOpsAlert(cloudID, "a1")
		require.NoError(t, err)
		assert.Equal(t, "d1", out.RequestID)
	})
}
