package jira

import (
	"io"
	"net/http"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIssue__Setup(t *testing.T) {
	trigger := &OnIssue{}

	t.Run("project is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": "", "events": []string{"created"}},
		})
		require.ErrorContains(t, err, "project is required")
	})

	t.Run("at least one event is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": "ENG", "events": []string{}},
		})
		require.ErrorContains(t, err, "at least one event")
	})

	t.Run("valid config resolves the project and sets up a webhook", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"id":"10000","key":"ENG","name":"Engineering"}]`))},
			},
		}
		metadata := &contexts.MetadataContext{}
		err := trigger.Setup(core.TriggerContext{
			HTTP:          httpCtx,
			Integration:   newAuthorizedIntegration(),
			Metadata:      metadata,
			Webhook:       &contexts.NodeWebhookContext{},
			Configuration: map[string]any{"project": "ENG", "events": []string{"created", "updated"}},
		})
		require.NoError(t, err)
		stored := metadata.Metadata.(OnIssueMetadata)
		require.NotNil(t, stored.Project)
		assert.Equal(t, "ENG", stored.Project.Key)
		assert.NotEmpty(t, stored.WebhookURL)
	})
}

func Test__OnIssue__HandleWebhook(t *testing.T) {
	trigger := &OnIssue{}
	meta := func() *contexts.MetadataContext {
		return &contexts.MetadataContext{
			Metadata: OnIssueMetadata{Project: &Project{Key: "ENG"}},
		}
	}

	body := []byte(`{
		"webhookEvent": "jira:issue_created",
		"issue": {
			"id": "10001",
			"key": "ENG-42",
			"self": "https://example.atlassian.net/rest/api/3/issue/10001",
			"fields": {
				"summary": "Login page returns 500",
				"project": {"key": "ENG"}
			}
		},
		"user": {"accountId": "acct-1", "displayName": "Alice"}
	}`)

	t.Run("emits a created event for a configured project and action", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Events:        events,
			Metadata:      meta(),
			Configuration: map[string]any{"events": []string{"created"}},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, IssueEventPayloadType, events.Payloads[0].Type)
		event := events.Payloads[0].Data.(IssueEvent)
		assert.Equal(t, "created", event.Action)
		assert.Equal(t, "ENG-42", event.Issue.Key)
		require.NotNil(t, event.User)
		assert.Equal(t, "Alice", event.User.DisplayName)
	})

	t.Run("ignores events for a different project", func(t *testing.T) {
		events := &contexts.EventContext{}
		metadata := &contexts.MetadataContext{Metadata: OnIssueMetadata{Project: &Project{Key: "OTHER"}}}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Events:        events,
			Metadata:      metadata,
			Configuration: map[string]any{"events": []string{"created"}},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("ignores events not in configured actions", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Events:        events,
			Metadata:      meta(),
			Configuration: map[string]any{"events": []string{"updated"}},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("ignores unsupported webhookEvent values", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte(`{"webhookEvent": "comment_created", "issue": {"key": "ENG-1"}}`),
			Events:        events,
			Metadata:      meta(),
			Configuration: map[string]any{"events": []string{"created", "updated", "deleted"}},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("rejects requests missing a valid shared secret when one is configured", func(t *testing.T) {
		events := &contexts.EventContext{}
		integration := &contexts.IntegrationContext{Configuration: map[string]any{"webhookSharedSecret": "s3cr3t"}}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Events:        events,
			Metadata:      meta(),
			Integration:   integration,
			Configuration: map[string]any{"events": []string{"created"}},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
		})
		require.ErrorContains(t, err, "invalid or missing webhook authorization")
		assert.Equal(t, http.StatusForbidden, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("accepts requests with a valid shared secret", func(t *testing.T) {
		events := &contexts.EventContext{}
		integration := &contexts.IntegrationContext{Configuration: map[string]any{"webhookSharedSecret": "s3cr3t"}}
		headers := http.Header{}
		headers.Set("Authorization", "Bearer s3cr3t")
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Events:        events,
			Metadata:      meta(),
			Integration:   integration,
			Configuration: map[string]any{"events": []string{"created"}},
			Headers:       headers,
			Logger:        log.NewEntry(log.New()),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 1, events.Count())
	})
}
