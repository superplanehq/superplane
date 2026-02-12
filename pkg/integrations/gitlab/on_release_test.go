package gitlab

import (
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnRelease__HandleWebhook__ActionMatch(t *testing.T) {
	trigger := &OnRelease{}
	body := []byte(`{"action":"create","name":"v1.2.0"}`)
	events := &contexts.EventContext{}

	code, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Release Hook", "token"),
		Body:          body,
		Configuration: map[string]any{"project": "123", "actions": []string{"create"}},
		Webhook:       &contexts.WebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.release", events.Payloads[0].Type)
}

func Test__OnRelease__HandleWebhook__ActionMismatch(t *testing.T) {
	trigger := &OnRelease{}
	body := []byte(`{"action":"delete","name":"v1.2.0"}`)
	events := &contexts.EventContext{}

	code, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Release Hook", "token"),
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
