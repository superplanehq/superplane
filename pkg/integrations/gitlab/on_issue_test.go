package gitlab

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
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
