package gitlab

import (
	"encoding/json"
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIssueComment__Setup(t *testing.T) {
	trigger := &OnIssueComment{}
	metadata := Metadata{
		Projects: []ProjectMetadata{
			{ID: 123, Name: "group/example", URL: "https://gitlab.com/group/example"},
		},
	}

	t.Run("project is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{Metadata: metadata},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": ""},
		})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("invalid content filter", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{Metadata: metadata},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": "123", "contentFilter": "["},
		})

		require.ErrorContains(t, err, "invalid content filter pattern")
	})

	t.Run("project is not accessible", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{Metadata: metadata},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": "456"},
		})

		require.ErrorContains(t, err, "project 456 is not accessible to integration")
	})

	t.Run("metadata is set and webhook is requested", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{Metadata: metadata}
		metadataCtx := &contexts.MetadataContext{}

		require.NoError(t, trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      metadataCtx,
			Configuration: map[string]any{"project": "123", "contentFilter": "/sp-investigate"},
		}))

		require.Len(t, integrationCtx.WebhookRequests, 1)
		webhookConfig, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, "note", webhookConfig.EventType)
		assert.Equal(t, "123", webhookConfig.ProjectID)
	})
}

func Test__OnIssueComment__HandleWebhook__MissingEventHeader(t *testing.T) {
	trigger := &OnIssueComment{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       http.Header{},
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123"},
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusBadRequest, code)
	assert.ErrorContains(t, err, "X-Gitlab-Event")
}

func Test__OnIssueComment__HandleWebhook__WrongEventType(t *testing.T) {
	trigger := &OnIssueComment{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Issue Hook", "token"),
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnIssueComment__HandleWebhook__InvalidToken(t *testing.T) {
	trigger := &OnIssueComment{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Note Hook", "wrong"),
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123"},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusForbidden, code)
	assert.ErrorContains(t, err, "invalid webhook token")
}

func Test__OnIssueComment__HandleWebhook__IssueComment(t *testing.T) {
	trigger := &OnIssueComment{}

	body, _ := json.Marshal(map[string]any{
		"object_attributes": map[string]any{
			"note":          "/sp-investigate",
			"noteable_type": "Issue",
			"action":        "create",
		},
		"issue": map[string]any{
			"iid":   8,
			"title": "Login page throws 500 on invalid credentials",
		},
	})

	events := &contexts.EventContext{}
	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Note Hook", "token"),
		Body:          body,
		Configuration: map[string]any{"project": "123"},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.issueComment", events.Payloads[0].Type)
}

func Test__OnIssueComment__HandleWebhook__EditedComment(t *testing.T) {
	trigger := &OnIssueComment{}

	body, _ := json.Marshal(map[string]any{
		"object_attributes": map[string]any{
			"note":          "/sp-investigate",
			"noteable_type": "Issue",
			"action":        "update",
		},
	})

	events := &contexts.EventContext{}
	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Note Hook", "token"),
		Body:          body,
		Configuration: map[string]any{"project": "123"},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnIssueComment__HandleWebhook__MissingAction(t *testing.T) {
	trigger := &OnIssueComment{}

	// Older self-managed GitLab instances (pre-16.11) don't send `action`
	// on Note Hook payloads at all. Comments should still fire in that case.
	body, _ := json.Marshal(map[string]any{
		"object_attributes": map[string]any{
			"note":          "/sp-investigate",
			"noteable_type": "Issue",
		},
	})

	events := &contexts.EventContext{}
	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Note Hook", "token"),
		Body:          body,
		Configuration: map[string]any{"project": "123"},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.issueComment", events.Payloads[0].Type)
}

func Test__OnIssueComment__HandleWebhook__NonIssueComment(t *testing.T) {
	trigger := &OnIssueComment{}

	body, _ := json.Marshal(map[string]any{
		"object_attributes": map[string]any{
			"note":          "This looks good to me",
			"noteable_type": "MergeRequest",
			"action":        "create",
		},
	})

	events := &contexts.EventContext{}
	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Note Hook", "token"),
		Body:          body,
		Configuration: map[string]any{"project": "123"},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnIssueComment__HandleWebhook__SystemNote(t *testing.T) {
	trigger := &OnIssueComment{}

	body, _ := json.Marshal(map[string]any{
		"object_attributes": map[string]any{
			"note":          "changed the description",
			"noteable_type": "Issue",
			"system":        true,
			"action":        "create",
		},
	})

	events := &contexts.EventContext{}
	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Note Hook", "token"),
		Body:          body,
		Configuration: map[string]any{"project": "123"},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnIssueComment__HandleWebhook__ContentFilterMatch(t *testing.T) {
	trigger := &OnIssueComment{}

	body, _ := json.Marshal(map[string]any{
		"object_attributes": map[string]any{
			"note":          "/sp-investigate",
			"noteable_type": "Issue",
			"action":        "create",
		},
	})

	events := &contexts.EventContext{}
	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Note Hook", "token"),
		Body:          body,
		Configuration: map[string]any{"project": "123", "contentFilter": "/sp-investigate"},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.issueComment", events.Payloads[0].Type)
}

func Test__OnIssueComment__HandleWebhook__ContentFilterMismatch(t *testing.T) {
	trigger := &OnIssueComment{}

	body, _ := json.Marshal(map[string]any{
		"object_attributes": map[string]any{
			"note":          "Just a regular comment",
			"noteable_type": "Issue",
			"action":        "create",
		},
	})

	events := &contexts.EventContext{}
	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Note Hook", "token"),
		Body:          body,
		Configuration: map[string]any{"project": "123", "contentFilter": "/sp-investigate"},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}
