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

	t.Run("valid config resolves the project and registers a Jira webhook", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"id":"10000","key":"ENG","name":"Engineering"}]`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"createdWebhookId":1000}]`))},
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
		require.NotNil(t, stored.WebhookID)
		assert.Equal(t, int64(1000), *stored.WebhookID)

		require.Len(t, httpCtx.Requests, 2)
		createReq := httpCtx.Requests[1]
		assert.Equal(t, http.MethodPost, createReq.Method)
		assert.Contains(t, createReq.URL.String(), "/rest/api/3/webhook")
		reqBody, _ := io.ReadAll(createReq.Body)
		assert.Contains(t, string(reqBody), `"jira:issue_created"`)
		assert.Contains(t, string(reqBody), `"jira:issue_updated"`)
		assert.Contains(t, string(reqBody), `project = \"ENG\"`)
	})

	t.Run("tears down a previous webhook before registering a new one", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"id":"10000","key":"ENG","name":"Engineering"}]`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"createdWebhookId":1001}]`))},
			},
		}
		existingID := int64(1000)
		metadata := &contexts.MetadataContext{Metadata: OnIssueMetadata{WebhookID: &existingID}}
		err := trigger.Setup(core.TriggerContext{
			HTTP:          httpCtx,
			Integration:   newAuthorizedIntegration(),
			Metadata:      metadata,
			Webhook:       &contexts.NodeWebhookContext{},
			Configuration: map[string]any{"project": "ENG", "events": []string{"created"}},
		})
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 3)
		deleteReq := httpCtx.Requests[1]
		assert.Equal(t, http.MethodDelete, deleteReq.Method)
		deleteBody, _ := io.ReadAll(deleteReq.Body)
		assert.Contains(t, string(deleteBody), "1000")

		stored := metadata.Metadata.(OnIssueMetadata)
		require.NotNil(t, stored.WebhookID)
		assert.Equal(t, int64(1001), *stored.WebhookID)
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

}

func Test__OnIssue__Cleanup(t *testing.T) {
	trigger := &OnIssue{}

	t.Run("deletes the registered webhook", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
			},
		}
		webhookID := int64(1000)
		metadata := &contexts.MetadataContext{Metadata: OnIssueMetadata{WebhookID: &webhookID}}

		err := trigger.Cleanup(core.TriggerContext{
			HTTP:        httpCtx,
			Integration: newAuthorizedIntegration(),
			Metadata:    metadata,
		})
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodDelete, httpCtx.Requests[0].Method)
		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		assert.Contains(t, string(body), "1000")
	})

	t.Run("no-op when no webhook was registered", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}
		metadata := &contexts.MetadataContext{Metadata: OnIssueMetadata{}}

		err := trigger.Cleanup(core.TriggerContext{
			HTTP:        httpCtx,
			Integration: newAuthorizedIntegration(),
			Metadata:    metadata,
		})
		require.NoError(t, err)
		assert.Empty(t, httpCtx.Requests)
	})
}
