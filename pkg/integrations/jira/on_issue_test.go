package jira

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIssue__Setup(t *testing.T) {
	trigger := OnIssue{}

	t.Run("missing events -> error", func(t *testing.T) {
		appCtx := newAuthorizedIntegration()
		err := trigger.Setup(core.TriggerContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"events": []string{},
			},
		})
		require.ErrorContains(t, err, "at least one event")
	})

	t.Run("unknown project -> error", func(t *testing.T) {
		appCtx := newAuthorizedIntegrationWithMetadata(Metadata{
			Projects: []Project{{Key: "OTHER", Name: "Other"}},
		})
		err := trigger.Setup(core.TriggerContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"project": "TEST",
				"events":  []string{JiraEventIssueCreated},
			},
		})
		require.ErrorContains(t, err, "project TEST not found")
	})

	t.Run("valid setup requests a webhook scoped to the project", func(t *testing.T) {
		appCtx := newAuthorizedIntegrationWithMetadata(Metadata{
			Projects: []Project{{Key: "TEST", Name: "Test"}},
		})
		metadataCtx := &contexts.MetadataContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration: appCtx,
			Metadata:    metadataCtx,
			Configuration: map[string]any{
				"project": "TEST",
				"events":  []string{JiraEventIssueCreated},
			},
		})
		require.NoError(t, err)

		md, ok := metadataCtx.Metadata.(OnIssueMetadata)
		require.True(t, ok)
		require.NotNil(t, md.Project)
		assert.Equal(t, "TEST", md.Project.Key)

		require.Len(t, appCtx.WebhookRequests, 1)
		webhookConfig, ok := appCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, "project = TEST", webhookConfig.JQLFilter)
		assert.Equal(t, []string{JiraEventIssueCreated}, webhookConfig.Events)
	})

	t.Run("setup without project still requests a webhook", func(t *testing.T) {
		appCtx := newAuthorizedIntegrationWithMetadata(Metadata{})
		metadataCtx := &contexts.MetadataContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration: appCtx,
			Metadata:    metadataCtx,
			Configuration: map[string]any{
				"events": []string{JiraEventIssueCreated, JiraEventIssueUpdated},
				"jql":    "labels in (bug)",
			},
		})
		require.NoError(t, err)

		md := metadataCtx.Metadata.(OnIssueMetadata)
		assert.Nil(t, md.Project)

		require.Len(t, appCtx.WebhookRequests, 1)
		webhookConfig := appCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Equal(t, "labels in (bug)", webhookConfig.JQLFilter)
	})
}

func Test__OnIssue__HandleWebhook(t *testing.T) {
	trigger := OnIssue{}

	body := []byte(`{
		"webhookEvent": "jira:issue_created",
		"issue": {
			"key": "TEST-1",
			"fields": {
				"project": {"key": "TEST"},
				"summary": "Hi"
			}
		}
	}`)

	t.Run("emits event when configured event matches", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Configuration: map[string]any{
				"events": []string{JiraEventIssueCreated},
			},
			Body:    body,
			Headers: http.Header{},
			Events:  events,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 1, events.Count())
		assert.Equal(t, OnIssuePayloadType, events.Payloads[0].Type)
	})

	t.Run("ignores unrelated webhook event", func(t *testing.T) {
		events := &contexts.EventContext{}
		other := []byte(`{"webhookEvent":"jira:issue_deleted","issue":{"fields":{"project":{"key":"TEST"}}}}`)
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Configuration: map[string]any{
				"events": []string{JiraEventIssueCreated},
			},
			Body:    other,
			Headers: http.Header{},
			Events:  events,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("filters by project", func(t *testing.T) {
		events := &contexts.EventContext{}
		other := []byte(`{"webhookEvent":"jira:issue_created","issue":{"fields":{"project":{"key":"OTHER"}}}}`)
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Configuration: map[string]any{
				"project": "TEST",
				"events":  []string{JiraEventIssueCreated},
			},
			Body:    other,
			Headers: http.Header{},
			Events:  events,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("malformed JSON -> 400", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Configuration: map[string]any{
				"events": []string{JiraEventIssueCreated},
			},
			Body:    []byte("not json"),
			Headers: http.Header{},
			Events:  events,
		})
		require.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, code)
	})
}

func Test__OnIssue__ScopeJQL(t *testing.T) {
	assert.Equal(t, "project = TEST", scopeJQLToProject("", "TEST"))
	assert.Equal(t, "project = TEST AND priority = High", scopeJQLToProject("project = TEST AND priority = High", "TEST"))
	assert.Equal(t, "project = TEST AND (labels in (bug))", scopeJQLToProject("labels in (bug)", "TEST"))
}

func Test__OnIssue__TriggerInfo(t *testing.T) {
	trigger := OnIssue{}
	assert.Equal(t, "jira.onIssue", trigger.Name())
	assert.Equal(t, "On Issue", trigger.Label())
	assert.NotEmpty(t, trigger.Documentation())
	assert.NotNil(t, trigger.ExampleData())
}
