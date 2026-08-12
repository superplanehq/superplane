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

	t.Run("valid config resolves the project and requests the shared Jira webhook", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"id":"10000","key":"ENG","name":"Engineering"}]`))},
			},
		}
		metadata := &contexts.MetadataContext{}
		integration := newAuthorizedIntegration()
		err := trigger.Setup(core.TriggerContext{
			HTTP:          httpCtx,
			Integration:   integration,
			Metadata:      metadata,
			Configuration: map[string]any{"project": "ENG", "events": []string{"created", "updated"}},
		})
		require.NoError(t, err)
		stored := metadata.Metadata.(OnIssueMetadata)
		require.NotNil(t, stored.Project)
		assert.Equal(t, "ENG", stored.Project.Key)

		// Setup only resolves the project - it doesn't call Jira's webhook API itself.
		require.Len(t, httpCtx.Requests, 1)
		require.Len(t, integration.WebhookRequests, 1)
		assert.Equal(t, WebhookConfiguration{
			Events: []string{issueEventCreated, issueEventUpdated, issueEventDeleted},
		}, integration.WebhookRequests[0])
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

	// Regression test: the webhook is shared by every jira.onIssue trigger on the integration, so
	// a payload that doesn't carry a project key (e.g. a stripped-down delete payload) must not
	// fail open and fire for a trigger configured for a different project.
	t.Run("ignores an event missing project info rather than fanning it out to every trigger", func(t *testing.T) {
		events := &contexts.EventContext{}
		bodyWithoutProject := []byte(`{
			"webhookEvent": "jira:issue_created",
			"issue": {"id": "10001", "key": "ENG-42", "self": "https://example.atlassian.net/rest/api/3/issue/10001", "fields": {}}
		}`)
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          bodyWithoutProject,
			Events:        events,
			Metadata:      meta(),
			Configuration: map[string]any{"events": []string{"created"}},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})
}
