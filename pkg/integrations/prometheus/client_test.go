package prometheus

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__NewClient(t *testing.T) {
	httpCtx := &contexts.HTTPContext{}

	t.Run("missing baseURL returns error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"authType": AuthTypeNone}}
		_, err := NewClient(httpCtx, integrationCtx)
		require.ErrorContains(t, err, "baseURL is required")
	})

	t.Run("invalid auth type returns error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"baseURL":  "https://prometheus.example.com",
			"authType": "invalid",
		}}
		_, err := NewClient(httpCtx, integrationCtx)
		require.ErrorContains(t, err, "invalid authType")
	})

	t.Run("basic auth requires username and password", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"baseURL":  "https://prometheus.example.com",
			"authType": AuthTypeBasic,
		}}
		_, err := NewClient(httpCtx, integrationCtx)
		require.ErrorContains(t, err, "username is required")
	})

	t.Run("creates bearer client", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"baseURL":     "https://prometheus.example.com/",
			"authType":    AuthTypeBearer,
			"bearerToken": "secret-token",
		}}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)
		assert.Equal(t, "https://prometheus.example.com", client.baseURL)
		assert.Equal(t, AuthTypeBearer, client.authType)
	})
}

func Test__Client__GetAlertsFromPrometheus(t *testing.T) {
	t.Run("adds bearer auth header", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{"status":"success","data":{"alerts":[{"state":"firing","labels":{"alertname":"HighLatency"}}]}}
					`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"baseURL":     "https://prometheus.example.com",
			"authType":    AuthTypeBearer,
			"bearerToken": "token-1",
		}}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)

		alerts, err := client.GetAlertsFromPrometheus()
		require.NoError(t, err)
		require.Len(t, alerts, 1)
		assert.Equal(t, "HighLatency", alerts[0].Labels["alertname"])

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "Bearer token-1", httpCtx.Requests[0].Header.Get("Authorization"))
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/api/v1/alerts")
	})

	t.Run("adds basic auth header", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status":"success","data":{"alerts":[]}}`))}},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"baseURL":  "https://prometheus.example.com",
			"authType": AuthTypeBasic,
			"username": "admin",
			"password": "password",
		}}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)

		_, err = client.GetAlertsFromPrometheus()
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		username, password, ok := httpCtx.Requests[0].BasicAuth()
		require.True(t, ok)
		assert.Equal(t, "admin", username)
		assert.Equal(t, "password", password)
	})

	t.Run("non-2xx returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`unauthorized`))}},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"baseURL":  "https://prometheus.example.com",
			"authType": AuthTypeNone,
		}}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)

		_, err = client.GetAlertsFromPrometheus()
		require.ErrorContains(t, err, "status 401")
	})

	t.Run("response too large returns error", func(t *testing.T) {
		largeBody := strings.Repeat("x", MaxResponseSize+1)
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(largeBody))}},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"baseURL":  "https://prometheus.example.com",
			"authType": AuthTypeNone,
		}}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)

		_, err = client.GetAlertsFromPrometheus()
		require.ErrorContains(t, err, "response too large")
	})

	t.Run("invalid json returns decode error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`not-json`))}},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"baseURL":  "https://prometheus.example.com",
			"authType": AuthTypeNone,
		}}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)

		_, err = client.GetAlertsFromPrometheus()
		require.ErrorContains(t, err, "failed to decode response JSON")
	})
}

func Test__Client__Query(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{"resultType":"vector","result":[]}}`)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
		"baseURL":  "https://prometheus.example.com",
		"authType": AuthTypeNone,
	}}

	client, err := NewClient(httpCtx, integrationCtx)
	require.NoError(t, err)

	_, err = client.Query("up")
	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 1)
	assert.Contains(t, httpCtx.Requests[0].URL.String(), "/api/v1/query?query=up")
}
