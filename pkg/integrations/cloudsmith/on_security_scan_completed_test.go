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
// Shared by the cloudsmith webhook-trigger tests.
func cloudsmithSig(secret string, body []byte) string {
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write(body)
	return "sha1=" + hex.EncodeToString(mac.Sum(nil))
}

func Test__OnSecurityScanCompleted__Setup(t *testing.T) {
	trigger := &OnSecurityScanCompleted{}

	t.Run("repository is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": ""},
		})
		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("valid config provisions a webhook for package.security_scanned", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"slug_perm":"wh-sec","events":["package.security_scanned"],"is_active":true}`))},
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
		stored := metadata.Metadata.(OnSecurityScanCompletedMetadata)
		require.NotNil(t, stored.Repository)
		assert.Equal(t, "wh-sec", stored.WebhookID)
	})

	t.Run("reconcile re-asserts the event and active flag on an existing webhook", func(t *testing.T) {
		metadata := &contexts.MetadataContext{
			Metadata: OnSecurityScanCompletedMetadata{
				Repository: &RepositoryRef{Namespace: "weskk", Slug: "superplane-compliance"},
				WebhookURL: "https://sp.example/hook",
				WebhookID:  "wh-existing",
			},
		}
		// GetWebhook: still present and targeting our URL, but disabled and
		// stripped of its event out of band; UpdateWebhook (PATCH) then succeeds.
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"slug_perm":"wh-existing","target_url":"https://sp.example/hook","events":[],"is_active":false}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"slug_perm":"wh-existing","target_url":"https://sp.example/hook","events":["package.security_scanned"],"is_active":true}`))},
			},
		}
		err := trigger.Setup(core.TriggerContext{
			HTTP:          httpCtx,
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			Metadata:      metadata,
			Webhook:       &contexts.NodeWebhookContext{Secret: "node-secret"},
			Configuration: map[string]any{"repository": "weskk/superplane-compliance"},
		})
		require.NoError(t, err)
		// Both responses (GET + PATCH) consumed -> the webhook was reconciled, not skipped.
		assert.Empty(t, httpCtx.Responses)
		require.Len(t, httpCtx.Requests, 2)
		patch := httpCtx.Requests[1]
		assert.Equal(t, http.MethodPatch, patch.Method)
		patchBody, _ := io.ReadAll(patch.Body)
		assert.Contains(t, string(patchBody), `"is_active":true`)
		assert.Contains(t, string(patchBody), "package.security_scanned")
		stored := metadata.Metadata.(OnSecurityScanCompletedMetadata)
		assert.Equal(t, "wh-existing", stored.WebhookID)
	})
}

func Test__OnSecurityScanCompleted__HandleWebhook(t *testing.T) {
	trigger := &OnSecurityScanCompleted{}
	meta := func() *contexts.MetadataContext {
		return &contexts.MetadataContext{
			Metadata: OnSecurityScanCompletedMetadata{Repository: &RepositoryRef{Namespace: "weskk", Slug: "superplane-compliance"}},
		}
	}

	t.Run("emits vulnerability details from the payload context", func(t *testing.T) {
		body := []byte(`{"context":{"vulnerability_scan_results":{"has_vulnerabilities":true,"max_severity":"High","num_vulnerabilities":2}},"data":{"name":"sp-compliance-gpl","version":"1.0.0","slug_perm":"abc","namespace":"weskk","repository":"superplane-compliance","format":"npm","security_scan_status":"2 Vulnerabilities Detected","vulnerability_scan_results_url":"https://api.cloudsmith.io/v1/vulnerabilities/weskk/superplane-compliance/abc/"}}`)
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
		assert.Equal(t, "cloudsmith.package.securityScanned", events.Payloads[0].Type)
		event := events.Payloads[0].Data.(SecurityScanEvent)
		assert.Equal(t, "sp-compliance-gpl", event.Name)
		assert.Equal(t, "2 Vulnerabilities Detected", event.SecurityScanStatus)
		assert.True(t, event.HasVulnerabilities)
		assert.Equal(t, "High", event.MaxSeverity)
		assert.Equal(t, 2, event.NumVulnerabilities)
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
