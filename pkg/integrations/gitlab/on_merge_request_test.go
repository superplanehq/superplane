package gitlab

import (
	"encoding/json"
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnMergeRequest__HandleWebhook__MissingEventHeader(t *testing.T) {
	trigger := &OnMergeRequest{}

	code, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       http.Header{},
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123", "actions": []string{"open"}},
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusBadRequest, code)
	assert.ErrorContains(t, err, "X-Gitlab-Event")
}

func Test__OnMergeRequest__HandleWebhook__WrongEventType(t *testing.T) {
	trigger := &OnMergeRequest{}
	events := &contexts.EventContext{}

	code, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Issue Hook", "token"),
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123", "actions": []string{"open"}},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnMergeRequest__HandleWebhook__InvalidToken(t *testing.T) {
	trigger := &OnMergeRequest{}

	code, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Merge Request Hook", "wrong"),
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123", "actions": []string{"open"}},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusForbidden, code)
	assert.ErrorContains(t, err, "invalid webhook token")
}

func Test__OnMergeRequest__HandleWebhook__ActionMatch(t *testing.T) {
	trigger := &OnMergeRequest{}

	body, _ := json.Marshal(map[string]any{
		"object_attributes": map[string]any{
			"action": "open",
			"title":  "New MR",
		},
	})

	events := &contexts.EventContext{}
	code, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers: gitlabHeaders("Merge Request Hook", "token"),
		Body:    body,
		Configuration: map[string]any{
			"project": "123",
			"actions": []string{"open"},
		},
		Webhook: &contexts.NodeWebhookContext{Secret: "token"},
		Events:  events,
		Logger:  log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.mergeRequest", events.Payloads[0].Type)
}

func Test__OnMergeRequest__HandleWebhook__ActionMismatch(t *testing.T) {
	trigger := &OnMergeRequest{}

	body, _ := json.Marshal(map[string]any{
		"object_attributes": map[string]any{
			"action": "merge",
		},
	})

	events := &contexts.EventContext{}
	code, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Merge Request Hook", "token"),
		Body:          body,
		Configuration: map[string]any{"project": "123", "actions": []string{"open"}},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}
