package cloudsmith

import (
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

	t.Run("already provisioned for the same repository is a no-op", func(t *testing.T) {
		metadata := &contexts.MetadataContext{
			Metadata: OnComplianceCheckCompletedMetadata{
				Repository: &ComplianceRepositoryMetadata{Namespace: "weskk", Slug: "superplane-compliance"},
				WebhookURL: "https://superplane.example/hook",
				WebhookID:  "wh-existing",
			},
		}
		// No HTTP responses configured: a no-op must not make API calls.
		err := trigger.Setup(core.TriggerContext{
			HTTP:          &contexts.HTTPContext{},
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			Metadata:      metadata,
			Webhook:       &contexts.NodeWebhookContext{},
			Configuration: map[string]any{"repository": "weskk/superplane-compliance"},
		})
		require.NoError(t, err)
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
