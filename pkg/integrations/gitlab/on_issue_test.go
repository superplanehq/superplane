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

func Test__OnIssue__HandleWebhook__MissingEventHeader(t *testing.T) {
	trigger := &OnIssue{}

	ctx := core.WebhookRequestContext{
		Headers:       http.Header{},
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123", "actions": []string{"open"}},
	}

	code, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "X-Gitlab-Event")
}

func Test__OnIssue__HandleWebhook__WrongEventType(t *testing.T) {
	trigger := &OnIssue{}

	eventsCtx := &contexts.EventContext{}
	headers := http.Header{}
	headers.Set("X-Gitlab-Event", "Push Hook")

	ctx := core.WebhookRequestContext{
		Headers:       headers,
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123", "actions": []string{"open"}},
		Events:        eventsCtx,
		Logger:        log.NewEntry(log.New()),
	}

	code, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, eventsCtx.Count())
}

func Test__OnIssue__HandleWebhook__InvalidToken(t *testing.T) {
	trigger := &OnIssue{}

	headers := http.Header{}
	headers.Set("X-Gitlab-Event", "Issue Hook")
	headers.Set("X-Gitlab-Token", "wrong-token")

	webhookCtx := &contexts.NodeWebhookContext{Secret: "correct-token"}

	ctx := core.WebhookRequestContext{
		Headers:       headers,
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123", "actions": []string{"open"}},
		Webhook:       webhookCtx,
		Logger:        log.NewEntry(log.New()),
	}

	code, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusForbidden, code)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid webhook token")
}

func Test__OnIssue__HandleWebhook__StateNotOpened(t *testing.T) {
	trigger := &OnIssue{}

	headers := http.Header{}
	headers.Set("X-Gitlab-Event", "Issue Hook")
	headers.Set("X-Gitlab-Token", "token")

	webhookCtx := &contexts.NodeWebhookContext{Secret: "token"}
	eventsCtx := &contexts.EventContext{}

	data := map[string]any{
		"object_attributes": map[string]any{
			"state":  "closed",
			"action": "close",
		},
	}
	body, _ := json.Marshal(data)

	ctx := core.WebhookRequestContext{
		Headers:       headers,
		Body:          body,
		Configuration: map[string]any{"project": "123", "actions": []string{"close"}},
		Webhook:       webhookCtx,
		Events:        eventsCtx,
		Logger:        log.NewEntry(log.New()),
	}

	code, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)

	assert.Equal(t, 1, eventsCtx.Count())
	assert.Equal(t, "gitlab.issue", eventsCtx.Payloads[0].Type)
}

func Test__OnIssue__HandleWebhook__Success(t *testing.T) {
	trigger := &OnIssue{}

	headers := http.Header{}
	headers.Set("X-Gitlab-Event", "Issue Hook")
	headers.Set("X-Gitlab-Token", "token")

	webhookCtx := &contexts.NodeWebhookContext{Secret: "token"}
	eventsCtx := &contexts.EventContext{}

	data := map[string]any{
		"object_attributes": map[string]any{
			"state":  "opened",
			"action": "open",
			"title":  "Test Issue",
		},
	}
	body, _ := json.Marshal(data)

	ctx := core.WebhookRequestContext{
		Headers:       headers,
		Body:          body,
		Configuration: map[string]any{"project": "123", "actions": []string{"open"}},
		Webhook:       webhookCtx,
		Events:        eventsCtx,
		Logger:        log.NewEntry(log.New()),
	}

	code, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)

	assert.Equal(t, 1, eventsCtx.Count())
	assert.Equal(t, "gitlab.issue", eventsCtx.Payloads[0].Type)
}

func Test__OnIssue__HandleWebhook__Filters(t *testing.T) {
	trigger := &OnIssue{}

	headers := http.Header{}
	headers.Set("X-Gitlab-Event", "Issue Hook")
	headers.Set("X-Gitlab-Token", "token")

	webhookCtx := &contexts.NodeWebhookContext{Secret: "token"}

	baseAttributes := map[string]any{
		"state":  "opened",
		"action": "open",
	}

	t.Run("label match", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		data := map[string]any{
			"object_attributes": baseAttributes,
			"labels": []map[string]any{
				{"title": "bug"},
				{"title": "backend"},
			},
		}
		body, _ := json.Marshal(data)

		ctx := core.WebhookRequestContext{
			Headers:       headers,
			Body:          body,
			Configuration: map[string]any{"project": "123", "actions": []string{"open"}, "labels": []configuration.Predicate{{Type: configuration.PredicateTypeEquals, Value: "backend"}}},
			Webhook:       webhookCtx,
			Events:        eventsCtx,
			Logger:        log.NewEntry(log.New()),
		}

		code, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)

		assert.Equal(t, 1, eventsCtx.Count())
		assert.Equal(t, "gitlab.issue", eventsCtx.Payloads[0].Type)
	})

	t.Run("label no match", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		data := map[string]any{
			"object_attributes": baseAttributes,
			"labels": []map[string]any{
				{"title": "bug"},
			},
		}
		body, _ := json.Marshal(data)

		ctx := core.WebhookRequestContext{
			Headers:       headers,
			Body:          body,
			Configuration: map[string]any{"project": "123", "actions": []string{"open"}, "labels": []configuration.Predicate{{Type: configuration.PredicateTypeEquals, Value: "backend"}}},
			Webhook:       webhookCtx,
			Events:        eventsCtx,
			Logger:        log.NewEntry(log.New()),
		}

		code, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)

		assert.Zero(t, eventsCtx.Count())
	})

}

func Test__WhitelistedAction__ValidAction(t *testing.T) {
	trigger := &OnIssue{}

	t.Run("valid action", func(t *testing.T) {
		data := map[string]any{
			"object_attributes": map[string]any{
				"action": "open",
			},
		}
		result := trigger.whitelistedAction(log.NewEntry(log.New()), data, []string{"open", "close"})
		assert.True(t, result)
	})

	t.Run("invalid action", func(t *testing.T) {
		data := map[string]any{
			"object_attributes": map[string]any{
				"action": "update",
			},
		}

		result := trigger.whitelistedAction(log.NewEntry(log.New()), data, []string{"open", "close"})
		assert.False(t, result)
	})

	t.Run("missing action", func(t *testing.T) {
		data := map[string]any{
			"object_attributes": map[string]any{},
		}

		result := trigger.whitelistedAction(log.NewEntry(log.New()), data, []string{"open", "close"})
		assert.False(t, result)
	})

}

func Test__OnIssue__HandleWebhook__UpdateOnClosed(t *testing.T) {
	trigger := &OnIssue{}

	headers := http.Header{}
	headers.Set("X-Gitlab-Event", "Issue Hook")
	headers.Set("X-Gitlab-Token", "token")

	webhookCtx := &contexts.NodeWebhookContext{Secret: "token"}
	eventsCtx := &contexts.EventContext{}

	data := map[string]any{
		"object_attributes": map[string]any{
			"state":  "closed",
			"action": "update",
		},
	}
	body, _ := json.Marshal(data)

	ctx := core.WebhookRequestContext{
		Headers:       headers,
		Body:          body,
		Configuration: map[string]any{"project": "123", "actions": []string{"update"}},
		Webhook:       webhookCtx,
		Events:        eventsCtx,
		Logger:        log.NewEntry(log.New()),
	}

	code, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)

	assert.Equal(t, 0, eventsCtx.Count())
}
