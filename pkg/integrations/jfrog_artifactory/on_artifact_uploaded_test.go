package jfrogartifactory

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func signBody(secret string, body []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func jfrogHeaders(secret string, body []byte) http.Header {
	headers := http.Header{}
	headers.Set("X-JFrog-Event-Auth", signBody(secret, body))
	return headers
}

func Test__OnArtifactUploaded__TriggerInfo(t *testing.T) {
	trigger := &OnArtifactUploaded{}

	assert.Equal(t, "jfrogArtifactory.onArtifactUploaded", trigger.Name())
	assert.Equal(t, "On Artifact Uploaded", trigger.Label())
	assert.Equal(t, "jfrogArtifactory", trigger.Icon())
	assert.Equal(t, "green", trigger.Color())
	assert.NotEmpty(t, trigger.Description())
	assert.NotEmpty(t, trigger.Documentation())
}

func Test__OnArtifactUploaded__Configuration(t *testing.T) {
	trigger := &OnArtifactUploaded{}
	config := trigger.Configuration()

	require.Len(t, config, 1)
	assert.Equal(t, "repository", config[0].Name)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, config[0].Type)
	assert.False(t, config[0].Required)
}

func Test__OnArtifactUploaded__Setup(t *testing.T) {
	trigger := &OnArtifactUploaded{}

	t.Run("no repository -> requests webhook with empty repo", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Configuration: map[string]any{},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)
		config := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Equal(t, "", config.Repository)
	})

	t.Run("with repository -> requests webhook with repo", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Configuration: map[string]any{"repository": "libs-release-local"},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)
		config := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Equal(t, "libs-release-local", config.Repository)
	})
}

func Test__OnArtifactUploaded__HandleWebhook(t *testing.T) {
	trigger := &OnArtifactUploaded{}

	t.Run("missing signature header -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
			Body:    []byte(`{}`),
			Logger:  log.NewEntry(log.New()),
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "X-JFrog-Event-Auth")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-JFrog-Event-Auth", "invalidsignature")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body:    []byte(`{}`),
			Webhook: &contexts.WebhookContext{Secret: "test-secret"},
			Logger:  log.NewEntry(log.New()),
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid webhook signature")
	})

	t.Run("wrong event_type -> no emit, returns 200", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"domain":     "artifact",
			"event_type": "deleted",
			"data": map[string]any{
				"repo_key": "libs-release-local",
				"path":     "com/example/artifact-1.0.jar",
				"name":     "artifact-1.0.jar",
			},
		})

		secret := "test-secret"
		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: jfrogHeaders(secret, body),
			Body:    body,
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  events,
			Logger:  log.NewEntry(log.New()),
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, events.Count())
	})

	t.Run("deployed event -> emits event with correct payload", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"domain":     "artifact",
			"event_type": "deployed",
			"data": map[string]any{
				"repo_key": "libs-release-local",
				"path":     "com/example/artifact-1.0.jar",
				"name":     "artifact-1.0.jar",
				"size":     12345,
				"sha256":   "abc123",
			},
		})

		secret := "test-secret"
		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       jfrogHeaders(secret, body),
			Body:          body,
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        events,
			Logger:        log.NewEntry(log.New()),
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, "jfrogArtifactory.artifactUploaded", events.Payloads[0].Type)

		data, ok := events.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "libs-release-local", data["repo"])
		assert.Equal(t, "com/example/artifact-1.0.jar", data["path"])
		assert.Equal(t, "artifact-1.0.jar", data["name"])
		assert.Equal(t, "abc123", data["sha256"])
	})

	t.Run("deployed event with no repo filter -> emits event", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"domain":     "artifact",
			"event_type": "deployed",
			"data": map[string]any{
				"repo_key": "any-repo",
				"path":     "some/path/artifact.jar",
				"name":     "artifact.jar",
				"size":     999,
				"sha256":   "def456",
			},
		})

		secret := "test-secret"
		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       jfrogHeaders(secret, body),
			Body:          body,
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        events,
			Logger:        log.NewEntry(log.New()),
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("deployed event with matching repo filter -> emits event", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"domain":     "artifact",
			"event_type": "deployed",
			"data": map[string]any{
				"repo_key": "libs-release-local",
				"path":     "com/example/artifact-1.0.jar",
				"name":     "artifact-1.0.jar",
				"size":     12345,
				"sha256":   "abc123",
			},
		})

		secret := "test-secret"
		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       jfrogHeaders(secret, body),
			Body:          body,
			Configuration: map[string]any{"repository": "libs-release-local"},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        events,
			Logger:        log.NewEntry(log.New()),
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("deployed event with non-matching repo filter -> skipped", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"domain":     "artifact",
			"event_type": "deployed",
			"data": map[string]any{
				"repo_key": "libs-snapshot-local",
				"path":     "com/example/artifact-1.0.jar",
				"name":     "artifact-1.0.jar",
				"size":     12345,
				"sha256":   "abc123",
			},
		})

		secret := "test-secret"
		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       jfrogHeaders(secret, body),
			Body:          body,
			Configuration: map[string]any{"repository": "libs-release-local"},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        events,
			Logger:        log.NewEntry(log.New()),
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, events.Count())
	})
}
