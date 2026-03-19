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

func TestOnBlobDeleted_Metadata(t *testing.T) {
	trigger := &OnBlobDeleted{}

	assert.Equal(t, "azure.onBlobDeleted", trigger.Name())
	assert.Equal(t, "On Blob Deleted", trigger.Label())
	assert.Equal(t, "azure", trigger.Icon())
	assert.Equal(t, "blue", trigger.Color())
	assert.NotEmpty(t, trigger.Description())
	assert.NotEmpty(t, trigger.Documentation())
}

func TestOnBlobDeleted_Configuration(t *testing.T) {
	trigger := &OnBlobDeleted{}
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

func TestOnBlobDeleted_ExampleData(t *testing.T) {
	trigger := &OnBlobDeleted{}
	example := trigger.ExampleData()

	require.NotNil(t, example)
	assert.Equal(t, "Microsoft.Storage.BlobDeleted", example["eventType"])
	assert.Contains(t, example, "subject")
	assert.Contains(t, example, "data")
}

func TestOnBlobDeleted_Setup(t *testing.T) {
	trigger := &OnBlobDeleted{}

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

func TestOnBlobDeleted_Actions(t *testing.T) {
	trigger := &OnBlobDeleted{}
	assert.Empty(t, trigger.Actions())
}

func TestOnBlobDeleted_Cleanup(t *testing.T) {
	trigger := &OnBlobDeleted{}
	ctx := core.TriggerContext{
		Logger:        logrus.NewEntry(logrus.New()),
		Configuration: map[string]any{},
	}
	assert.NoError(t, trigger.Cleanup(ctx))
}

func TestOnBlobDeleted_HandleWebhook_SubscriptionValidation(t *testing.T) {
	trigger := &OnBlobDeleted{}

	validationCode := "test-validation-code-99999"
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

	ctx := core.WebhookRequestContext{
		Body:          body,
		Headers:       http.Header{},
		Configuration: map[string]any{},
		Webhook:       &contexts.NodeWebhookContext{},
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

func TestOnBlobDeleted_HandleWebhook_BlobDeleted(t *testing.T) {
	trigger := &OnBlobDeleted{}

	t.Run("emits event with no filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "blob-del-event-1",
				Subject:   "/blobServices/default/containers/mycontainer/blobs/data/myfile.csv",
				EventType: EventTypeBlobDeleted,
				EventTime: time.Now(),
				Data: map[string]any{
					"api":      "DeleteBlob",
					"blobType": "BlockBlob",
					"url":      "https://mystorageaccount.blob.core.windows.net/mycontainer/data/myfile.csv",
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
		assert.Equal(t, "azure.blob.deleted", eventsCtx.Payloads[0].Type)

		payload, ok := eventsCtx.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "blob-del-event-1", payload["id"])
		assert.Equal(t, EventTypeBlobDeleted, payload["eventType"])
	})

	t.Run("skips event with non-matching container filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "blob-del-event-2",
				Subject:   "/blobServices/default/containers/archive/blobs/old.txt",
				EventType: EventTypeBlobDeleted,
				EventTime: time.Now(),
				Data:      map[string]any{"api": "DeleteBlob"},
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

	t.Run("skips event with non-matching blob filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "blob-del-event-3",
				Subject:   "/blobServices/default/containers/data/blobs/image.png",
				EventType: EventTypeBlobDeleted,
				EventTime: time.Now(),
				Data:      map[string]any{"api": "DeleteBlob"},
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

func TestOnBlobDeleted_HandleWebhook_InvalidJSON(t *testing.T) {
	trigger := &OnBlobDeleted{}

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
