package monitoring

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnAlert__Setup(t *testing.T) {
	tr := &OnAlert{}

	t.Run("creates a webhook notification channel and stores metadata", func(t *testing.T) {
		var postURL string
		var postBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				postURL = url
				postBody, _ = body.(map[string]any)
				return []byte(`{"name":"projects/my-project/notificationChannels/123","type":"webhook_basicauth"}`), nil
			},
		}
		withFactory(mc)

		meta := &contexts.MetadataContext{}
		err := tr.Setup(core.TriggerContext{
			Configuration: map[string]any{"states": []string{"open"}},
			Integration:   &contexts.IntegrationContext{},
			Webhook:       &contexts.NodeWebhookContext{Secret: "test-webhook-secret"},
			Metadata:      meta,
		})

		require.NoError(t, err)
		assert.Contains(t, postURL, "/v3/projects/my-project/notificationChannels")
		// Signed delivery: a Basic-auth channel carrying the node's webhook secret.
		assert.Equal(t, "webhook_basicauth", postBody["type"])
		labels := postBody["labels"].(map[string]any)
		assert.Equal(t, webhookAuthUsername, labels["username"])
		assert.Equal(t, "test-webhook-secret", labels["password"])

		stored := OnAlertMetadata{}
		require.NoError(t, mapstructure.Decode(meta.Get(), &stored))
		assert.Equal(t, "projects/my-project/notificationChannels/123", stored.NotificationChannel)
		assert.NotEmpty(t, stored.WebhookURL)
		// The channel points at the node's webhook URL.
		assert.Equal(t, stored.WebhookURL, labels["url"])
	})

	t.Run("resyncs the existing channel URL and password instead of creating a second channel", func(t *testing.T) {
		postCalled := false
		var patchURL string
		var patchBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				postCalled = true
				return []byte(`{"name":"x"}`), nil
			},
			patchFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				patchURL = url
				patchBody, _ = body.(map[string]any)
				return []byte(`{}`), nil
			},
		}
		withFactory(mc)

		meta := &contexts.MetadataContext{Metadata: OnAlertMetadata{
			NotificationChannel: "projects/my-project/notificationChannels/existing",
			WebhookURL:          "https://old.example/api/v1/webhooks/abc",
		}}
		err := tr.Setup(core.TriggerContext{
			Configuration: map[string]any{"states": []string{"open"}},
			Integration:   &contexts.IntegrationContext{},
			Webhook:       &contexts.NodeWebhookContext{Secret: "rotated-secret"},
			Metadata:      meta,
		})

		require.NoError(t, err)
		assert.False(t, postCalled, "should not create a second channel")
		// The existing channel is patched (URL + Basic-auth credentials), so a
		// moved URL or a rotated secret keeps signed deliveries working.
		assert.Contains(t, patchURL, "/notificationChannels/existing?updateMask=labels.url")
		assert.Contains(t, patchURL, "labels.password")
		labels := patchBody["labels"].(map[string]any)
		assert.Equal(t, webhookAuthUsername, labels["username"])
		assert.Equal(t, "rotated-secret", labels["password"])
		stored := OnAlertMetadata{}
		require.NoError(t, mapstructure.Decode(meta.Get(), &stored))
		assert.Equal(t, stored.WebhookURL, labels["url"])
	})

	t.Run("requires a connected GCP integration", func(t *testing.T) {
		withFactory(&mockClient{projectID: "my-project"})
		err := tr.Setup(core.TriggerContext{
			Configuration: map[string]any{"states": []string{"open"}},
			Integration:   nil,
			Webhook:       &contexts.NodeWebhookContext{},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "GCP integration is required")
	})

	t.Run("rejects an invalid state", func(t *testing.T) {
		withFactory(&mockClient{projectID: "my-project"})
		err := tr.Setup(core.TriggerContext{
			Configuration: map[string]any{"states": []string{"sideways"}},
			Integration:   &contexts.IntegrationContext{},
			Webhook:       &contexts.NodeWebhookContext{},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "invalid state")
	})
}

func Test__parseOnAlertConfiguration(t *testing.T) {
	t.Run("defaults to open when states is missing", func(t *testing.T) {
		cfg, err := parseOnAlertConfiguration(map[string]any{})
		require.NoError(t, err)
		assert.Equal(t, []string{incidentStateOpen}, cfg.States)
	})

	t.Run("defaults to open when states is empty", func(t *testing.T) {
		cfg, err := parseOnAlertConfiguration(map[string]any{"states": []string{}})
		require.NoError(t, err)
		assert.Equal(t, []string{incidentStateOpen}, cfg.States)
	})

	t.Run("normalizes and dedupes provided states", func(t *testing.T) {
		cfg, err := parseOnAlertConfiguration(map[string]any{"states": []string{"OPEN", "open", "Closed"}})
		require.NoError(t, err)
		assert.Equal(t, []string{incidentStateOpen, incidentStateClosed}, cfg.States)
	})

	t.Run("rejects an invalid state", func(t *testing.T) {
		_, err := parseOnAlertConfiguration(map[string]any{"states": []string{"bogus"}})
		require.ErrorContains(t, err, "invalid state")
	})
}

func Test__OnAlert__HandleWebhook(t *testing.T) {
	tr := &OnAlert{}
	openIncident := []byte(`{"version":"1.2","incident":{"incident_id":"0.abc","state":"open","policy_name":"projects/my-project/alertPolicies/1","condition_name":"High CPU","summary":"CPU high","url":"https://console","resource_name":"my-vm","observed_value":"0.93","threshold_value":"0.8","metric":{"type":"compute.googleapis.com/instance/cpu/utilization","displayName":"CPU utilization"}}}`)
	closedIncident := []byte(`{"version":"1.2","incident":{"incident_id":"0.abc","state":"closed","policy_name":"projects/my-project/alertPolicies/1"}}`)

	t.Run("emits an open incident when open is selected", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := tr.HandleWebhook(core.WebhookRequestContext{
			Configuration: map[string]any{"states": []string{"open"}},
			Body:          openIncident,
			Events:        events,
		})
		require.NoError(t, err)
		assert.Equal(t, 200, code)
		require.Len(t, events.Payloads, 1)
		assert.Equal(t, "gcp.monitoring.alert", events.Payloads[0].Type)
		data := events.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "0.abc", data["incidentId"])
		assert.Equal(t, "open", data["state"])
		assert.Equal(t, "compute.googleapis.com/instance/cpu/utilization", data["metricType"])
	})

	t.Run("handles a null ended_at on an open incident and omits it from the payload", func(t *testing.T) {
		// Cloud Monitoring sends "ended_at": null while an incident is still open.
		// This must parse and emit, not fail the whole body.
		body := []byte(`{"version":"1.2","incident":{"incident_id":"0.def","state":"open","policy_name":"projects/my-project/alertPolicies/1","started_at":1767225600,"ended_at":null}}`)
		events := &contexts.EventContext{}
		code, _, err := tr.HandleWebhook(core.WebhookRequestContext{
			Configuration: map[string]any{"states": []string{"open"}},
			Body:          body,
			Events:        events,
		})
		require.NoError(t, err)
		assert.Equal(t, 200, code)
		require.Len(t, events.Payloads, 1)
		data := events.Payloads[0].Data.(map[string]any)
		assert.Equal(t, int64(1767225600), data["startedAt"])
		_, hasEnded := data["endedAt"]
		assert.False(t, hasEnded, "open incident should omit endedAt")
	})

	t.Run("includes ended_at once the incident has resolved", func(t *testing.T) {
		body := []byte(`{"version":"1.2","incident":{"incident_id":"0.ghi","state":"closed","policy_name":"projects/my-project/alertPolicies/1","started_at":1767225600,"ended_at":1767229200}}`)
		events := &contexts.EventContext{}
		code, _, err := tr.HandleWebhook(core.WebhookRequestContext{
			Configuration: map[string]any{"states": []string{"closed"}},
			Body:          body,
			Events:        events,
		})
		require.NoError(t, err)
		assert.Equal(t, 200, code)
		require.Len(t, events.Payloads, 1)
		data := events.Payloads[0].Data.(map[string]any)
		assert.Equal(t, int64(1767229200), data["endedAt"])
	})

	t.Run("ignores a closed incident when only open is selected", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := tr.HandleWebhook(core.WebhookRequestContext{
			Configuration: map[string]any{"states": []string{"open"}},
			Body:          closedIncident,
			Events:        events,
		})
		require.NoError(t, err)
		assert.Equal(t, 200, code)
		assert.Empty(t, events.Payloads)
	})

	t.Run("emits a closed incident when closed is selected", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := tr.HandleWebhook(core.WebhookRequestContext{
			Configuration: map[string]any{"states": []string{"open", "closed"}},
			Body:          closedIncident,
			Events:        events,
		})
		require.NoError(t, err)
		assert.Equal(t, 200, code)
		require.Len(t, events.Payloads, 1)
		assert.Equal(t, "closed", events.Payloads[0].Data.(map[string]any)["state"])
	})

	t.Run("acknowledges a verification ping with no incident", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := tr.HandleWebhook(core.WebhookRequestContext{
			Configuration: map[string]any{"states": []string{"open"}},
			Body:          []byte(`{"version":"1.2"}`),
			Events:        events,
		})
		require.NoError(t, err)
		assert.Equal(t, 200, code)
		assert.Empty(t, events.Payloads)
	})

	t.Run("accepts a request signed with the node's webhook secret", func(t *testing.T) {
		const secret = "test-webhook-secret"
		headers := http.Header{}
		headers.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(webhookAuthUsername+":"+secret)))

		events := &contexts.EventContext{}
		code, _, err := tr.HandleWebhook(core.WebhookRequestContext{
			Configuration: map[string]any{"states": []string{"open"}},
			Body:          openIncident,
			Headers:       headers,
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        events,
		})
		require.NoError(t, err)
		assert.Equal(t, 200, code)
		require.Len(t, events.Payloads, 1)
	})

	t.Run("rejects a request with an invalid Basic-auth secret", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(webhookAuthUsername+":wrong-secret")))

		events := &contexts.EventContext{}
		code, _, err := tr.HandleWebhook(core.WebhookRequestContext{
			Configuration: map[string]any{"states": []string{"open"}},
			Body:          openIncident,
			Headers:       headers,
			Webhook:       &contexts.NodeWebhookContext{Secret: "test-webhook-secret"},
			Events:        events,
		})
		require.Error(t, err)
		assert.Equal(t, http.StatusUnauthorized, code)
		assert.Empty(t, events.Payloads)
	})

	t.Run("rejects a request that is missing credentials", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := tr.HandleWebhook(core.WebhookRequestContext{
			Configuration: map[string]any{"states": []string{"open"}},
			Body:          openIncident,
			Headers:       http.Header{},
			Webhook:       &contexts.NodeWebhookContext{Secret: "test-webhook-secret"},
			Events:        events,
		})
		require.Error(t, err)
		assert.Equal(t, http.StatusUnauthorized, code)
		assert.Empty(t, events.Payloads)
	})

	t.Run("fails closed when the webhook secret cannot be read", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(webhookAuthUsername+":test-webhook-secret")))

		events := &contexts.EventContext{}
		code, _, err := tr.HandleWebhook(core.WebhookRequestContext{
			Configuration: map[string]any{"states": []string{"open"}},
			Body:          openIncident,
			Headers:       headers,
			// A transient decrypt/lookup error must not be treated as "no secret".
			Webhook: secretErrorWebhookContext{&contexts.NodeWebhookContext{}},
			Events:  events,
		})
		require.Error(t, err)
		assert.Equal(t, http.StatusUnauthorized, code)
		assert.Empty(t, events.Payloads)
	})
}

// secretErrorWebhookContext simulates a transient failure when reading the node
// webhook secret (e.g. a decrypt or database error), to verify auth fails closed.
type secretErrorWebhookContext struct {
	*contexts.NodeWebhookContext
}

func (secretErrorWebhookContext) GetSecret() ([]byte, error) {
	return nil, errors.New("decrypt failed")
}

func Test__OnAlert__Cleanup(t *testing.T) {
	tr := &OnAlert{}

	t.Run("deletes the notification channel", func(t *testing.T) {
		var deleteURL string
		mc := &mockClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, url string) ([]byte, error) {
				deleteURL = url
				return []byte(`{}`), nil
			},
		}
		withFactory(mc)

		err := tr.Cleanup(core.TriggerContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{Metadata: OnAlertMetadata{NotificationChannel: "projects/my-project/notificationChannels/123"}},
		})
		require.NoError(t, err)
		assert.Contains(t, deleteURL, "/v3/projects/my-project/notificationChannels/123")
	})

	t.Run("no-op when no channel was created", func(t *testing.T) {
		withFactory(&mockClient{projectID: "my-project"})
		err := tr.Cleanup(core.TriggerContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("does not block node removal when the integration is gone", func(t *testing.T) {
		deleteCalled := false
		withFactory(&mockClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, url string) ([]byte, error) {
				deleteCalled = true
				return []byte(`{}`), nil
			},
		})
		err := tr.Cleanup(core.TriggerContext{
			Integration: nil,
			Metadata:    &contexts.MetadataContext{Metadata: OnAlertMetadata{NotificationChannel: "projects/my-project/notificationChannels/123"}},
		})
		require.NoError(t, err)
		assert.False(t, deleteCalled, "cannot call Cloud Monitoring without an integration")
	})
}
