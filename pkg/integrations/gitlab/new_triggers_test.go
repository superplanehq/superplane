package gitlab

import (
	"encoding/json"
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func gitlabHeaders(event, token string) http.Header {
	headers := http.Header{}
	if event != "" {
		headers.Set("X-Gitlab-Event", event)
	}
	if token != "" {
		headers.Set("X-Gitlab-Token", token)
	}

	return headers
}

func Test__OnMergeRequest__HandleWebhook(t *testing.T) {
	trigger := &OnMergeRequest{}

	t.Run("missing event header", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Body:          []byte(`{}`),
			Configuration: map[string]any{"project": "123", "actions": []string{"open"}},
			Logger:        log.NewEntry(log.New()),
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "X-Gitlab-Event")
	})

	t.Run("wrong event type is ignored", func(t *testing.T) {
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
	})

	t.Run("invalid token", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       gitlabHeaders("Merge Request Hook", "wrong"),
			Body:          []byte(`{}`),
			Configuration: map[string]any{"project": "123", "actions": []string{"open"}},
			Webhook:       &contexts.WebhookContext{Secret: "token"},
			Logger:        log.NewEntry(log.New()),
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid webhook token")
	})

	t.Run("action and label match emits event", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"object_attributes": map[string]any{
				"action": "open",
				"title":  "New MR",
			},
			"labels": []map[string]any{
				{"title": "backend"},
			},
		})

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: gitlabHeaders("Merge Request Hook", "token"),
			Body:    body,
			Configuration: map[string]any{
				"project": "123",
				"actions": []string{"open"},
				"labels": []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "backend"},
				},
			},
			Webhook: &contexts.WebhookContext{Secret: "token"},
			Events:  events,
			Logger:  log.NewEntry(log.New()),
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
		assert.Equal(t, "gitlab.mergeRequest", events.Payloads[0].Type)
	})

	t.Run("action does not match", func(t *testing.T) {
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
			Webhook:       &contexts.WebhookContext{Secret: "token"},
			Events:        events,
			Logger:        log.NewEntry(log.New()),
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, events.Count())
	})
}

func Test__OnMilestone__HandleWebhook(t *testing.T) {
	trigger := &OnMilestone{}

	t.Run("top-level action emits event", func(t *testing.T) {
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
	})

	t.Run("object_attributes action emits event", func(t *testing.T) {
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
	})

	t.Run("non-whitelisted action is ignored", func(t *testing.T) {
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
	})
}

func Test__OnRelease__HandleWebhook(t *testing.T) {
	trigger := &OnRelease{}

	t.Run("action match emits event", func(t *testing.T) {
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
	})

	t.Run("action mismatch does not emit", func(t *testing.T) {
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
	})
}

func Test__OnTag__HandleWebhook(t *testing.T) {
	trigger := &OnTag{}

	t.Run("full ref match emits event", func(t *testing.T) {
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
	})

	t.Run("tag name match emits event", func(t *testing.T) {
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
	})

	t.Run("non-matching tag is ignored", func(t *testing.T) {
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
	})
}

func Test__OnVulnerability__HandleWebhook(t *testing.T) {
	trigger := &OnVulnerability{}

	t.Run("wrong event type is ignored", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       gitlabHeaders("Issue Hook", "token"),
			Body:          []byte(`{}`),
			Configuration: map[string]any{"project": "123"},
			Events:        events,
			Logger:        log.NewEntry(log.New()),
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, events.Count())
	})

	t.Run("event is emitted", func(t *testing.T) {
		body := []byte(`{"object_kind":"vulnerability","object_attributes":{"severity":"high"}}`)
		events := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       gitlabHeaders("Vulnerability Hook", "token"),
			Body:          body,
			Configuration: map[string]any{"project": "123"},
			Webhook:       &contexts.WebhookContext{Secret: "token"},
			Events:        events,
			Logger:        log.NewEntry(log.New()),
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
		assert.Equal(t, "gitlab.vulnerability", events.Payloads[0].Type)
	})
}
