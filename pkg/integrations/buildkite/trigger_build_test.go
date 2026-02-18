package buildkite

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test_TriggerBuild_HandleWebhook_Deprecated(t *testing.T) {
	component := &TriggerBuild{}

	webhookCtx := &contexts.WebhookContext{Secret: "test-secret"}
	eventCtx := &contexts.EventContext{}

	ctx := core.WebhookRequestContext{
		Headers: http.Header{
			"X-Buildkite-Event": []string{"build.finished"},
		},
		Body:    createTriggerBuildPayload("build-123", "passed", false),
		Events:  eventCtx,
		Webhook: webhookCtx,
	}

	statusCode, err := component.HandleWebhook(ctx)

	assert.Equal(t, http.StatusOK, statusCode)
	assert.NoError(t, err)

	assert.Equal(t, 0, eventCtx.Count())
}

func createTriggerBuildPayload(buildID, state string, blocked bool) []byte {
	payload := map[string]any{
		"event": "build.finished",
		"build": map[string]any{
			"id":      buildID,
			"state":   state,
			"blocked": blocked,
		},
		"pipeline": map[string]any{
			"slug": "test-pipeline",
		},
		"organization": map[string]any{
			"slug": "test-org",
		},
	}
	data, _ := json.Marshal(payload)
	return data
}
