package gitlab

import (
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnTag__HandleWebhook__FullRefMatch(t *testing.T) {
	trigger := &OnTag{}
	body := []byte(`{"ref":"refs/tags/v1.0.0","event_name":"tag_push"}`)
	events := &contexts.EventContext{}

	code, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers: gitlabHeaders("Tag Push Hook", "token"),
		Body:    body,
		Configuration: map[string]any{
			"project": "123",
			"tags": []configuration.Predicate{
				{Type: configuration.PredicateTypeEquals, Value: "refs/tags/v1.0.0"},
			},
		},
		Webhook: &contexts.WebhookContext{Secret: "token"},
		Events:  events,
		Logger:  log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.tag", events.Payloads[0].Type)
}

func Test__OnTag__HandleWebhook__TagNameMatch(t *testing.T) {
	trigger := &OnTag{}
	body := []byte(`{"ref":"refs/tags/v1.0.0","event_name":"tag_push"}`)
	events := &contexts.EventContext{}

	code, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers: gitlabHeaders("Tag Push Hook", "token"),
		Body:    body,
		Configuration: map[string]any{
			"project": "123",
			"tags": []configuration.Predicate{
				{Type: configuration.PredicateTypeEquals, Value: "v1.0.0"},
			},
		},
		Webhook: &contexts.WebhookContext{Secret: "token"},
		Events:  events,
		Logger:  log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.tag", events.Payloads[0].Type)
}

func Test__OnTag__HandleWebhook__TagMismatch(t *testing.T) {
	trigger := &OnTag{}
	body := []byte(`{"ref":"refs/tags/v2.0.0","event_name":"tag_push"}`)
	events := &contexts.EventContext{}

	code, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers: gitlabHeaders("Tag Push Hook", "token"),
		Body:    body,
		Configuration: map[string]any{
			"project": "123",
			"tags": []configuration.Predicate{
				{Type: configuration.PredicateTypeEquals, Value: "v1.0.0"},
			},
		},
		Webhook: &contexts.WebhookContext{Secret: "token"},
		Events:  events,
		Logger:  log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}
