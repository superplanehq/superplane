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

func Test__OnPackageCreated__Setup(t *testing.T) {
	trigger := &OnPackageCreated{}

	t.Run("repository is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": ""},
		})
		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("valid config provisions a webhook for package.created", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"slug_perm":"wh-pc","events":["package.created"],"is_active":true}`))},
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
		stored := metadata.Metadata.(OnPackageCreatedMetadata)
		require.NotNil(t, stored.Repository)
		assert.Equal(t, "weskk", stored.Repository.Namespace)
		assert.Equal(t, "wh-pc", stored.WebhookID)
	})

	t.Run("reconcile re-asserts the event and active flag on an existing webhook", func(t *testing.T) {
		const webhookURL = "http://localhost:3000/api/v1/webhooks/wh-pc"
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// GetWebhook: still present and targeting our URL, but disabled and
				// stripped of its event out of band at Cloudsmith.
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"slug_perm":"wh-pc","target_url":"` + webhookURL + `","events":[],"is_active":false}`))},
				// UpdateWebhook (PATCH) succeeds.
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"slug_perm":"wh-pc","target_url":"` + webhookURL + `","events":["package.created"],"is_active":true}`))},
			},
		}
		metadata := &contexts.MetadataContext{
			Metadata: OnPackageCreatedMetadata{
				Repository: &RepositoryRef{Namespace: "weskk", Slug: "superplane-compliance"},
				WebhookURL: webhookURL,
				WebhookID:  "wh-pc",
			},
		}
		err := trigger.Setup(core.TriggerContext{
			HTTP:          httpCtx,
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			Metadata:      metadata,
			Webhook:       &contexts.NodeWebhookContext{Secret: "s3cr3t"},
			Configuration: map[string]any{"repository": "weskk/superplane-compliance"},
		})
		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 2)
		patch := httpCtx.Requests[1]
		assert.Equal(t, http.MethodPatch, patch.Method)
		patchBody, _ := io.ReadAll(patch.Body)
		assert.Contains(t, string(patchBody), `"is_active":true`)
		assert.Contains(t, string(patchBody), "package.created")
	})
}

func Test__OnPackageCreated__HandleWebhook(t *testing.T) {
	trigger := &OnPackageCreated{}
	meta := func() *contexts.MetadataContext {
		return &contexts.MetadataContext{
			Metadata: OnPackageCreatedMetadata{Repository: &RepositoryRef{Namespace: "weskk", Slug: "superplane-compliance"}},
		}
	}

	t.Run("emits package created event with a valid signature", func(t *testing.T) {
		body := []byte(`{"data":{"name":"sp-compliance-mit","version":"1.0.0","slug_perm":"wxu9RDqPfCj0","namespace":"weskk","repository":"superplane-compliance","format":"npm","license":"MIT","uploader":"superplane-dnig","uploaded_at":"2026-06-17T14:50:00Z","status_str":"Completed"}}`)
		events := &contexts.EventContext{}
		headers := http.Header{}
		headers.Set("X-Cloudsmith-Signature", cloudsmithSig("s3cr3t", body))
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:     body,
			Events:   events,
			Metadata: meta(),
			Webhook:  &contexts.NodeWebhookContext{Secret: "s3cr3t"},
			Headers:  headers,
			Logger:   log.NewEntry(log.New()),
		})
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, "cloudsmith.package.created", events.Payloads[0].Type)
		event := events.Payloads[0].Data.(PackageCreatedEvent)
		assert.Equal(t, "sp-compliance-mit", event.Name)
		assert.Equal(t, "MIT", event.License)
		assert.Equal(t, "superplane-dnig", event.Uploader)
	})

	t.Run("rejects an invalid signature", func(t *testing.T) {
		body := []byte(`{"data":{"name":"x","slug_perm":"y","namespace":"weskk","repository":"superplane-compliance"}}`)
		events := &contexts.EventContext{}
		headers := http.Header{}
		headers.Set("X-Cloudsmith-Signature", "sha1=bad")
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:     body,
			Events:   events,
			Metadata: meta(),
			Webhook:  &contexts.NodeWebhookContext{Secret: "s3cr3t"},
			Headers:  headers,
			Logger:   log.NewEntry(log.New()),
		})
		assert.Equal(t, http.StatusForbidden, code)
		require.ErrorContains(t, err, "invalid signature")
		assert.Equal(t, 0, events.Count())
	})
}
