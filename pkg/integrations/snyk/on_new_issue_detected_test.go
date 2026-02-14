package snyk

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
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
	assert.Len(t, configFields, 2)

	fieldNames := make(map[string]bool)
	for _, field := range configFields {
		fieldNames[field.Name] = true
	}

	expectedFields := []string{"projectId", "severity"}
	for _, fieldName := range expectedFields {
		assert.True(t, fieldNames[fieldName], "Missing field: %s", fieldName)
	}
}

func Test__OnNewIssueDetected__HandleWebhook(t *testing.T) {
	trigger := &OnNewIssueDetected{}
	secret := "test-webhook-secret"

	signatureFor := func(secret string, body []byte) string {
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		return fmt.Sprintf("sha256=%x", h.Sum(nil))
	}

	t.Run("missing signature -> 403", func(t *testing.T) {
		headers := http.Header{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "missing signature")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		body := []byte(`{}`)
		headers := http.Header{}
		headers.Set("X-Hub-Signature", "sha256=invalidsignature")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: secret},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("missing event header -> 400", func(t *testing.T) {
		body := []byte(`{}`)
		headers := http.Header{}
		headers.Set("X-Hub-Signature", signatureFor(secret, body))

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: secret},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "missing Snyk event header")
	})

	t.Run("non-matching event type -> 200, no events emitted", func(t *testing.T) {
		body := []byte(`{}`)
		headers := http.Header{}
		headers.Set("X-Snyk-Event", "ping")
		headers.Set("X-Hub-Signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Events:        eventContext,
			Webhook:       &contexts.WebhookContext{Secret: secret},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("invalid JSON body -> 400", func(t *testing.T) {
		body := []byte(`not json`)
		headers := http.Header{}
		headers.Set("X-Snyk-Event", "project_snapshot/v0")
		headers.Set("X-Hub-Signature", signatureFor(secret, body))

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Events:        &contexts.EventContext{},
			Webhook:       &contexts.WebhookContext{Secret: secret},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "error parsing request body")
	})

	t.Run("no new issues -> 200, no events emitted", func(t *testing.T) {
		body := []byte(`{"newIssues": [], "project": {"id": "p1"}}`)
		headers := http.Header{}
		headers.Set("X-Snyk-Event", "project_snapshot/v0")
		headers.Set("X-Hub-Signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Events:        eventContext,
			Webhook:       &contexts.WebhookContext{Secret: secret},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("new issue detected -> event emitted", func(t *testing.T) {
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

		headers := http.Header{}
		headers.Set("X-Snyk-Event", "project_snapshot/v0")
		headers.Set("X-Hub-Signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Events:        eventContext,
			Webhook:       &contexts.WebhookContext{Secret: secret},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "snyk.issue.detected", eventContext.Payloads[0].Type)
	})

	t.Run("multiple new issues -> multiple events emitted", func(t *testing.T) {
		body := []byte(`{
			"newIssues": [
				{"id": "SNYK-JS-001", "severity": "high"},
				{"id": "SNYK-JS-002", "severity": "critical"}
			],
			"project": {"id": "project-123", "name": "my-web-app"},
			"org": {"id": "org-123", "name": "my-org"}
		}`)

		headers := http.Header{}
		headers.Set("X-Snyk-Event", "project_snapshot/v0")
		headers.Set("X-Hub-Signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Events:        eventContext,
			Webhook:       &contexts.WebhookContext{Secret: secret},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 2, eventContext.Count())
	})

	t.Run("severity filter -> only exact matching issues emitted", func(t *testing.T) {
		body := []byte(`{
			"newIssues": [
				{"id": "SNYK-JS-001", "severity": "low"},
				{"id": "SNYK-JS-002", "severity": "high"},
				{"id": "SNYK-JS-003", "severity": "critical"}
			],
			"project": {"id": "project-123", "name": "my-web-app"},
			"org": {"id": "org-123", "name": "my-org"}
		}`)

		headers := http.Header{}
		headers.Set("X-Snyk-Event", "project_snapshot/v0")
		headers.Set("X-Hub-Signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"severity": []string{"high", "critical"},
			},
			Events:  eventContext,
			Webhook: &contexts.WebhookContext{Secret: secret},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 2, eventContext.Count())
	})

	t.Run("severity filter -> missing severity in issue rejects", func(t *testing.T) {
		body := []byte(`{
			"newIssues": [
				{"id": "SNYK-JS-001"}
			],
			"project": {"id": "project-123", "name": "my-web-app"},
			"org": {"id": "org-123", "name": "my-org"}
		}`)

		headers := http.Header{}
		headers.Set("X-Snyk-Event", "project_snapshot/v0")
		headers.Set("X-Hub-Signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"severity": []string{"high"},
			},
			Events:  eventContext,
			Webhook: &contexts.WebhookContext{Secret: secret},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("project filter -> missing project data rejects", func(t *testing.T) {
		body := []byte(`{
			"newIssues": [
				{"id": "SNYK-JS-001", "severity": "high"}
			],
			"org": {"id": "org-123", "name": "my-org"}
		}`)

		headers := http.Header{}
		headers.Set("X-Snyk-Event", "project_snapshot/v0")
		headers.Set("X-Hub-Signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"projectId": "project-123",
			},
			Events:  eventContext,
			Webhook: &contexts.WebhookContext{Secret: secret},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("project filter -> only matching project emitted", func(t *testing.T) {
		body := []byte(`{
			"newIssues": [
				{"id": "SNYK-JS-001", "severity": "high"}
			],
			"project": {"id": "project-123", "name": "my-web-app"},
			"org": {"id": "org-123", "name": "my-org"}
		}`)

		headers := http.Header{}
		headers.Set("X-Snyk-Event", "project_snapshot/v0")
		headers.Set("X-Hub-Signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"projectId": "project-999",
			},
			Events:  eventContext,
			Webhook: &contexts.WebhookContext{Secret: secret},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("project filter matches -> event emitted", func(t *testing.T) {
		body := []byte(`{
			"newIssues": [
				{"id": "SNYK-JS-001", "severity": "high"}
			],
			"project": {"id": "project-123", "name": "my-web-app"},
			"org": {"id": "org-123", "name": "my-org"}
		}`)

		headers := http.Header{}
		headers.Set("X-Snyk-Event", "project_snapshot/v0")
		headers.Set("X-Hub-Signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"projectId": "project-123",
			},
			Events:  eventContext,
			Webhook: &contexts.WebhookContext{Secret: secret},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("X-Snyk-Event-Type header also works", func(t *testing.T) {
		body := []byte(`{
			"newIssues": [
				{"id": "SNYK-JS-001", "severity": "high"}
			],
			"project": {"id": "project-123", "name": "my-web-app"},
			"org": {"id": "org-123", "name": "my-org"}
		}`)

		headers := http.Header{}
		headers.Set("X-Snyk-Event-Type", "project_snapshot/v0")
		headers.Set("X-Hub-Signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Events:        eventContext,
			Webhook:       &contexts.WebhookContext{Secret: secret},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})
}

func Test__OnNewIssueDetected__Setup(t *testing.T) {
	trigger := &OnNewIssueDetected{}

	t.Run("organizationId is required in integration config", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"organizationId": ""},
		}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "organizationId is required")
	})

	t.Run("webhook is requested with correct config", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"organizationId": "org-123"},
		}
		require.NoError(t, trigger.Setup(core.TriggerContext{
			Integration: integrationCtx,
			Configuration: map[string]any{
				"projectId": "project-123",
			},
		}))

		require.Len(t, integrationCtx.WebhookRequests, 1)

		webhookRequest := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Equal(t, "issue.detected", webhookRequest.EventType)
		assert.Equal(t, "org-123", webhookRequest.OrgID)
		assert.Equal(t, "project-123", webhookRequest.ProjectID)
	})
}
