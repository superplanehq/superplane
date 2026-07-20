package gitlab

import (
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const branchCreationBody = `{"ref":"refs/heads/feature/x","before":"0000000000000000000000000000000000000000","after":"bbb"}`

func Test__OnBranchCreated__HandleWebhook__MissingEventHeader(t *testing.T) {
	trigger := &OnBranchCreated{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       http.Header{},
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123"},
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusBadRequest, code)
	assert.ErrorContains(t, err, "X-Gitlab-Event")
}

func Test__OnBranchCreated__HandleWebhook__WrongEventType(t *testing.T) {
	trigger := &OnBranchCreated{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Tag Push Hook", "token"),
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnBranchCreated__HandleWebhook__MissingToken(t *testing.T) {
	trigger := &OnBranchCreated{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Push Hook", ""),
		Body:          []byte(branchCreationBody),
		Configuration: map[string]any{"project": "123"},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusForbidden, code)
	assert.ErrorContains(t, err, "missing X-Gitlab-Token header")
	assert.Zero(t, events.Count())
}

func Test__OnBranchCreated__HandleWebhook__InvalidToken(t *testing.T) {
	trigger := &OnBranchCreated{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Push Hook", "wrong"),
		Body:          []byte(branchCreationBody),
		Configuration: map[string]any{"project": "123"},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusForbidden, code)
	assert.ErrorContains(t, err, "invalid webhook token")
	assert.Zero(t, events.Count())
}

func Test__OnBranchCreated__HandleWebhook__NonCreationPushIgnored(t *testing.T) {
	trigger := &OnBranchCreated{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers: gitlabHeaders("Push Hook", "token"),
		Body:    []byte(`{"ref":"refs/heads/main","before":"aaa","after":"bbb"}`),
		Configuration: map[string]any{
			"project": "123",
			"branches": []configuration.Predicate{
				{Type: configuration.PredicateTypeMatches, Value: ".*"},
			},
		},
		Webhook: &contexts.NodeWebhookContext{Secret: "token"},
		Events:  events,
		Logger:  log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnBranchCreated__HandleWebhook__FullRefMatch(t *testing.T) {
	trigger := &OnBranchCreated{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers: gitlabHeaders("Push Hook", "token"),
		Body:    []byte(branchCreationBody),
		Configuration: map[string]any{
			"project": "123",
			"branches": []configuration.Predicate{
				{Type: configuration.PredicateTypeEquals, Value: "refs/heads/feature/x"},
			},
		},
		Webhook: &contexts.NodeWebhookContext{Secret: "token"},
		Events:  events,
		Logger:  log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.branchCreated", events.Payloads[0].Type)
}

func Test__OnBranchCreated__HandleWebhook__BranchNameMatch(t *testing.T) {
	trigger := &OnBranchCreated{}
	events := &contexts.EventContext{}

	// Equals against the short name only matches via the refs/heads/ trim path,
	// since the full ref ("refs/heads/feature/x") is not equal to "feature/x".
	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers: gitlabHeaders("Push Hook", "token"),
		Body:    []byte(branchCreationBody),
		Configuration: map[string]any{
			"project": "123",
			"branches": []configuration.Predicate{
				{Type: configuration.PredicateTypeEquals, Value: "feature/x"},
			},
		},
		Webhook: &contexts.NodeWebhookContext{Secret: "token"},
		Events:  events,
		Logger:  log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.branchCreated", events.Payloads[0].Type)
}

func Test__OnBranchCreated__HandleWebhook__BranchMismatch(t *testing.T) {
	trigger := &OnBranchCreated{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers: gitlabHeaders("Push Hook", "token"),
		Body:    []byte(branchCreationBody),
		Configuration: map[string]any{
			"project": "123",
			"branches": []configuration.Predicate{
				{Type: configuration.PredicateTypeEquals, Value: "main"},
			},
		},
		Webhook: &contexts.NodeWebhookContext{Secret: "token"},
		Events:  events,
		Logger:  log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnBranchCreated__Setup(t *testing.T) {
	trigger := &OnBranchCreated{}
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

		require.NoError(t, trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": "123"},
		}))

		require.Len(t, integrationCtx.WebhookRequests, 1)
		webhookConfig, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, "push", webhookConfig.EventType)
		assert.Equal(t, "123", webhookConfig.ProjectID)
	})
}
