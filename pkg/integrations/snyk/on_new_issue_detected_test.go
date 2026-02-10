package snyk

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestOnNewIssueDetectedTrigger(t *testing.T) {
	trigger := &OnNewIssueDetected{}

	assert.Equal(t, "snyk.onNewIssueDetected", trigger.Name())
	assert.Equal(t, "On New Issue Detected", trigger.Label())
	assert.Equal(t, "Listen to Snyk for new security issues", trigger.Description())
	assert.Equal(t, "shield", trigger.Icon())

	configFields := trigger.Configuration()
	assert.Len(t, configFields, 3)

	fieldNames := make(map[string]bool)
	for _, field := range configFields {
		fieldNames[field.Name] = true
	}

	expectedFields := []string{"organizationId", "projectId", "severity"}
	for _, fieldName := range expectedFields {
		assert.True(t, fieldNames[fieldName], "Missing field: %s", fieldName)
	}
}

func TestSeverityComparison(t *testing.T) {
	trigger := &OnNewIssueDetected{}

	tests := []struct {
		actual    string
		threshold string
		expected  bool
		name      string
	}{
		{"critical", "high", true, "critical is higher than high"},
		{"high", "high", true, "high equals high"},
		{"medium", "high", false, "medium is lower than high"},
		{"low", "critical", false, "low is lower than critical"},
		{"critical", "critical", true, "critical equals critical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trigger.isSeverityEqualOrHigher(tt.actual, tt.threshold)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test__OnNewIssueDetected__HandleWebhook(t *testing.T) {
	trigger := &OnNewIssueDetected{}

	t.Run("missing event header -> 400", func(t *testing.T) {
		headers := http.Header{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       headers,
			Configuration: map[string]any{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "missing Snyk event header")
	})

	t.Run("non-matching event type -> 200, no events emitted", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Snyk-Event", "ping")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte(`{}`),
			Headers:       headers,
			Configuration: map[string]any{},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("invalid JSON body -> 400", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Snyk-Event", "project_snapshot/v0")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte(`not json`),
			Headers:       headers,
			Configuration: map[string]any{},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "error parsing request body")
	})

	t.Run("no new issues -> 200, no events emitted", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Snyk-Event", "project_snapshot/v0")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte(`{"newIssues": [], "project": {"id": "p1"}}`),
			Headers:       headers,
			Configuration: map[string]any{},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("new issue detected -> event emitted", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Snyk-Event", "project_snapshot/v0")

		body := []byte(`{
			"newIssues": [
				{
					"id": "SNYK-JS-12345",
					"title": "Remote Code Execution",
					"severity": "high",
					"packageName": "lodash",
					"packageVersion": "4.17.20"
				}
			],
			"project": {"id": "project-123", "name": "my-web-app"},
			"org": {"id": "org-123", "name": "my-org"}
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "snyk.issue.detected", eventContext.Payloads[0].Type)
	})

	t.Run("multiple new issues -> multiple events emitted", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Snyk-Event", "project_snapshot/v0")

		body := []byte(`{
			"newIssues": [
				{"id": "SNYK-JS-001", "severity": "high"},
				{"id": "SNYK-JS-002", "severity": "critical"}
			],
			"project": {"id": "project-123", "name": "my-web-app"},
			"org": {"id": "org-123", "name": "my-org"}
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 2, eventContext.Count())
	})

	t.Run("severity filter -> only matching issues emitted", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Snyk-Event", "project_snapshot/v0")

		body := []byte(`{
			"newIssues": [
				{"id": "SNYK-JS-001", "severity": "low"},
				{"id": "SNYK-JS-002", "severity": "high"},
				{"id": "SNYK-JS-003", "severity": "critical"}
			],
			"project": {"id": "project-123", "name": "my-web-app"},
			"org": {"id": "org-123", "name": "my-org"}
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"severity": "high",
			},
			Events: eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 2, eventContext.Count())
	})

	t.Run("project filter -> only matching project emitted", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Snyk-Event", "project_snapshot/v0")

		body := []byte(`{
			"newIssues": [
				{"id": "SNYK-JS-001", "severity": "high"}
			],
			"project": {"id": "project-123", "name": "my-web-app"},
			"org": {"id": "org-123", "name": "my-org"}
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"projectId": "project-999",
			},
			Events: eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("project filter matches -> event emitted", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Snyk-Event", "project_snapshot/v0")

		body := []byte(`{
			"newIssues": [
				{"id": "SNYK-JS-001", "severity": "high"}
			],
			"project": {"id": "project-123", "name": "my-web-app"},
			"org": {"id": "org-123", "name": "my-org"}
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"projectId": "project-123",
			},
			Events: eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("X-Snyk-Event-Type header also works", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Snyk-Event-Type", "project_snapshot/v0")

		body := []byte(`{
			"newIssues": [
				{"id": "SNYK-JS-001", "severity": "high"}
			],
			"project": {"id": "project-123", "name": "my-web-app"},
			"org": {"id": "org-123", "name": "my-org"}
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})
}

func Test__OnNewIssueDetected__Setup(t *testing.T) {
	trigger := &OnNewIssueDetected{}

	t.Run("organizationId is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Configuration: map[string]any{"organizationId": ""},
		})

		require.ErrorContains(t, err, "organizationId is required")
	})

	t.Run("webhook is requested with correct config", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		require.NoError(t, trigger.Setup(core.TriggerContext{
			Integration: integrationCtx,
			Configuration: map[string]any{
				"organizationId": "org-123",
				"projectId":      "project-123",
			},
		}))

		require.Len(t, integrationCtx.WebhookRequests, 1)

		webhookRequest := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Equal(t, "issue.detected", webhookRequest.EventType)
		assert.Equal(t, "org-123", webhookRequest.OrgID)
		assert.Equal(t, "project-123", webhookRequest.ProjectID)
	})
}
