package datadog

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Datadog__Sync(t *testing.T) {
	d := &Datadog{}

	t.Run("no site -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":   "",
				"apiKey": "test-api-key",
				"appKey": "test-app-key",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "site is required")
	})

	t.Run("no apiKey -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":   "datadoghq.com",
				"apiKey": "",
				"appKey": "test-app-key",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "apiKey is required")
	})

	t.Run("no appKey -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":   "datadoghq.com",
				"apiKey": "test-api-key",
				"appKey": "",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "appKey is required")
	})

	t.Run("successful validation configures webhook and becomes ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"valid": true}`)),
				},
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"errors":["Not found"]}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"name":"superplane"}`)),
				},
			},
		}

		integrationID := uuid.New()
		appCtx := &contexts.IntegrationContext{
			IntegrationID: integrationID.String(),
			Configuration: map[string]any{
				"site":   "datadoghq.com",
				"apiKey": "test-api-key",
				"appKey": "test-app-key",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			HTTP:            httpContext,
			Integration:     appCtx,
			BaseURL:         "https://app.example.com",
			WebhooksBaseURL: "https://hooks.example.com",
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 3)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "api.datadoghq.com/api/v1/validate")
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/webhooks/configuration/webhooks/superplane")
		assert.Equal(t, http.MethodPut, httpContext.Requests[1].Method)
		assert.Contains(t, httpContext.Requests[2].URL.String(), "/webhooks/configuration/webhooks")
		assert.Equal(t, http.MethodPost, httpContext.Requests[2].Method)

		token, err := webhookToken(appCtx)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		body, err := io.ReadAll(httpContext.Requests[2].Body)
		require.NoError(t, err)
		var webhookConfig WebhookConfiguration
		require.NoError(t, json.Unmarshal(body, &webhookConfig))
		assert.Equal(t, IntegrationWebhookName, webhookConfig.Name)
		assert.Contains(t, webhookConfig.URL, integrationID.String())
		assert.Contains(t, webhookConfig.CustomHeaders, token)
		assert.Contains(t, webhookConfig.Payload, "$ALERT_QUERY")
		assert.Contains(t, webhookConfig.Payload, "$ALERT_SCOPE")
	})

	t.Run("EU site -> uses correct base URL", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"valid": true}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"name":"superplane"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			IntegrationID: uuid.New().String(),
			Configuration: map[string]any{
				"site":   "datadoghq.eu",
				"apiKey": "test-api-key",
				"appKey": "test-app-key",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
			BaseURL:       "https://app.example.com",
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "api.datadoghq.eu/api/v1/validate")
	})

	t.Run("validation failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"errors": ["Invalid API key"]}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":   "datadoghq.com",
				"apiKey": "invalid-key",
				"appKey": "test-app-key",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid credentials")
		assert.NotEqual(t, "ready", appCtx.State)
	})
}

func Test__Datadog__HandleRequest(t *testing.T) {
	d := &Datadog{}
	integrationID := uuid.New()
	organizationID := uuid.New().String()

	t.Run("dispatches triggered error tracking alerts", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			IntegrationID: integrationID.String(),
			CurrentSecrets: map[string]core.IntegrationSecret{
				WebhookSecretName: {Name: WebhookSecretName, Value: []byte("secret-token")},
			},
			Subscriptions: []contexts.Subscription{
				{ID: uuid.New(), Configuration: SubscriptionConfiguration{}},
			},
		}

		body := `{
			"event_type":"error_tracking_alert",
			"alert_transition":"Triggered",
			"title":"[Triggered] checkout new issues",
			"body":"InventoryTimeout: checkout failed"
		}`
		request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/"+integrationID.String()+"/events", strings.NewReader(body))
		request.Header.Set(WebhookHeaderName, "secret-token")
		recorder := httptest.NewRecorder()
		logs := captureDatadogWebhookLogs(t)

		d.HandleRequest(core.HTTPRequestContext{
			OrganizationID: organizationID,
			Integration:    appCtx,
			Request:        request,
			Response:       recorder,
			Logger:         logrus.NewEntry(logrus.New()),
		})

		assert.Equal(t, http.StatusOK, recorder.Code)
		lines := datadogWebhookLogLines(t, logs.String())
		require.Len(t, lines, 1)
		assertDatadogReceipt(t, lines[0], organizationID, integrationID.String(), "received", "error_tracking_alert", "Triggered")
		assert.Equal(t, "info", lines[0]["level"])
		assert.NotContains(t, logs.String(), "secret-token")
		assert.NotContains(t, logs.String(), "InventoryTimeout")
	})

	t.Run("rejects invalid token", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			IntegrationID: integrationID.String(),
			CurrentSecrets: map[string]core.IntegrationSecret{
				WebhookSecretName: {Name: WebhookSecretName, Value: []byte("secret-token")},
			},
		}

		request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/"+integrationID.String()+"/events", strings.NewReader(`{}`))
		request.Header.Set(WebhookHeaderName, "wrong-token")
		recorder := httptest.NewRecorder()
		logs := captureDatadogWebhookLogs(t)

		d.HandleRequest(core.HTTPRequestContext{
			OrganizationID: organizationID,
			Integration:    appCtx,
			Request:        request,
			Response:       recorder,
			Logger:         logrus.NewEntry(logrus.New()),
		})

		assert.Equal(t, http.StatusForbidden, recorder.Code)
		lines := datadogWebhookLogLines(t, logs.String())
		require.Len(t, lines, 1)
		assertDatadogReceipt(t, lines[0], organizationID, integrationID.String(), "rejected", "unknown", "")
		assert.Equal(t, "warning", lines[0]["level"])
		assert.NotContains(t, logs.String(), "wrong-token")
		assert.NotContains(t, logs.String(), "secret-token")
	})

	t.Run("dispatches re-triggered error tracking alerts", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			IntegrationID: integrationID.String(),
			CurrentSecrets: map[string]core.IntegrationSecret{
				WebhookSecretName: {Name: WebhookSecretName, Value: []byte("secret-token")},
			},
			Subscriptions: []contexts.Subscription{
				{ID: uuid.New(), Configuration: SubscriptionConfiguration{}},
			},
		}

		body := `{"event_type":"error_tracking_alert","alert_transition":"Re-Triggered"}`
		request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/"+integrationID.String()+"/events", strings.NewReader(body))
		request.Header.Set(WebhookHeaderName, "secret-token")
		recorder := httptest.NewRecorder()

		d.HandleRequest(core.HTTPRequestContext{
			Integration: appCtx,
			Request:     request,
			Response:    recorder,
			Logger:      logrus.NewEntry(logrus.New()),
		})

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("returns an error when a subscription does not accept the alert", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			IntegrationID: integrationID.String(),
			CurrentSecrets: map[string]core.IntegrationSecret{
				WebhookSecretName: {Name: WebhookSecretName, Value: []byte("secret-token")},
			},
			Subscriptions: []contexts.Subscription{
				{ID: uuid.New(), Configuration: SubscriptionConfiguration{}, SendErr: errors.New("queue is full")},
			},
		}

		body := `{"event_type":"error_tracking_alert","alert_transition":"Triggered"}`
		request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/"+integrationID.String()+"/events", strings.NewReader(body))
		request.Header.Set(WebhookHeaderName, "secret-token")
		recorder := httptest.NewRecorder()
		logs := captureDatadogWebhookLogs(t)

		d.HandleRequest(core.HTTPRequestContext{
			OrganizationID: organizationID,
			Integration:    appCtx,
			Request:        request,
			Response:       recorder,
			Logger:         logrus.NewEntry(logrus.New()),
		})

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		lines := datadogWebhookLogLines(t, logs.String())
		require.Len(t, lines, 1)
		assertDatadogReceipt(t, lines[0], organizationID, integrationID.String(), "failed", "error_tracking_alert", "Triggered")
		assert.Equal(t, "error", lines[0]["level"])
		assert.Contains(t, lines[0]["error"], "queue is full")
	})

	t.Run("acknowledges the alert when another subscription already accepted it", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			IntegrationID: integrationID.String(),
			CurrentSecrets: map[string]core.IntegrationSecret{
				WebhookSecretName: {Name: WebhookSecretName, Value: []byte("secret-token")},
			},
			Subscriptions: []contexts.Subscription{
				{ID: uuid.New(), Configuration: SubscriptionConfiguration{}, SendErr: errors.New("queue is full")},
				{ID: uuid.New(), Configuration: SubscriptionConfiguration{}},
			},
		}

		body := `{"event_type":"error_tracking_alert","alert_transition":"Triggered"}`
		request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/"+integrationID.String()+"/events", strings.NewReader(body))
		request.Header.Set(WebhookHeaderName, "secret-token")
		recorder := httptest.NewRecorder()
		logs := captureDatadogWebhookLogs(t)

		d.HandleRequest(core.HTTPRequestContext{
			OrganizationID: organizationID,
			Integration:    appCtx,
			Request:        request,
			Response:       recorder,
			Logger:         logrus.NewEntry(logrus.New()),
		})

		assert.Equal(t, http.StatusOK, recorder.Code)
		lines := datadogWebhookLogLines(t, logs.String())
		require.Len(t, lines, 1)
		assertDatadogReceipt(t, lines[0], organizationID, integrationID.String(), "received", "error_tracking_alert", "Triggered")
		assert.Equal(t, "error", lines[0]["level"])
		assert.Contains(t, lines[0]["error"], "queue is full")
	})

	t.Run("ignores recovered alerts", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			IntegrationID: integrationID.String(),
			CurrentSecrets: map[string]core.IntegrationSecret{
				WebhookSecretName: {Name: WebhookSecretName, Value: []byte("secret-token")},
			},
			Subscriptions: []contexts.Subscription{
				{ID: uuid.New(), Configuration: SubscriptionConfiguration{}},
			},
		}

		body := `{"event_type":"error_tracking_alert","alert_transition":"Recovered"}`
		request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/"+integrationID.String()+"/events", strings.NewReader(body))
		request.Header.Set(WebhookHeaderName, "secret-token")
		recorder := httptest.NewRecorder()
		logs := captureDatadogWebhookLogs(t)

		d.HandleRequest(core.HTTPRequestContext{
			OrganizationID: organizationID,
			Integration:    appCtx,
			Request:        request,
			Response:       recorder,
			Logger:         logrus.NewEntry(logrus.New()),
		})

		assert.Equal(t, http.StatusOK, recorder.Code)
		lines := datadogWebhookLogLines(t, logs.String())
		require.Len(t, lines, 1)
		assertDatadogReceipt(t, lines[0], organizationID, integrationID.String(), "ignored", "error_tracking_alert", "Recovered")
		assert.Equal(t, "info", lines[0]["level"])
	})

	t.Run("stays silent for the wrong path or method", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			IntegrationID: integrationID.String(),
		}
		logs := captureDatadogWebhookLogs(t)

		wrongPath := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/"+integrationID.String()+"/other", strings.NewReader(`{}`))
		wrongPathRecorder := httptest.NewRecorder()
		d.HandleRequest(core.HTTPRequestContext{
			OrganizationID: organizationID,
			Integration:    appCtx,
			Request:        wrongPath,
			Response:       wrongPathRecorder,
		})
		assert.Equal(t, http.StatusNotFound, wrongPathRecorder.Code)

		wrongMethod := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/"+integrationID.String()+"/events", nil)
		wrongMethodRecorder := httptest.NewRecorder()
		d.HandleRequest(core.HTTPRequestContext{
			OrganizationID: organizationID,
			Integration:    appCtx,
			Request:        wrongMethod,
			Response:       wrongMethodRecorder,
		})
		assert.Equal(t, http.StatusMethodNotAllowed, wrongMethodRecorder.Code)
		assert.Empty(t, strings.TrimSpace(logs.String()))
	})

	t.Run("logs an unreadable body as failed", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			IntegrationID: integrationID.String(),
			CurrentSecrets: map[string]core.IntegrationSecret{
				WebhookSecretName: {Name: WebhookSecretName, Value: []byte("secret-token")},
			},
		}
		request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/"+integrationID.String()+"/events", strings.NewReader(`{`))
		request.Header.Set(WebhookHeaderName, "secret-token")
		recorder := httptest.NewRecorder()
		logs := captureDatadogWebhookLogs(t)

		d.HandleRequest(core.HTTPRequestContext{
			OrganizationID: organizationID,
			Integration:    appCtx,
			Request:        request,
			Response:       recorder,
		})

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		lines := datadogWebhookLogLines(t, logs.String())
		require.Len(t, lines, 1)
		assertDatadogReceipt(t, lines[0], organizationID, integrationID.String(), "failed", "unknown", "")
		assert.Equal(t, "error", lines[0]["level"])
	})
}

func captureDatadogWebhookLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	logger := logging.DatadogWebhookLogger()
	previous := logger.Out
	buffer := &bytes.Buffer{}
	logger.SetOutput(buffer)
	t.Cleanup(func() {
		logger.SetOutput(previous)
	})
	return buffer
}

func datadogWebhookLogLines(t *testing.T, raw string) []map[string]any {
	t.Helper()

	lines := []map[string]any{}
	for _, line := range bytes.Split([]byte(raw), []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		payload := map[string]any{}
		require.NoError(t, json.Unmarshal(line, &payload))
		lines = append(lines, payload)
	}
	return lines
}

func assertDatadogReceipt(t *testing.T, line map[string]any, organizationID, integrationID, outcome, eventType, alertTransition string) {
	t.Helper()

	assert.Equal(t, logging.WebhookLogType, line["type"])
	assert.Equal(t, logging.DatadogIntegration, line["integration"])
	assert.Equal(t, outcome, line["outcome"])
	assert.Equal(t, organizationID, line["organization_id"])
	assert.Equal(t, integrationID, line["integration_id"])
	assert.Equal(t, eventType, line["event_type"])
	assert.Equal(t, alertTransition, line["alert_transition"])
	assert.Equal(t, "", line["workspace_id"])
	assert.Equal(t, "", line["intake_id"])
}
