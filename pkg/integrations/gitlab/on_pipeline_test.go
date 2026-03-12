package gitlab

import (
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnPipeline__HandleWebhook__StatusMatch(t *testing.T) {
	trigger := &OnPipeline{}
	body := []byte(`{"object_attributes":{"id":123,"status":"success"}}`)
	events := &contexts.EventContext{}

	code, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Pipeline Hook", "token"),
		Body:          body,
		Configuration: map[string]any{"project": "123", "statuses": []string{"success"}},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.pipeline", events.Payloads[0].Type)
}

func Test__OnPipeline__HandleWebhook__StatusMismatch(t *testing.T) {
	trigger := &OnPipeline{}
	body := []byte(`{"object_attributes":{"id":123,"status":"running"}}`)
	events := &contexts.EventContext{}

	code, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Pipeline Hook", "token"),
		Body:          body,
		Configuration: map[string]any{"project": "123", "statuses": []string{"success"}},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnPipeline__HandleWebhook__MissingStatus(t *testing.T) {
	trigger := &OnPipeline{}
	body := []byte(`{"object_attributes":{"id":123}}`)
	events := &contexts.EventContext{}

	code, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Pipeline Hook", "token"),
		Body:          body,
		Configuration: map[string]any{"project": "123", "statuses": []string{"success"}},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusBadRequest, code)
	assert.Error(t, err)
	assert.Zero(t, events.Count())
}
