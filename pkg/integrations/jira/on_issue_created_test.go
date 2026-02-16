package jira

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIssueCreated__HandleWebhook(t *testing.T) {
	trigger := &OnIssueCreated{}

	t.Run("missing signature -> 200 OK, event emitted (OAuth webhook)", func(t *testing.T) {
		headers := http.Header{}
		body := []byte(`{"webhookEvent":"jira:issue_created"}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"project": "TEST",
			},
			Webhook: &contexts.WebhookContext{Secret: "test-secret"},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Hub-Signature", "sha256=invalid")
		body := []byte(`{"webhookEvent":"jira:issue_created"}`)

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"project": "TEST",
			},
			Webhook: &contexts.WebhookContext{Secret: "test-secret"},
			Events:  &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("wrong event type -> 200 OK, no event emitted", func(t *testing.T) {
		body := []byte(`{"webhookEvent":"jira:issue_updated"}`)
		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature", "sha256="+signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"project": "TEST",
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("issue type not in filter -> 200 OK, no event emitted", func(t *testing.T) {
		body := []byte(`{
			"webhookEvent": "jira:issue_created",
			"issue": {
				"fields": {
					"issuetype": {"name": "Epic"}
				}
			}
		}`)
		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature", "sha256="+signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"project":    "TEST",
				"issueTypes": []string{"Task", "Bug"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("valid webhook, issue type matches -> event emitted", func(t *testing.T) {
		body := []byte(`{
			"webhookEvent": "jira:issue_created",
			"issue": {
				"id": "10001",
				"key": "TEST-123",
				"fields": {
					"issuetype": {"name": "Task"},
					"summary": "Test issue"
				}
			}
		}`)
		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature", "sha256="+signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"project":    "TEST",
				"issueTypes": []string{"Task", "Bug"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("valid webhook, no issue type filter -> event emitted for all types", func(t *testing.T) {
		body := []byte(`{
			"webhookEvent": "jira:issue_created",
			"issue": {
				"id": "10001",
				"key": "TEST-123",
				"fields": {
					"issuetype": {"name": "Epic"},
					"summary": "Test epic"
				}
			}
		}`)
		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature", "sha256="+signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"project": "TEST",
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})
}

func Test__OnIssueCreated__Setup(t *testing.T) {
	testProject := Project{ID: "10000", Key: "TEST", Name: "Test Project"}
	trigger := OnIssueCreated{}

	t.Run("api token auth -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"authType": AuthTypeAPIToken},
		}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": "TEST"},
		})

		require.ErrorContains(t, err, "webhook triggers require OAuth")
	})

	t.Run("missing project -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"authType": AuthTypeOAuth},
		}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": ""},
		})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("project not found in metadata -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"authType": AuthTypeOAuth},
			Metadata: Metadata{
				Projects: []Project{testProject},
			},
		}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": "OTHER"},
		})

		require.ErrorContains(t, err, "project OTHER is not accessible")
	})

	t.Run("valid setup -> metadata set, webhook requested", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"authType": AuthTypeOAuth},
			Metadata: Metadata{
				Projects: []Project{testProject},
			},
		}

		nodeMetadataCtx := contexts.MetadataContext{}
		require.NoError(t, trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &nodeMetadataCtx,
			Configuration: map[string]any{"project": "TEST"},
		}))

		require.Equal(t, nodeMetadataCtx.Get(), NodeMetadata{Project: &testProject})
		require.Len(t, integrationCtx.WebhookRequests, 1)

		webhookRequest := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Equal(t, "jira:issue_created", webhookRequest.EventType)
		assert.Equal(t, "TEST", webhookRequest.Project)
	})
}

func Test__OnIssueCreated__TriggerInfo(t *testing.T) {
	trigger := OnIssueCreated{}

	assert.Equal(t, "jira.onIssueCreated", trigger.Name())
	assert.Equal(t, "On Issue Created", trigger.Label())
	assert.Equal(t, "Listen for new issues created in Jira", trigger.Description())
	assert.Equal(t, "jira", trigger.Icon())
	assert.Equal(t, "blue", trigger.Color())
	assert.NotEmpty(t, trigger.Documentation())
}

func Test__OnIssueCreated__Configuration(t *testing.T) {
	trigger := OnIssueCreated{}

	config := trigger.Configuration()
	assert.Len(t, config, 2)

	fieldNames := make([]string, len(config))
	for i, f := range config {
		fieldNames[i] = f.Name
	}

	assert.Contains(t, fieldNames, "project")
	assert.Contains(t, fieldNames, "issueTypes")

	for _, f := range config {
		if f.Name == "project" {
			assert.True(t, f.Required, "project should be required")
		} else if f.Name == "issueTypes" {
			assert.False(t, f.Required, "issueTypes should be optional")
		}
	}
}
