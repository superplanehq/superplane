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

const commitStatusBody = `{
	"object_kind": "pipeline",
	"object_attributes": {"id": 12345, "ref": "main", "sha": "f4f6c5a0d2e5ad34be4c17c3f166f4d2ff8b0a55", "status": "success", "source": "external"},
	"builds": [
		{"id": 998877, "stage": "external", "name": "security-scan", "status": "success"},
		{"id": 998876, "stage": "test", "name": "unit-tests", "status": "success"}
	]
}`

func onCommitStatusRequest(body string, config map[string]any, events *contexts.EventContext) core.WebhookRequestContext {
	return core.WebhookRequestContext{
		Headers:       gitlabHeaders("Pipeline Hook", "token"),
		Body:          []byte(body),
		Configuration: config,
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	}
}

func Test__OnCommitStatus__HandleWebhook__MissingEventHeader(t *testing.T) {
	trigger := &OnCommitStatus{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       http.Header{},
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123"},
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusBadRequest, code)
	assert.ErrorContains(t, err, "X-Gitlab-Event")
}

func Test__OnCommitStatus__HandleWebhook__WrongEventType(t *testing.T) {
	trigger := &OnCommitStatus{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Push Hook", "token"),
		Body:          []byte(commitStatusBody),
		Configuration: map[string]any{"project": "123"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnCommitStatus__HandleWebhook__MissingToken(t *testing.T) {
	trigger := &OnCommitStatus{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Pipeline Hook", ""),
		Body:          []byte(commitStatusBody),
		Configuration: map[string]any{"project": "123"},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusForbidden, code)
	assert.ErrorContains(t, err, "missing X-Gitlab-Token header")
	assert.Zero(t, events.Count())
}

func Test__OnCommitStatus__HandleWebhook__InvalidToken(t *testing.T) {
	trigger := &OnCommitStatus{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Pipeline Hook", "wrong"),
		Body:          []byte(commitStatusBody),
		Configuration: map[string]any{"project": "123"},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusForbidden, code)
	assert.ErrorContains(t, err, "invalid webhook token")
	assert.Zero(t, events.Count())
}

func Test__OnCommitStatus__HandleWebhook__MissingObjectAttributes(t *testing.T) {
	trigger := &OnCommitStatus{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(onCommitStatusRequest(
		`{"object_kind":"pipeline"}`,
		map[string]any{"project": "123", "statuses": []string{"success"}},
		events,
	))

	assert.Equal(t, http.StatusBadRequest, code)
	assert.ErrorContains(t, err, "object_attributes missing")
	assert.Zero(t, events.Count())
}

func Test__OnCommitStatus__HandleWebhook__MissingStatus(t *testing.T) {
	trigger := &OnCommitStatus{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(onCommitStatusRequest(
		`{"object_attributes":{"id":12345,"ref":"main"}}`,
		map[string]any{"project": "123", "statuses": []string{"success"}},
		events,
	))

	assert.Equal(t, http.StatusBadRequest, code)
	assert.ErrorContains(t, err, "status missing")
	assert.Zero(t, events.Count())
}

func Test__OnCommitStatus__HandleWebhook__StatusMatch(t *testing.T) {
	trigger := &OnCommitStatus{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(onCommitStatusRequest(
		commitStatusBody,
		map[string]any{"project": "123", "statuses": []string{"success", "failed"}},
		events,
	))

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	require.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.commitStatusChanged", events.Payloads[0].Type)
}

func Test__OnCommitStatus__HandleWebhook__StatusMismatch(t *testing.T) {
	trigger := &OnCommitStatus{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(onCommitStatusRequest(
		commitStatusBody,
		map[string]any{"project": "123", "statuses": []string{"failed"}},
		events,
	))

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnCommitStatus__HandleWebhook__NameMatch(t *testing.T) {
	trigger := &OnCommitStatus{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(onCommitStatusRequest(
		commitStatusBody,
		map[string]any{
			"project":  "123",
			"statuses": []string{"success"},
			"names": []configuration.Predicate{
				{Type: configuration.PredicateTypeEquals, Value: "security-scan"},
			},
		},
		events,
	))

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	require.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.commitStatusChanged", events.Payloads[0].Type)
}

func Test__OnCommitStatus__HandleWebhook__NameMismatch(t *testing.T) {
	trigger := &OnCommitStatus{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(onCommitStatusRequest(
		commitStatusBody,
		map[string]any{
			"project":  "123",
			"statuses": []string{"success"},
			"names": []configuration.Predicate{
				{Type: configuration.PredicateTypeEquals, Value: "lint"},
			},
		},
		events,
	))

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnCommitStatus__HandleWebhook__RefMatch(t *testing.T) {
	trigger := &OnCommitStatus{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(onCommitStatusRequest(
		commitStatusBody,
		map[string]any{
			"project":  "123",
			"statuses": []string{"success"},
			"refs": []configuration.Predicate{
				{Type: configuration.PredicateTypeMatches, Value: "^ma.*"},
			},
		},
		events,
	))

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	require.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.commitStatusChanged", events.Payloads[0].Type)
}

func Test__OnCommitStatus__HandleWebhook__RefMismatch(t *testing.T) {
	trigger := &OnCommitStatus{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(onCommitStatusRequest(
		commitStatusBody,
		map[string]any{
			"project":  "123",
			"statuses": []string{"success"},
			"refs": []configuration.Predicate{
				{Type: configuration.PredicateTypeEquals, Value: "release"},
			},
		},
		events,
	))

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnCommitStatus__HandleWebhook__NoFiltersEmitsEveryStatus(t *testing.T) {
	trigger := &OnCommitStatus{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(onCommitStatusRequest(
		commitStatusBody,
		map[string]any{"project": "123"},
		events,
	))

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	require.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.commitStatusChanged", events.Payloads[0].Type)
}

func Test__OnCommitStatus__Setup(t *testing.T) {
	trigger := &OnCommitStatus{}
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

	t.Run("metadata is set and pipeline webhook is requested", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{Metadata: metadata}

		require.NoError(t, trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": "123", "statuses": []string{"success"}},
		}))

		require.Len(t, integrationCtx.WebhookRequests, 1)
		webhookConfig, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, "pipeline", webhookConfig.EventType)
		assert.Equal(t, "123", webhookConfig.ProjectID)
	})
}
