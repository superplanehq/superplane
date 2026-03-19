package azure

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestOnBlobCreated_Metadata(t *testing.T) {
	trigger := &OnBlobCreated{}

	assert.Equal(t, "azure.onBlobCreated", trigger.Name())
	assert.Equal(t, "On Blob Created", trigger.Label())
	assert.Equal(t, "azure", trigger.Icon())
	assert.Equal(t, "blue", trigger.Color())
	assert.NotEmpty(t, trigger.Description())
	assert.NotEmpty(t, trigger.Documentation())
}

func TestOnBlobCreated_Configuration(t *testing.T) {
	trigger := &OnBlobCreated{}
	config := trigger.Configuration()

	require.Len(t, config, 4)
	assert.Equal(t, "resourceGroup", config[0].Name)
	assert.True(t, config[0].Required)
	assert.Equal(t, "storageAccount", config[1].Name)
	assert.True(t, config[1].Required)
	assert.Equal(t, "containerFilter", config[2].Name)
	assert.False(t, config[2].Required)
	assert.Equal(t, "blobFilter", config[3].Name)
	assert.False(t, config[3].Required)
}

func TestOnBlobCreated_ExampleData(t *testing.T) {
	trigger := &OnBlobCreated{}
	example := trigger.ExampleData()

	require.NotNil(t, example)
	assert.Equal(t, "Microsoft.Storage.BlobCreated", example["eventType"])
	assert.Contains(t, example, "subject")
	assert.Contains(t, example, "data")
}

func TestOnBlobCreated_Setup(t *testing.T) {
	trigger := &OnBlobCreated{}

	t.Run("fails without storage account", func(t *testing.T) {
		ctx := core.TriggerContext{
			Logger:        logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{},
			Integration:   &contexts.IntegrationContext{},
		}
		err := trigger.Setup(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "storageAccount is required")
	})

	t.Run("succeeds with storage account", func(t *testing.T) {
		ctx := core.TriggerContext{
			Logger: logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"storageAccount": "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/mystorageaccount",
			},
			Integration: &contexts.IntegrationContext{},
		}
		err := trigger.Setup(ctx)
		assert.NoError(t, err)
	})
}

func TestOnBlobCreated_Actions(t *testing.T) {
	trigger := &OnBlobCreated{}
	assert.Empty(t, trigger.Actions())
}

func TestOnBlobCreated_Cleanup(t *testing.T) {
	trigger := &OnBlobCreated{}
	ctx := core.TriggerContext{
		Logger:        logrus.NewEntry(logrus.New()),
		Configuration: map[string]any{},
	}
	assert.NoError(t, trigger.Cleanup(ctx))
}

func TestOnBlobCreated_HandleWebhook_SubscriptionValidation(t *testing.T) {
	trigger := &OnBlobCreated{}

	validationCode := "test-validation-code-12345"
	events := []EventGridEvent{
		{
			ID:              "validation-event-1",
			Subject:         "",
			EventType:       EventTypeSubscriptionValidation,
			EventTime:       time.Now(),
			DataVersion:     "1.0",
			MetadataVersion: "1",
			Data: map[string]any{
				"validationCode": validationCode,
			},
		},
	}

	body, err := json.Marshal(events)
	require.NoError(t, err)

	eventsCtx := &contexts.EventContext{}
	webhookCtx := &contexts.NodeWebhookContext{}

	ctx := core.WebhookRequestContext{
		Body:          body,
		Headers:       http.Header{},
		Configuration: map[string]any{},
		Webhook:       webhookCtx,
		Events:        eventsCtx,
		Logger:        logrus.NewEntry(logrus.New()),
	}

	code, resp, err := trigger.HandleWebhook(ctx)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	require.NotNil(t, resp)
	assert.Contains(t, string(resp.Body), validationCode)
	assert.Equal(t, 0, eventsCtx.Count())
}

func TestOnBlobCreated_HandleWebhook_BlobCreated(t *testing.T) {
	trigger := &OnBlobCreated{}

	t.Run("emits event with no filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "blob-event-1",
				Subject:   "/blobServices/default/containers/mycontainer/blobs/data/myfile.csv",
				EventType: EventTypeBlobCreated,
				EventTime: time.Now(),
				Data: map[string]any{
					"api":         "PutBlob",
					"contentType": "text/csv",
					"blobType":    "BlockBlob",
					"url":         "https://mystorageaccount.blob.core.windows.net/mycontainer/data/myfile.csv",
				},
				DataVersion:     "",
				MetadataVersion: "1",
			},
		}

		body, err := json.Marshal(events)
		require.NoError(t, err)

		eventsCtx := &contexts.EventContext{}

		ctx := core.WebhookRequestContext{
			Body:          body,
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Webhook:       &contexts.NodeWebhookContext{},
			Events:        eventsCtx,
			Logger:        logrus.NewEntry(logrus.New()),
		}

		code, _, err := trigger.HandleWebhook(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, eventsCtx.Count())
		assert.Equal(t, "azure.blob.created", eventsCtx.Payloads[0].Type)

		payload, ok := eventsCtx.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "blob-event-1", payload["id"])
		assert.Equal(t, EventTypeBlobCreated, payload["eventType"])
		assert.Equal(t, "/blobServices/default/containers/mycontainer/blobs/data/myfile.csv", payload["subject"])
	})

	t.Run("emits event with matching container filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "blob-event-2",
				Subject:   "/blobServices/default/containers/uploads/blobs/report.pdf",
				EventType: EventTypeBlobCreated,
				EventTime: time.Now(),
				Data:      map[string]any{"api": "PutBlob"},
			},
		}

		body, err := json.Marshal(events)
		require.NoError(t, err)

		eventsCtx := &contexts.EventContext{}

		ctx := core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Configuration: map[string]any{
				"containerFilter": "uploads",
			},
			Webhook: &contexts.NodeWebhookContext{},
			Events:  eventsCtx,
			Logger:  logrus.NewEntry(logrus.New()),
		}

		code, _, err := trigger.HandleWebhook(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 1, eventsCtx.Count())
	})

	t.Run("skips event with non-matching container filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "blob-event-3",
				Subject:   "/blobServices/default/containers/logs/blobs/app.log",
				EventType: EventTypeBlobCreated,
				EventTime: time.Now(),
				Data:      map[string]any{"api": "PutBlob"},
			},
		}

		body, err := json.Marshal(events)
		require.NoError(t, err)

		eventsCtx := &contexts.EventContext{}

		ctx := core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Configuration: map[string]any{
				"containerFilter": "uploads",
			},
			Webhook: &contexts.NodeWebhookContext{},
			Events:  eventsCtx,
			Logger:  logrus.NewEntry(logrus.New()),
		}

		code, _, err := trigger.HandleWebhook(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, eventsCtx.Count())
	})

	t.Run("emits event with matching blob filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "blob-event-4",
				Subject:   "/blobServices/default/containers/data/blobs/2026/report.csv",
				EventType: EventTypeBlobCreated,
				EventTime: time.Now(),
				Data:      map[string]any{"api": "PutBlob"},
			},
		}

		body, err := json.Marshal(events)
		require.NoError(t, err)

		eventsCtx := &contexts.EventContext{}

		ctx := core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Configuration: map[string]any{
				"blobFilter": `.*\.csv`,
			},
			Webhook: &contexts.NodeWebhookContext{},
			Events:  eventsCtx,
			Logger:  logrus.NewEntry(logrus.New()),
		}

		code, _, err := trigger.HandleWebhook(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 1, eventsCtx.Count())
	})

	t.Run("skips event with non-matching blob filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "blob-event-5",
				Subject:   "/blobServices/default/containers/data/blobs/image.png",
				EventType: EventTypeBlobCreated,
				EventTime: time.Now(),
				Data:      map[string]any{"api": "PutBlob"},
			},
		}

		body, err := json.Marshal(events)
		require.NoError(t, err)

		eventsCtx := &contexts.EventContext{}

		ctx := core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Configuration: map[string]any{
				"blobFilter": `.*\.csv`,
			},
			Webhook: &contexts.NodeWebhookContext{},
			Events:  eventsCtx,
			Logger:  logrus.NewEntry(logrus.New()),
		}

		code, _, err := trigger.HandleWebhook(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, eventsCtx.Count())
	})
}

func TestOnBlobCreated_HandleWebhook_InvalidJSON(t *testing.T) {
	trigger := &OnBlobCreated{}

	ctx := core.WebhookRequestContext{
		Body:          []byte("invalid json"),
		Headers:       http.Header{},
		Configuration: map[string]any{},
		Webhook:       &contexts.NodeWebhookContext{},
		Events:        &contexts.EventContext{},
		Logger:        logrus.NewEntry(logrus.New()),
	}

	code, _, err := trigger.HandleWebhook(ctx)
	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, code)
}
