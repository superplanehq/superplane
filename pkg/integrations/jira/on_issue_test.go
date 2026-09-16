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

	createdWithoutFields := []byte(`{
		"webhookEvent": "jira:issue_created",
		"issue": {"id": "10001", "key": "ENG-42", "self": "https://example.atlassian.net/rest/api/3/issue/10001", "fields": {}}
	}`)

	t.Run("emits a created event when the issue key names the project and fields are empty", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          createdWithoutFields,
			Events:        events,
			Metadata:      meta(),
			Configuration: map[string]any{"events": []string{"created"}},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		event := events.Payloads[0].Data.(IssueEvent)
		assert.Equal(t, "created", event.Action)
		assert.Equal(t, "ENG-42", event.Issue.Key)
	})

	t.Run("ignores an empty-fields event whose issue key names a different project", func(t *testing.T) {
		events := &contexts.EventContext{}
		metadata := &contexts.MetadataContext{Metadata: OnIssueMetadata{Project: &Project{Key: "OTHER"}}}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          createdWithoutFields,
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

	// The webhook is shared by every jira.onIssue trigger on the integration, so a payload
	// with neither a project field nor an issue key must not fail open.
	t.Run("ignores an event with no issue key and no project field", func(t *testing.T) {
		events := &contexts.EventContext{}
		bodyWithoutIdentity := []byte(`{
			"webhookEvent": "jira:issue_created",
			"issue": {"id": "10001", "self": "https://example.atlassian.net/rest/api/3/issue/10001", "fields": {}}
		}`)
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          bodyWithoutIdentity,
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

	t.Run("loads the full issue when the webhook omits fields", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"id": "10001",
						"key": "ENG-42",
						"fields": {
							"summary": "Login page returns 500",
							"description": {"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Users cannot sign in."}]}]},
							"project": {"key": "ENG"}
						}
					}`)),
				},
			},
		}
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          createdWithoutFields,
			Events:        events,
			Metadata:      meta(),
			Configuration: map[string]any{"events": []string{"created"}},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
			HTTP:          httpCtx,
			Integration:   newAuthorizedIntegration(),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		event := events.Payloads[0].Data.(IssueEvent)
		assert.Equal(t, "ENG-42", event.Issue.Key)
		assert.Equal(t, "Login page returns 500", event.Issue.Fields["summary"])
		assert.Equal(t, "Users cannot sign in.", event.Description)
		require.Len(t, httpCtx.Requests, 1)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/rest/api/3/issue/ENG-42")
	})
}

func TestProjectKeyFromIssueKey(t *testing.T) {
	assert.Equal(t, "ENG", ProjectKeyFromIssueKey("ENG-42"))
	assert.Equal(t, "ENG-SUB", ProjectKeyFromIssueKey("ENG-SUB-42"))
	assert.Equal(t, "", ProjectKeyFromIssueKey("ENG42"))
	assert.Equal(t, "", ProjectKeyFromIssueKey("-42"))
	assert.Equal(t, "", ProjectKeyFromIssueKey(""))
}
