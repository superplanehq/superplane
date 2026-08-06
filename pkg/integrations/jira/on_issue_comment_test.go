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

func Test__OnIssueComment__Setup(t *testing.T) {
	trigger := &OnIssueComment{}

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

	t.Run("invalid content filter pattern -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": "ENG", "events": []string{"created"}, "contentFilter": "["},
		})
		require.ErrorContains(t, err, "invalid content filter pattern")
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
		stored := metadata.Metadata.(OnIssueCommentMetadata)
		require.NotNil(t, stored.Project)
		assert.Equal(t, "ENG", stored.Project.Key)

		// Setup only resolves the project - it doesn't call Jira's webhook API itself.
		require.Len(t, httpCtx.Requests, 1)
		require.Len(t, integration.WebhookRequests, 1)
		assert.Equal(t, WebhookConfiguration{
			Events: []string{commentEventCreated, commentEventUpdated, commentEventDeleted},
		}, integration.WebhookRequests[0])
	})
}

func Test__OnIssueComment__HandleWebhook(t *testing.T) {
	trigger := &OnIssueComment{}
	meta := func() *contexts.MetadataContext {
		return &contexts.MetadataContext{
			Metadata: OnIssueCommentMetadata{Project: &Project{Key: "ENG"}},
		}
	}

	body := []byte(`{
		"webhookEvent": "comment_created",
		"comment": {
			"id": "10019",
			"body": "Looking into this now.",
			"author": {"accountId": "acct-1", "displayName": "Alice"}
		},
		"issue": {
			"id": "10001",
			"key": "ENG-42",
			"self": "https://example.atlassian.net/rest/api/3/issue/10001",
			"fields": {
				"summary": "Login page returns 500",
				"project": {"key": "ENG"}
			}
		}
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
		assert.Equal(t, IssueCommentEventPayloadType, events.Payloads[0].Type)
		event := events.Payloads[0].Data.(IssueCommentEvent)
		assert.Equal(t, "created", event.Action)
		assert.Equal(t, "ENG-42", event.Issue.Key)
		require.NotNil(t, event.Comment)
		assert.Equal(t, "10019", event.Comment.ID)
		require.NotNil(t, event.Comment.Author)
		assert.Equal(t, "Alice", event.Comment.Author.DisplayName)
	})

	t.Run("ignores events for a different project", func(t *testing.T) {
		events := &contexts.EventContext{}
		metadata := &contexts.MetadataContext{Metadata: OnIssueCommentMetadata{Project: &Project{Key: "OTHER"}}}
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
			Body:          []byte(`{"webhookEvent": "jira:issue_created", "issue": {"key": "ENG-1"}}`),
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

	t.Run("ignores a payload missing a comment", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte(`{"webhookEvent": "comment_created", "issue": {"key": "ENG-1"}}`),
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

	t.Run("ignores an event missing project info rather than fanning it out to every trigger", func(t *testing.T) {
		events := &contexts.EventContext{}
		bodyWithoutProject := []byte(`{
			"webhookEvent": "comment_created",
			"comment": {"id": "10019", "body": "test"},
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

	t.Run("emits when the comment body matches the content filter", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Events:        events,
			Metadata:      meta(),
			Configuration: map[string]any{"events": []string{"created"}, "contentFilter": "Looking into"},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("ignores a comment that does not match the content filter", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Events:        events,
			Metadata:      meta(),
			Configuration: map[string]any{"events": []string{"created"}, "contentFilter": "deploy"},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("matches a content filter against an Atlassian Document Format body", func(t *testing.T) {
		events := &contexts.EventContext{}
		adfBody := []byte(`{
			"webhookEvent": "comment_created",
			"comment": {
				"id": "10019",
				"body": {"type": "doc", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "please deploy this"}]}]}
			},
			"issue": {
				"id": "10001", "key": "ENG-42", "self": "https://example.atlassian.net/rest/api/3/issue/10001",
				"fields": {"project": {"key": "ENG"}}
			}
		}`)
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          adfBody,
			Events:        events,
			Metadata:      meta(),
			Configuration: map[string]any{"events": []string{"created"}, "contentFilter": "deploy"},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("matches a content filter against a mention's attrs.text", func(t *testing.T) {
		events := &contexts.EventContext{}
		adfBody := []byte(`{
			"webhookEvent": "comment_created",
			"comment": {
				"id": "10019",
				"body": {"type": "doc", "content": [{"type": "paragraph", "content": [
					{"type": "mention", "attrs": {"id": "abc", "text": "@Alice Smith"}},
					{"type": "text", "text": " please take a look"}
				]}]}
			},
			"issue": {
				"id": "10001", "key": "ENG-42", "self": "https://example.atlassian.net/rest/api/3/issue/10001",
				"fields": {"project": {"key": "ENG"}}
			}
		}`)
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          adfBody,
			Events:        events,
			Metadata:      meta(),
			Configuration: map[string]any{"events": []string{"created"}, "contentFilter": "Alice"},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("separates text across paragraphs with a space", func(t *testing.T) {
		events := &contexts.EventContext{}
		adfBody := []byte(`{
			"webhookEvent": "comment_created",
			"comment": {
				"id": "10019",
				"body": {"type": "doc", "content": [
					{"type": "paragraph", "content": [{"type": "text", "text": "Hello"}]},
					{"type": "paragraph", "content": [{"type": "text", "text": "World"}]}
				]}
			},
			"issue": {
				"id": "10001", "key": "ENG-42", "self": "https://example.atlassian.net/rest/api/3/issue/10001",
				"fields": {"project": {"key": "ENG"}}
			}
		}`)
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          adfBody,
			Events:        events,
			Metadata:      meta(),
			Configuration: map[string]any{"events": []string{"created"}, "contentFilter": "Hello World"},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("skips empty paragraphs when joining block text", func(t *testing.T) {
		events := &contexts.EventContext{}
		adfBody := []byte(`{
			"webhookEvent": "comment_created",
			"comment": {
				"id": "10019",
				"body": {"type": "doc", "content": [
					{"type": "paragraph", "content": [{"type": "text", "text": "Hello"}]},
					{"type": "paragraph"},
					{"type": "paragraph", "content": [{"type": "text", "text": "World"}]}
				]}
			},
			"issue": {
				"id": "10001", "key": "ENG-42", "self": "https://example.atlassian.net/rest/api/3/issue/10001",
				"fields": {"project": {"key": "ENG"}}
			}
		}`)
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          adfBody,
			Events:        events,
			Metadata:      meta(),
			Configuration: map[string]any{"events": []string{"created"}, "contentFilter": "^Hello World$"},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("a comment with no extractable text never matches a configured filter", func(t *testing.T) {
		events := &contexts.EventContext{}
		bodyWithoutText := []byte(`{
			"webhookEvent": "comment_created",
			"comment": {"id": "10019"},
			"issue": {
				"id": "10001", "key": "ENG-42", "self": "https://example.atlassian.net/rest/api/3/issue/10001",
				"fields": {"project": {"key": "ENG"}}
			}
		}`)
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          bodyWithoutText,
			Events:        events,
			Metadata:      meta(),
			Configuration: map[string]any{"events": []string{"created"}, "contentFilter": "deploy"},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})
}
