package gitlab

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
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

	headers := http.Header{}
	headers.Set("X-Gitlab-Event", "Push Hook")

	ctx := core.WebhookRequestContext{
		Headers:       headers,
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123", "actions": []string{"open"}},
	}

	code, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
}

func Test__OnIssue__HandleWebhook__InvalidToken(t *testing.T) {
	trigger := &OnIssue{}

	headers := http.Header{}
	headers.Set("X-Gitlab-Event", "Issue Hook")
	headers.Set("X-Gitlab-Token", "wrong-token")

	webhookCtx := &contexts.WebhookContext{Secret: "correct-token"}

	ctx := core.WebhookRequestContext{
		Headers:       headers,
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123", "actions": []string{"open"}},
		Webhook:       webhookCtx,
	}

	code, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusForbidden, code)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid webhook token")
}

func Test__OnIssue__HandleWebhook__InvalidObjectAttributes(t *testing.T) {
	trigger := &OnIssue{}

	headers := http.Header{}
	headers.Set("X-Gitlab-Event", "Issue Hook")
	headers.Set("X-Gitlab-Token", "token")

	webhookCtx := &contexts.WebhookContext{Secret: "token"}

	body := []byte(`{"some": "data"}`)

	ctx := core.WebhookRequestContext{
		Headers:       headers,
		Body:          body,
		Configuration: map[string]any{"project": "123", "actions": []string{}},
		Webhook:       webhookCtx,
	}

	code, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid object_attributes")
}

func Test__OnIssue__HandleWebhook__StateNotOpened(t *testing.T) {
	trigger := &OnIssue{}

	headers := http.Header{}
	headers.Set("X-Gitlab-Event", "Issue Hook")
	headers.Set("X-Gitlab-Token", "token")

	webhookCtx := &contexts.WebhookContext{Secret: "token"}
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

	webhookCtx := &contexts.WebhookContext{Secret: "token"}
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
	}

	code, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)

	assert.Equal(t, 1, eventsCtx.Count())
	assert.Equal(t, "gitlab.issue", eventsCtx.Payloads[0].Type)
}

func Test__WhitelistedAction__ValidAction(t *testing.T) {
	data := map[string]any{
		"object_attributes": map[string]any{
			"action": "open",
		},
	}

	result := whitelistedAction(data, []string{"open", "close"})
	assert.True(t, result)
}

func Test__WhitelistedAction__InvalidAction(t *testing.T) {
	data := map[string]any{
		"object_attributes": map[string]any{
			"action": "update",
		},
	}

	result := whitelistedAction(data, []string{"open", "close"})
	assert.False(t, result)
}

func Test__WhitelistedAction__MissingAction(t *testing.T) {
	data := map[string]any{
		"object_attributes": map[string]any{},
	}

	result := whitelistedAction(data, []string{"open", "close"})
	assert.False(t, result)
}

func Test__OnIssue__HandleWebhook__UpdateOnClosed(t *testing.T) {
	trigger := &OnIssue{}

	headers := http.Header{}
	headers.Set("X-Gitlab-Event", "Issue Hook")
	headers.Set("X-Gitlab-Token", "token")

	webhookCtx := &contexts.WebhookContext{Secret: "token"}
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
	}

	code, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)

	assert.Equal(t, 0, eventsCtx.Count())
}
