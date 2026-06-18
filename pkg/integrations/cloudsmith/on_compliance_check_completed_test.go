package cloudsmith

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// cloudsmithSig computes the X-Cloudsmith-Signature value for a body + secret.
func cloudsmithSig(secret string, body []byte) string {
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write(body)
	return "sha1=" + hex.EncodeToString(mac.Sum(nil))
}

func Test__OnComplianceCheckCompleted__Setup(t *testing.T) {
	trigger := &OnComplianceCheckCompleted{}

	t.Run("repository is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": ""},
		})
		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("valid config provisions a webhook and stores metadata", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"slug_perm":"wh-abc123","events":["package.synced"],"is_active":true}`)),
				},
			},
		}
		metadata := &contexts.MetadataContext{}

		err := trigger.Setup(core.TriggerContext{
			HTTP:          httpCtx,
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			Metadata:      metadata,
			Webhook:       &contexts.NodeWebhookContext{},
			Configuration: map[string]any{"repository": "weskk/superplane-compliance"},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(OnComplianceCheckCompletedMetadata)
		require.True(t, ok)
		require.NotNil(t, stored.Repository)
		assert.Equal(t, "weskk", stored.Repository.Namespace)
		assert.Equal(t, "superplane-compliance", stored.Repository.Slug)
		assert.Equal(t, "wh-abc123", stored.WebhookID)
		assert.NotEmpty(t, stored.WebhookURL)
	})

	t.Run("healthy existing webhook is a no-op", func(t *testing.T) {
		metadata := &contexts.MetadataContext{
			Metadata: OnComplianceCheckCompletedMetadata{
				Repository: &ComplianceRepositoryMetadata{Namespace: "weskk", Slug: "superplane-compliance"},
				WebhookURL: "https://superplane.example/hook",
				WebhookID:  "wh-existing",
			},
		}
		// GetWebhook returns a webhook whose target matches: setup must not recreate.
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"slug_perm":"wh-existing","target_url":"https://superplane.example/hook"}`))},
			},
		}
		err := trigger.Setup(core.TriggerContext{
			HTTP:          httpCtx,
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			Metadata:      metadata,
			Webhook:       &contexts.NodeWebhookContext{},
			Configuration: map[string]any{"repository": "weskk/superplane-compliance"},
		})
		require.NoError(t, err)
		stored := metadata.Metadata.(OnComplianceCheckCompletedMetadata)
		assert.Equal(t, "wh-existing", stored.WebhookID)
	})

	t.Run("re-provisions when the remote webhook is gone", func(t *testing.T) {
		metadata := &contexts.MetadataContext{
			Metadata: OnComplianceCheckCompletedMetadata{
				Repository: &ComplianceRepositoryMetadata{Namespace: "weskk", Slug: "superplane-compliance"},
				WebhookURL: "https://superplane.example/hook",
				WebhookID:  "wh-deleted",
			},
		}
		// GetWebhook 404 -> recreate.
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"detail":"Not found."}`))},
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"slug_perm":"wh-new","events":["package.synced"],"is_active":true}`))},
			},
		}
		err := trigger.Setup(core.TriggerContext{
			HTTP:          httpCtx,
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			Metadata:      metadata,
			Webhook:       &contexts.NodeWebhookContext{},
			Configuration: map[string]any{"repository": "weskk/superplane-compliance"},
		})
		require.NoError(t, err)
		stored := metadata.Metadata.(OnComplianceCheckCompletedMetadata)
		assert.Equal(t, "wh-new", stored.WebhookID)
	})
}

func Test__OnComplianceCheckCompleted__HandleWebhook(t *testing.T) {
	trigger := &OnComplianceCheckCompleted{}

	metadata := func() *contexts.MetadataContext {
		return &contexts.MetadataContext{
			Metadata: OnComplianceCheckCompletedMetadata{
				Repository: &ComplianceRepositoryMetadata{Namespace: "weskk", Slug: "superplane-compliance"},
			},
		}
	}

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:     []byte(`not-json`),
			Events:   &contexts.EventContext{},
			Metadata: metadata(),
			Logger:   log.NewEntry(log.New()),
		})
		assert.Equal(t, http.StatusBadRequest, code)
		require.ErrorContains(t, err, "error parsing webhook body")
	})

	t.Run("emits compliance event from the data envelope", func(t *testing.T) {
		body := []byte(`{"event":"package.synced","data":{"name":"sp-compliance-gpl","version":"1.0.0","slug_perm":"f3XvJCI9ufJa","namespace":"weskk","repository":"superplane-compliance","license":"GPL-3.0-only","spdx_license":"GPL-3.0-only","osi_approved":true,"is_quarantined":true,"status_str":"Quarantined"}}`)
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:     body,
			Events:   events,
			Metadata: metadata(),
			Logger:   log.NewEntry(log.New()),
		})
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, "cloudsmith.package.complianceChecked", events.Payloads[0].Type)

		event, ok := events.Payloads[0].Data.(ComplianceCheckEvent)
		require.True(t, ok)
		assert.Equal(t, "GPL-3.0-only", event.License)
		assert.True(t, event.IsQuarantined)
		assert.Equal(t, "Quarantined", event.Status)
	})

	t.Run("parses a top-level package payload", func(t *testing.T) {
		body := []byte(`{"name":"sp-compliance-mit","slug_perm":"wxu9RDqPfCj0","namespace":"weskk","repository":"superplane-compliance","license":"MIT","is_quarantined":false,"status_str":"Completed"}`)
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:     body,
			Events:   events,
			Metadata: metadata(),
			Logger:   log.NewEntry(log.New()),
		})
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		event := events.Payloads[0].Data.(ComplianceCheckEvent)
		assert.Equal(t, "MIT", event.License)
		assert.False(t, event.IsQuarantined)
	})

	t.Run("rejects an invalid signature when a secret is configured", func(t *testing.T) {
		body := []byte(`{"data":{"name":"x","slug_perm":"y","namespace":"weskk","repository":"superplane-compliance"}}`)
		events := &contexts.EventContext{}
		headers := http.Header{}
		headers.Set("X-Cloudsmith-Signature", "sha1=deadbeef")
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:     body,
			Events:   events,
			Metadata: metadata(),
			Webhook:  &contexts.NodeWebhookContext{Secret: "topsecret"},
			Headers:  headers,
			Logger:   log.NewEntry(log.New()),
		})
		assert.Equal(t, http.StatusForbidden, code)
		require.ErrorContains(t, err, "invalid signature")
		assert.Equal(t, 0, events.Count())
	})

	t.Run("rejects a missing signature when a secret is configured", func(t *testing.T) {
		body := []byte(`{"data":{"name":"x","slug_perm":"y","namespace":"weskk","repository":"superplane-compliance"}}`)
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:     body,
			Events:   events,
			Metadata: metadata(),
			Webhook:  &contexts.NodeWebhookContext{Secret: "topsecret"},
			Headers:  http.Header{},
			Logger:   log.NewEntry(log.New()),
		})
		assert.Equal(t, http.StatusForbidden, code)
		require.ErrorContains(t, err, "missing signature")
		assert.Equal(t, 0, events.Count())
	})

	t.Run("accepts a valid signature and emits", func(t *testing.T) {
		body := []byte(`{"data":{"name":"sp-compliance-apache","slug_perm":"z","namespace":"weskk","repository":"superplane-compliance","license":"Apache-2.0","status_str":"Completed"}}`)
		events := &contexts.EventContext{}
		headers := http.Header{}
		headers.Set("X-Cloudsmith-Signature", cloudsmithSig("topsecret", body))
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:     body,
			Events:   events,
			Metadata: metadata(),
			Webhook:  &contexts.NodeWebhookContext{Secret: "topsecret"},
			Headers:  headers,
			Logger:   log.NewEntry(log.New()),
		})
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		event := events.Payloads[0].Data.(ComplianceCheckEvent)
		assert.Equal(t, "Apache-2.0", event.License)
	})

	t.Run("ignores events from a different repository", func(t *testing.T) {
		body := []byte(`{"event":"package.synced","data":{"name":"other","slug_perm":"x","namespace":"weskk","repository":"some-other-repo"}}`)
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:     body,
			Events:   events,
			Metadata: metadata(),
			Logger:   log.NewEntry(log.New()),
		})
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})
}

func Test__OnComplianceCheckCompleted__Cleanup(t *testing.T) {
	trigger := &OnComplianceCheckCompleted{}

	t.Run("deletes the provisioned webhook", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(""))},
			},
		}
		err := trigger.Cleanup(core.TriggerContext{
			HTTP:        httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			Logger:      log.NewEntry(log.New()),
			Metadata: &contexts.MetadataContext{
				Metadata: OnComplianceCheckCompletedMetadata{
					Repository: &ComplianceRepositoryMetadata{Namespace: "weskk", Slug: "superplane-compliance"},
					WebhookID:  "wh-abc123",
				},
			},
		})
		require.NoError(t, err)
	})

	t.Run("no webhook recorded is a no-op", func(t *testing.T) {
		err := trigger.Cleanup(core.TriggerContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			Logger:      log.NewEntry(log.New()),
			Metadata:    &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})
}
