package monitoring

import (
	"context"
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
				return []byte(`{"name":"projects/my-project/notificationChannels/123","type":"webhook_tokenauth"}`), nil
			},
		}
		withFactory(mc)

		meta := &contexts.MetadataContext{}
		err := tr.Setup(core.TriggerContext{
			Configuration: map[string]any{"states": []string{"open"}},
			Integration:   &contexts.IntegrationContext{},
			Webhook:       &contexts.NodeWebhookContext{},
			Metadata:      meta,
		})

		require.NoError(t, err)
		assert.Contains(t, postURL, "/v3/projects/my-project/notificationChannels")
		assert.Equal(t, "webhook_tokenauth", postBody["type"])

		stored := OnAlertMetadata{}
		require.NoError(t, mapstructure.Decode(meta.Get(), &stored))
		assert.Equal(t, "projects/my-project/notificationChannels/123", stored.NotificationChannel)
		assert.NotEmpty(t, stored.WebhookURL)
		// The channel points at the node's webhook URL.
		assert.Equal(t, stored.WebhookURL, postBody["labels"].(map[string]any)["url"])
	})

	t.Run("does not recreate the channel when metadata already has one", func(t *testing.T) {
		called := false
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				called = true
				return []byte(`{"name":"x"}`), nil
			},
		}
		withFactory(mc)

		meta := &contexts.MetadataContext{Metadata: OnAlertMetadata{NotificationChannel: "projects/my-project/notificationChannels/existing"}}
		err := tr.Setup(core.TriggerContext{
			Configuration: map[string]any{"states": []string{"open"}},
			Integration:   &contexts.IntegrationContext{},
			Webhook:       &contexts.NodeWebhookContext{},
			Metadata:      meta,
		})

		require.NoError(t, err)
		assert.False(t, called, "should not create a second channel")
	})

	t.Run("invalid states -> error", func(t *testing.T) {
		withFactory(&mockClient{projectID: "my-project"})
		err := tr.Setup(core.TriggerContext{
			Configuration: map[string]any{"states": []string{}},
			Integration:   &contexts.IntegrationContext{},
			Webhook:       &contexts.NodeWebhookContext{},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "at least one state")
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
}
