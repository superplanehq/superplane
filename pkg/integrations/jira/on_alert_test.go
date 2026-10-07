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

func Test__OnAlert__Setup(t *testing.T) {
	trigger := &OnAlert{}

	t.Run("at least one event is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"events": []string{}},
		})
		require.ErrorContains(t, err, "at least one event")
	})

	t.Run("no team selected requests a dedicated alert webhook for every team", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		integration := newAuthorizedIntegration()
		err := trigger.Setup(core.TriggerContext{
			Integration:   integration,
			Metadata:      metadata,
			Configuration: map[string]any{"events": []string{"created"}},
		})
		require.NoError(t, err)

		stored := metadata.Metadata.(OnAlertMetadata)
		assert.Nil(t, stored.Team)

		require.Len(t, integration.WebhookRequests, 1)
		assert.Equal(t, WebhookConfiguration{Kind: webhookKindAlert}, integration.WebhookRequests[0])
	})

	t.Run("valid team resolves it and scopes the requested webhook", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`{"platformTeams":[{"teamId":"team-1","teamName":"Platform"}]}`,
				))},
			},
		}
		metadata := &contexts.MetadataContext{}
		integration := newAuthorizedIntegration()
		err := trigger.Setup(core.TriggerContext{
			HTTP:          httpCtx,
			Integration:   integration,
			Metadata:      metadata,
			Configuration: map[string]any{"team": "team-1", "events": []string{"created"}},
		})
		require.NoError(t, err)

		stored := metadata.Metadata.(OnAlertMetadata)
		require.NotNil(t, stored.Team)
		assert.Equal(t, "Platform", stored.Team.TeamName)

		require.Len(t, integration.WebhookRequests, 1)
		assert.Equal(t, WebhookConfiguration{Kind: webhookKindAlert, TeamID: "team-1"}, integration.WebhookRequests[0])
	})

	t.Run("unknown team -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"platformTeams":[]}`))},
			},
		}
		err := trigger.Setup(core.TriggerContext{
			HTTP:          httpCtx,
			Integration:   newAuthorizedIntegration(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "team-99", "events": []string{"created"}},
		})
		require.ErrorContains(t, err, "team team-99 not found")
	})
}

func Test__OnAlert__HandleWebhook(t *testing.T) {
	trigger := &OnAlert{}

	body := func(action string) []byte {
		return []byte(`{
			"action": "` + action + `",
			"alert": {
				"alertId": "alert-1",
				"tinyId": "42",
				"message": "CPU usage above 90%",
				"priority": "P2",
				"tags": ["production"]
			}
		}`)
	}

	t.Run("emits a created event for a configured action", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body("Create"),
			Events:        events,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"events": []string{"created"}},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, AlertEventPayloadType, events.Payloads[0].Type)
		event := events.Payloads[0].Data.(OnAlertEvent)
		assert.Equal(t, "created", event.Action)
		require.NotNil(t, event.Alert)
		assert.Equal(t, "alert-1", event.Alert.AlertID)
		assert.Equal(t, "P2", event.Alert.Priority)
	})

	t.Run("emits an acknowledged event", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body("Acknowledge"),
			Events:        events,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"events": []string{"acknowledged"}},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, "acknowledged", events.Payloads[0].Data.(OnAlertEvent).Action)
	})

	t.Run("ignores actions not in configured events", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body("Create"),
			Events:        events,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"events": []string{"closed"}},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("ignores unsupported actions", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body("Escalate"),
			Events:        events,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"events": []string{"created", "acknowledged", "closed", "deleted"}},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("ignores a payload missing an alert", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte(`{"action": "Create"}`),
			Events:        events,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"events": []string{"created"}},
			Headers:       http.Header{},
			Logger:        log.NewEntry(log.New()),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})
}
