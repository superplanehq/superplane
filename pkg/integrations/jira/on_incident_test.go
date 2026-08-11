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

func Test__OnIncident__Setup(t *testing.T) {
	trigger := &OnIncident{}

	t.Run("service desk is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"serviceDesk": "", "requestTypes": []string{"5"}, "events": []string{"created"}},
		})
		require.ErrorContains(t, err, "service desk is required")
	})

	t.Run("at least one request type must be selected", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"serviceDesk": "2", "requestTypes": []string{}, "events": []string{"created"}},
		})
		require.ErrorContains(t, err, "at least one request type")
	})

	t.Run("at least one event is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"serviceDesk": "2", "requestTypes": []string{"5"}, "events": []string{}},
		})
		require.ErrorContains(t, err, "at least one event")
	})

	t.Run("valid config resolves the service desk, request types, and requests the shared webhook", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`{"values":[{"id":"2","projectName":"Demo service space","projectKey":"DEMO"}],"isLastPage":true}`,
				))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`{"id":"5","name":"Computer support","issueTypeId":"10075"}`,
				))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`{"id":"7","name":"Employee exit","issueTypeId":"10076"}`,
				))},
			},
		}
		metadata := &contexts.MetadataContext{}
		integration := newAuthorizedIntegration()
		err := trigger.Setup(core.TriggerContext{
			HTTP:          httpCtx,
			Integration:   integration,
			Metadata:      metadata,
			Configuration: map[string]any{"serviceDesk": "2", "requestTypes": []string{"5", "7"}, "events": []string{"created"}},
		})
		require.NoError(t, err)

		stored := metadata.Metadata.(OnIncidentMetadata)
		require.NotNil(t, stored.ServiceDesk)
		assert.Equal(t, "DEMO", stored.ServiceDesk.ProjectKey)
		assert.ElementsMatch(t, []string{"10075", "10076"}, stored.IssueTypeIDs)

		// Setup only resolves the service desk and request types - it doesn't call Jira's webhook API itself.
		require.Len(t, httpCtx.Requests, 3)
		require.Len(t, integration.WebhookRequests, 1)
		assert.Equal(t, WebhookConfiguration{
			Events: []string{issueEventCreated, issueEventUpdated, issueEventDeleted},
		}, integration.WebhookRequests[0])
	})

	t.Run("unknown service desk -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"values":[],"isLastPage":true}`))},
			},
		}
		err := trigger.Setup(core.TriggerContext{
			HTTP:          httpCtx,
			Integration:   newAuthorizedIntegration(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"serviceDesk": "99", "requestTypes": []string{"5"}, "events": []string{"created"}},
		})
		require.ErrorContains(t, err, "service desk 99 not found")
	})

	// Regression test: a request type with no underlying issue type would silently never match
	// anything in HandleWebhook, leaving the trigger looking configured but permanently inert -
	// Setup must fail loudly instead (BUGBOT_BUG_ID ed7585e9).
	t.Run("request type without an issue type -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`{"values":[{"id":"2","projectName":"Demo service space","projectKey":"DEMO"}],"isLastPage":true}`,
				))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`{"id":"5","name":"Computer support","issueTypeId":""}`,
				))},
			},
		}
		err := trigger.Setup(core.TriggerContext{
			HTTP:          httpCtx,
			Integration:   newAuthorizedIntegration(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"serviceDesk": "2", "requestTypes": []string{"5"}, "events": []string{"created"}},
		})
		require.ErrorContains(t, err, "no underlying issue type")
	})
}

func Test__OnIncident__HandleWebhook(t *testing.T) {
	trigger := &OnIncident{}
	meta := func() *contexts.MetadataContext {
		return &contexts.MetadataContext{
			Metadata: OnIncidentMetadata{
				ServiceDesk:  &ServiceDesk{ID: "2", ProjectKey: "DEMO"},
				IssueTypeIDs: []string{"10075"},
			},
		}
	}

	body := func(issueTypeID string) []byte {
		return []byte(`{
			"webhookEvent": "jira:issue_created",
			"issue": {
				"id": "10001",
				"key": "DEMO-42",
				"self": "https://example.atlassian.net/rest/api/3/issue/10001",
				"fields": {
					"summary": "Payments API returning 500s",
					"issuetype": {"id": "` + issueTypeID + `"},
					"project": {"key": "DEMO"}
				}
			},
			"user": {"accountId": "acct-1", "displayName": "Alice"}
		}`)
	}

	t.Run("emits a created event for a matching request type's issue type", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body("10075"),
			Events:        events,
			Metadata:      meta(),
			Configuration: map[string]any{"events": []string{"created"}},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, IncidentEventPayloadType, events.Payloads[0].Type)
		event := events.Payloads[0].Data.(IssueEvent)
		assert.Equal(t, "created", event.Action)
		assert.Equal(t, "DEMO-42", event.Issue.Key)
	})

	// Regression test: issue type ids are global to the site, not scoped to a project, so an
	// issue in a different project reusing the same issue type must not match (BUGBOT_BUG_ID
	// 4eb011cb).
	t.Run("ignores a matching issue type in a different project's service desk", func(t *testing.T) {
		events := &contexts.EventContext{}
		otherProjectBody := []byte(`{
			"webhookEvent": "jira:issue_created",
			"issue": {
				"id": "10001",
				"key": "OTHER-42",
				"self": "https://example.atlassian.net/rest/api/3/issue/10001",
				"fields": {
					"summary": "Unrelated issue in another project",
					"issuetype": {"id": "10075"},
					"project": {"key": "OTHER"}
				}
			}
		}`)
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          otherProjectBody,
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

	t.Run("ignores an issue whose type is not one of the configured request types", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body("10076"),
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

	t.Run("ignores events not in configured actions", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body("10075"),
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
			Body:          []byte(`{"webhookEvent": "comment_created", "issue": {"key": "DEMO-1"}}`),
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

	// Regression test: the webhook is shared by every trigger on the integration, so a payload
	// that doesn't carry an issue type must not fail open and fire regardless of type.
	t.Run("ignores an event missing issue type info rather than fanning it out to every trigger", func(t *testing.T) {
		events := &contexts.EventContext{}
		bodyWithoutIssueType := []byte(`{
			"webhookEvent": "jira:issue_created",
			"issue": {"id": "10001", "key": "DEMO-42", "self": "https://example.atlassian.net/rest/api/3/issue/10001", "fields": {"project": {"key": "DEMO"}}}
		}`)
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          bodyWithoutIssueType,
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
