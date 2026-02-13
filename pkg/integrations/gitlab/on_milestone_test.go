package gitlab

import (
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnMilestone__HandleWebhook__TopLevelAction(t *testing.T) {
	trigger := &OnMilestone{}
	body := []byte(`{"action":"create","object_attributes":{"title":"v1.0"}}`)
	events := &contexts.EventContext{}

	code, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Milestone Hook", "token"),
		Body:          body,
		Configuration: map[string]any{"project": "123", "actions": []string{"create"}},
		Webhook:       &contexts.WebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.milestone", events.Payloads[0].Type)
}

func Test__OnMilestone__HandleWebhook__ObjectAttributesAction(t *testing.T) {
	trigger := &OnMilestone{}
	body := []byte(`{"object_attributes":{"action":"reopen","title":"v1.0"}}`)
	events := &contexts.EventContext{}

	code, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Milestone Hook", "token"),
		Body:          body,
		Configuration: map[string]any{"project": "123", "actions": []string{"reopen"}},
		Webhook:       &contexts.WebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.milestone", events.Payloads[0].Type)
}

func Test__OnMilestone__HandleWebhook__NonWhitelistedAction(t *testing.T) {
	trigger := &OnMilestone{}
	body := []byte(`{"action":"close"}`)
	events := &contexts.EventContext{}

	code, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Milestone Hook", "token"),
		Body:          body,
		Configuration: map[string]any{"project": "123", "actions": []string{"create"}},
		Webhook:       &contexts.WebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}
