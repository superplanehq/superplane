package azure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// TestOnVMStarted_Metadata verifies the trigger's metadata methods
func TestOnVMStarted_Metadata(t *testing.T) {
	trigger := &OnVMStarted{}

	assert.Equal(t, "azure.onVirtualMachineStarted", trigger.Name())
	assert.Equal(t, "On VM Started", trigger.Label())
	assert.Equal(t, "azure", trigger.Icon())
	assert.Equal(t, "blue", trigger.Color())
	assert.NotEmpty(t, trigger.Description())
	assert.NotEmpty(t, trigger.Documentation())
}

// TestOnVMStarted_Configuration verifies the trigger's configuration fields
func TestOnVMStarted_Configuration(t *testing.T) {
	trigger := &OnVMStarted{}
	config := trigger.Configuration()

	require.Len(t, config, 2)
	assert.Equal(t, "resourceGroup", config[0].Name)
	assert.Equal(t, "Resource Group", config[0].Label)
	assert.False(t, config[0].Required)
	assert.Equal(t, "nameFilter", config[1].Name)
	assert.Equal(t, "VM Name Filter", config[1].Label)
	assert.False(t, config[1].Required)
}

// TestOnVMStarted_ExampleData verifies the trigger's example output
func TestOnVMStarted_ExampleData(t *testing.T) {
	trigger := &OnVMStarted{}
	example := trigger.ExampleData()

	require.NotNil(t, example)
	assert.Contains(t, example, "id")
	assert.Contains(t, example, "eventType")
	assert.Equal(t, "Microsoft.Resources.ResourceActionSuccess", example["eventType"])
	assert.Contains(t, example, "subject")
	assert.Contains(t, example, "data")

	data, ok := example["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Microsoft.Compute/virtualMachines/start/action", data["operationName"])
}

// TestOnVMStarted_Setup verifies the trigger setup method
func TestOnVMStarted_Setup(t *testing.T) {
	trigger := &OnVMStarted{}

	t.Run("setup with no resource group filter", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		logger := logrus.NewEntry(logrus.New())

		ctx := core.TriggerContext{
			Logger:        logger,
			Configuration: map[string]any{},
			Metadata:      metadataCtx,
			Integration:   &contexts.IntegrationContext{},
		}

		err := trigger.Setup(ctx)
		assert.NoError(t, err)
	})

	t.Run("setup with resource group filter", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		logger := logrus.NewEntry(logrus.New())

		ctx := core.TriggerContext{
			Logger: logger,
			Configuration: map[string]any{
				"resourceGroup": "my-rg",
			},
			Metadata:    metadataCtx,
			Integration: &contexts.IntegrationContext{},
		}

		err := trigger.Setup(ctx)
		assert.NoError(t, err)
	})
}

// TestOnVMStarted_Cleanup verifies the trigger cleanup method
func TestOnVMStarted_Cleanup(t *testing.T) {
	trigger := &OnVMStarted{}
	metadataCtx := &contexts.MetadataContext{}
	logger := logrus.NewEntry(logrus.New())

	ctx := core.TriggerContext{
		Logger:        logger,
		Configuration: map[string]any{},
		Metadata:      metadataCtx,
	}

	err := trigger.Cleanup(ctx)
	assert.NoError(t, err)
}

// TestOnVMStarted_Actions verifies the trigger has no actions
func TestOnVMStarted_Actions(t *testing.T) {
	trigger := &OnVMStarted{}
	actions := trigger.Actions()
	assert.Empty(t, actions)
}

// TestOnVMStarted_HandleAction verifies the trigger's action handler
func TestOnVMStarted_HandleAction(t *testing.T) {
	trigger := &OnVMStarted{}
	logger := logrus.NewEntry(logrus.New())

	ctx := core.TriggerActionContext{
		Name:          "test",
		Parameters:    map[string]any{},
		Configuration: map[string]any{},
		Logger:        logger,
	}

	result, err := trigger.HandleAction(ctx)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// TestOnVMStarted_HandleWebhook_SubscriptionValidation verifies subscription validation handling
func TestOnVMStarted_HandleWebhook_SubscriptionValidation(t *testing.T) {
	trigger := &OnVMStarted{}

	// Create a test server to serve as the validation URL
	validationServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer validationServer.Close()

	validationCode := "test-validation-code-12345"
	events := []EventGridEvent{
		{
			ID:              "validation-event-1",
			Topic:           "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:         "",
			EventType:       EventTypeSubscriptionValidation,
			EventTime:       time.Now(),
			DataVersion:     "1.0",
			MetadataVersion: "1",
			Data: map[string]any{
				"validationCode": validationCode,
				"validationUrl":  validationServer.URL,
			},
		},
	}

	body, err := json.Marshal(events)
	require.NoError(t, err)

	eventsCtx := &contexts.EventContext{}
	webhookCtx := &contexts.NodeWebhookContext{}
	logger := logrus.NewEntry(logrus.New())

	ctx := core.WebhookRequestContext{
		Body:          body,
		Headers:       http.Header{},
		Configuration: map[string]any{},
		Webhook:       webhookCtx,
		Events:        eventsCtx,
		Logger:        logger,
	}

	code, _, err := trigger.HandleWebhook(ctx)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	// Subscription validation should not emit any events to the workflow
	assert.Equal(t, 0, eventsCtx.Count())
}

// TestOnVMStarted_HandleWebhook_VMStartSuccess verifies VM start event handling
func TestOnVMStarted_HandleWebhook_VMStartSuccess(t *testing.T) {
	trigger := &OnVMStarted{}

	t.Run("VM started with no filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "vm-event-1",
				Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
				Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm/start",
				EventType: EventTypeResourceActionSuccess,
				EventTime: time.Now(),
				Data: map[string]any{
					"subscriptionId": "test-sub",
					"operationName":  "Microsoft.Compute/virtualMachines/start/action",
					"status":         "Succeeded",
				},
				DataVersion:     "1.0",
				MetadataVersion: "1",
			},
		}

		body, err := json.Marshal(events)
		require.NoError(t, err)

		eventsCtx := &contexts.EventContext{}
		webhookCtx := &contexts.NodeWebhookContext{}
		logger := logrus.NewEntry(logrus.New())

		ctx := core.WebhookRequestContext{
			Body:          body,
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Webhook:       webhookCtx,
			Events:        eventsCtx,
			Logger:        logger,
		}

		code, _, err := trigger.HandleWebhook(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)

		require.Equal(t, 1, eventsCtx.Count())
		assert.Equal(t, "azure.vm.started", eventsCtx.Payloads[0].Type)

		// The emitted payload is the full, raw Event Grid event
		payload, ok := eventsCtx.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "vm-event-1", payload["id"])
		assert.Equal(t, EventTypeResourceActionSuccess, payload["eventType"])
		assert.Equal(t, "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm/start", payload["subject"])

		data, ok := payload["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Succeeded", data["status"])
		assert.Equal(t, "Microsoft.Compute/virtualMachines/start/action", data["operationName"])
	})

	t.Run("VM started with matching resource group filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "vm-event-2",
				Topic:     "/subscriptions/test-sub/resourceGroups/my-target-rg",
				Subject:   "/subscriptions/test-sub/resourceGroups/my-target-rg/providers/Microsoft.Compute/virtualMachines/test-vm-2/start",
				EventType: EventTypeResourceActionSuccess,
				EventTime: time.Now(),
				Data: map[string]any{
					"subscriptionId": "test-sub",
					"operationName":  "Microsoft.Compute/virtualMachines/start/action",
					"status":         "Succeeded",
				},
				DataVersion:     "1.0",
				MetadataVersion: "1",
			},
		}

		body, err := json.Marshal(events)
		require.NoError(t, err)

		eventsCtx := &contexts.EventContext{}
		webhookCtx := &contexts.NodeWebhookContext{}
		logger := logrus.NewEntry(logrus.New())

		ctx := core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Configuration: map[string]any{
				"resourceGroup": "my-target-rg",
			},
			Webhook: webhookCtx,
			Events:  eventsCtx,
			Logger:  logger,
		}

		code, _, err := trigger.HandleWebhook(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)

		require.Equal(t, 1, eventsCtx.Count())
		assert.Equal(t, "azure.vm.started", eventsCtx.Payloads[0].Type)

		payload, ok := eventsCtx.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Contains(t, payload["subject"], "my-target-rg")
	})
}

// TestOnVMStarted_HandleWebhook_FilterMismatch verifies resource group filtering
func TestOnVMStarted_HandleWebhook_FilterMismatch(t *testing.T) {
	trigger := &OnVMStarted{}

	events := []EventGridEvent{
		{
			ID:        "vm-event-3",
			Topic:     "/subscriptions/test-sub/resourceGroups/rg-other",
			Subject:   "/subscriptions/test-sub/resourceGroups/rg-other/providers/Microsoft.Compute/virtualMachines/test-vm-other/start",
			EventType: EventTypeResourceActionSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"subscriptionId": "test-sub",
				"operationName":  "Microsoft.Compute/virtualMachines/start/action",
				"status":         "Succeeded",
			},
			DataVersion:     "1.0",
			MetadataVersion: "1",
		},
	}

	body, err := json.Marshal(events)
	require.NoError(t, err)

	eventsCtx := &contexts.EventContext{}
	webhookCtx := &contexts.NodeWebhookContext{}
	logger := logrus.NewEntry(logrus.New())

	ctx := core.WebhookRequestContext{
		Body:    body,
		Headers: http.Header{},
		Configuration: map[string]any{
			"resourceGroup": "rg-target",
		},
		Webhook: webhookCtx,
		Events:  eventsCtx,
		Logger:  logger,
	}

	code, _, err := trigger.HandleWebhook(ctx)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	assert.Equal(t, 0, eventsCtx.Count())
}

// TestOnVMStarted_HandleWebhook_NameFilter verifies VM name regex filtering
func TestOnVMStarted_HandleWebhook_NameFilter(t *testing.T) {
	trigger := &OnVMStarted{}

	t.Run("matching name filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "vm-event-name-1",
				Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
				Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/prod-web-01/start",
				EventType: EventTypeResourceActionSuccess,
				EventTime: time.Now(),
				Data: map[string]any{
					"subscriptionId": "test-sub",
					"operationName":  "Microsoft.Compute/virtualMachines/start/action",
					"status":         "Succeeded",
				},
				DataVersion:     "1.0",
				MetadataVersion: "1",
			},
		}

		body, err := json.Marshal(events)
		require.NoError(t, err)

		eventsCtx := &contexts.EventContext{}
		webhookCtx := &contexts.NodeWebhookContext{}
		logger := logrus.NewEntry(logrus.New())

		ctx := core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Configuration: map[string]any{
				"nameFilter": "prod-.*",
			},
			Webhook: webhookCtx,
			Events:  eventsCtx,
			Logger:  logger,
		}

		code, _, err := trigger.HandleWebhook(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)

		require.Equal(t, 1, eventsCtx.Count())
		payload, ok := eventsCtx.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Contains(t, payload["subject"], "prod-web-01")
	})

	t.Run("non-matching name filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "vm-event-name-2",
				Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
				Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/dev-web-01/start",
				EventType: EventTypeResourceActionSuccess,
				EventTime: time.Now(),
				Data: map[string]any{
					"subscriptionId": "test-sub",
					"operationName":  "Microsoft.Compute/virtualMachines/start/action",
					"status":         "Succeeded",
				},
				DataVersion:     "1.0",
				MetadataVersion: "1",
			},
		}

		body, err := json.Marshal(events)
		require.NoError(t, err)

		eventsCtx := &contexts.EventContext{}
		webhookCtx := &contexts.NodeWebhookContext{}
		logger := logrus.NewEntry(logrus.New())

		ctx := core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Configuration: map[string]any{
				"nameFilter": "prod-.*",
			},
			Webhook: webhookCtx,
			Events:  eventsCtx,
			Logger:  logger,
		}

		code, _, err := trigger.HandleWebhook(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)

		assert.Equal(t, 0, eventsCtx.Count())
	})

	t.Run("empty name filter triggers for all", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "vm-event-name-3",
				Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
				Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/any-vm/start",
				EventType: EventTypeResourceActionSuccess,
				EventTime: time.Now(),
				Data: map[string]any{
					"subscriptionId": "test-sub",
					"operationName":  "Microsoft.Compute/virtualMachines/start/action",
					"status":         "Succeeded",
				},
				DataVersion:     "1.0",
				MetadataVersion: "1",
			},
		}

		body, err := json.Marshal(events)
		require.NoError(t, err)

		eventsCtx := &contexts.EventContext{}
		webhookCtx := &contexts.NodeWebhookContext{}
		logger := logrus.NewEntry(logrus.New())

		ctx := core.WebhookRequestContext{
			Body:          body,
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Webhook:       webhookCtx,
			Events:        eventsCtx,
			Logger:        logger,
		}

		code, _, err := trigger.HandleWebhook(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)

		require.Equal(t, 1, eventsCtx.Count())
	})
}

// TestOnVMStarted_HandleWebhook_NonVMResource verifies non-VM resource filtering
func TestOnVMStarted_HandleWebhook_NonVMResource(t *testing.T) {
	trigger := &OnVMStarted{}

	events := []EventGridEvent{
		{
			ID:        "other-action-1",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorage/listKeys",
			EventType: EventTypeResourceActionSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"subscriptionId": "test-sub",
				"operationName":  "Microsoft.Storage/storageAccounts/listKeys/action",
				"status":         "Succeeded",
			},
			DataVersion:     "1.0",
			MetadataVersion: "1",
		},
	}

	body, err := json.Marshal(events)
	require.NoError(t, err)

	eventsCtx := &contexts.EventContext{}
	webhookCtx := &contexts.NodeWebhookContext{}
	logger := logrus.NewEntry(logrus.New())

	ctx := core.WebhookRequestContext{
		Body:          body,
		Headers:       http.Header{},
		Configuration: map[string]any{},
		Webhook:       webhookCtx,
		Events:        eventsCtx,
		Logger:        logger,
	}

	code, _, err := trigger.HandleWebhook(ctx)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	assert.Equal(t, 0, eventsCtx.Count())
}

// TestOnVMStarted_HandleWebhook_NonStartAction verifies that non-start VM actions are ignored
func TestOnVMStarted_HandleWebhook_NonStartAction(t *testing.T) {
	trigger := &OnVMStarted{}

	t.Run("deallocate action is ignored", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "vm-deallocate-1",
				Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
				Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm/deallocate",
				EventType: EventTypeResourceActionSuccess,
				EventTime: time.Now(),
				Data: map[string]any{
					"subscriptionId": "test-sub",
					"operationName":  "Microsoft.Compute/virtualMachines/deallocate/action",
					"status":         "Succeeded",
				},
				DataVersion:     "1.0",
				MetadataVersion: "1",
			},
		}

		body, err := json.Marshal(events)
		require.NoError(t, err)

		eventsCtx := &contexts.EventContext{}
		webhookCtx := &contexts.NodeWebhookContext{}
		logger := logrus.NewEntry(logrus.New())

		ctx := core.WebhookRequestContext{
			Body:          body,
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Webhook:       webhookCtx,
			Events:        eventsCtx,
			Logger:        logger,
		}

		code, _, err := trigger.HandleWebhook(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)

		assert.Equal(t, 0, eventsCtx.Count())
	})

	t.Run("restart action is ignored", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "vm-restart-1",
				Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
				Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm/restart",
				EventType: EventTypeResourceActionSuccess,
				EventTime: time.Now(),
				Data: map[string]any{
					"subscriptionId": "test-sub",
					"operationName":  "Microsoft.Compute/virtualMachines/restart/action",
					"status":         "Succeeded",
				},
				DataVersion:     "1.0",
				MetadataVersion: "1",
			},
		}

		body, err := json.Marshal(events)
		require.NoError(t, err)

		eventsCtx := &contexts.EventContext{}
		webhookCtx := &contexts.NodeWebhookContext{}
		logger := logrus.NewEntry(logrus.New())

		ctx := core.WebhookRequestContext{
			Body:          body,
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Webhook:       webhookCtx,
			Events:        eventsCtx,
			Logger:        logger,
		}

		code, _, err := trigger.HandleWebhook(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)

		assert.Equal(t, 0, eventsCtx.Count())
	})
}

// TestOnVMStarted_HandleWebhook_FailedStatus verifies failed VM start handling
func TestOnVMStarted_HandleWebhook_FailedStatus(t *testing.T) {
	trigger := &OnVMStarted{}

	events := []EventGridEvent{
		{
			ID:        "vm-event-failed",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/failed-vm/start",
			EventType: EventTypeResourceActionSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"subscriptionId": "test-sub",
				"operationName":  "Microsoft.Compute/virtualMachines/start/action",
				"status":         "Failed",
			},
			DataVersion:     "1.0",
			MetadataVersion: "1",
		},
	}

	body, err := json.Marshal(events)
	require.NoError(t, err)

	eventsCtx := &contexts.EventContext{}
	webhookCtx := &contexts.NodeWebhookContext{}
	logger := logrus.NewEntry(logrus.New())

	ctx := core.WebhookRequestContext{
		Body:          body,
		Headers:       http.Header{},
		Configuration: map[string]any{},
		Webhook:       webhookCtx,
		Events:        eventsCtx,
		Logger:        logger,
	}

	code, _, err := trigger.HandleWebhook(ctx)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	assert.Equal(t, 0, eventsCtx.Count())
}

// TestOnVMStarted_HandleWebhook_MultipleEvents verifies handling multiple events in one batch
func TestOnVMStarted_HandleWebhook_MultipleEvents(t *testing.T) {
	trigger := &OnVMStarted{}

	events := []EventGridEvent{
		{
			ID:        "vm-event-1",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm-1/start",
			EventType: EventTypeResourceActionSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"status":         ProvisioningStateSucceeded,
				"subscriptionId": "test-sub",
				"operationName":  "Microsoft.Compute/virtualMachines/start/action",
			},
			DataVersion:     "1.0",
			MetadataVersion: "1",
		},
		{
			ID:        "vm-event-2",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm-2/start",
			EventType: EventTypeResourceActionSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"status":         ProvisioningStateSucceeded,
				"subscriptionId": "test-sub",
				"operationName":  "Microsoft.Compute/virtualMachines/start/action",
			},
			DataVersion:     "1.0",
			MetadataVersion: "1",
		},
		{
			ID:        "storage-event",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorage/listKeys",
			EventType: EventTypeResourceActionSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"status":         ProvisioningStateSucceeded,
				"subscriptionId": "test-sub",
				"operationName":  "Microsoft.Storage/storageAccounts/listKeys/action",
			},
			DataVersion:     "1.0",
			MetadataVersion: "1",
		},
	}

	body, err := json.Marshal(events)
	require.NoError(t, err)

	eventsCtx := &contexts.EventContext{}
	webhookCtx := &contexts.NodeWebhookContext{}
	logger := logrus.NewEntry(logrus.New())

	ctx := core.WebhookRequestContext{
		Body:          body,
		Headers:       http.Header{},
		Configuration: map[string]any{},
		Webhook:       webhookCtx,
		Events:        eventsCtx,
		Logger:        logger,
	}

	code, _, err := trigger.HandleWebhook(ctx)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	assert.Equal(t, 2, eventsCtx.Count())
	assert.Equal(t, "azure.vm.started", eventsCtx.Payloads[0].Type)
	assert.Equal(t, "azure.vm.started", eventsCtx.Payloads[1].Type)
}

// TestOnVMStarted_HandleWebhook_InvalidJSON verifies error handling for invalid JSON
func TestOnVMStarted_HandleWebhook_InvalidJSON(t *testing.T) {
	trigger := &OnVMStarted{}

	eventsCtx := &contexts.EventContext{}
	webhookCtx := &contexts.NodeWebhookContext{}
	logger := logrus.NewEntry(logrus.New())

	ctx := core.WebhookRequestContext{
		Body:          []byte("invalid json"),
		Headers:       http.Header{},
		Configuration: map[string]any{},
		Webhook:       webhookCtx,
		Events:        eventsCtx,
		Logger:        logger,
	}

	code, _, err := trigger.HandleWebhook(ctx)
	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, 0, eventsCtx.Count())
}

// TestOnVMStarted_HandleWebhook_InvalidConfiguration verifies error handling for invalid configuration
func TestOnVMStarted_HandleWebhook_InvalidConfiguration(t *testing.T) {
	trigger := &OnVMStarted{}

	events := []EventGridEvent{
		{
			ID:        "vm-event-1",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm/start",
			EventType: EventTypeResourceActionSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"status":        ProvisioningStateSucceeded,
				"operationName": "Microsoft.Compute/virtualMachines/start/action",
			},
			DataVersion:     "1.0",
			MetadataVersion: "1",
		},
	}

	body, err := json.Marshal(events)
	require.NoError(t, err)

	eventsCtx := &contexts.EventContext{}
	webhookCtx := &contexts.NodeWebhookContext{}
	logger := logrus.NewEntry(logrus.New())

	ctx := core.WebhookRequestContext{
		Body:    body,
		Headers: http.Header{},
		Configuration: map[string]any{
			"resourceGroup": 123, // Invalid type - should be string
		},
		Webhook: webhookCtx,
		Events:  eventsCtx,
		Logger:  logger,
	}

	code, _, err := trigger.HandleWebhook(ctx)
	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, code)
}

// TestOnVMStarted_HandleWebhook_IgnoresDeleteEvents verifies that delete events are ignored
func TestOnVMStarted_HandleWebhook_IgnoresDeleteEvents(t *testing.T) {
	trigger := &OnVMStarted{}

	events := []EventGridEvent{
		{
			ID:        "vm-delete-event",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm",
			EventType: EventTypeResourceDeleteSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"subscriptionId": "test-sub",
				"operationName":  "Microsoft.Compute/virtualMachines/delete",
				"status":         "Succeeded",
			},
			DataVersion:     "1.0",
			MetadataVersion: "1",
		},
	}

	body, err := json.Marshal(events)
	require.NoError(t, err)

	eventsCtx := &contexts.EventContext{}
	webhookCtx := &contexts.NodeWebhookContext{}
	logger := logrus.NewEntry(logrus.New())

	ctx := core.WebhookRequestContext{
		Body:          body,
		Headers:       http.Header{},
		Configuration: map[string]any{},
		Webhook:       webhookCtx,
		Events:        eventsCtx,
		Logger:        logger,
	}

	code, _, err := trigger.HandleWebhook(ctx)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	// Delete events should be ignored by the start trigger
	assert.Equal(t, 0, eventsCtx.Count())
}

// TestOnVMStarted_HandleWebhook_IgnoresWriteEvents verifies that write events are ignored
func TestOnVMStarted_HandleWebhook_IgnoresWriteEvents(t *testing.T) {
	trigger := &OnVMStarted{}

	events := []EventGridEvent{
		{
			ID:        "vm-write-event",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm",
			EventType: EventTypeResourceWriteSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"subscriptionId": "test-sub",
				"operationName":  "Microsoft.Compute/virtualMachines/write",
				"status":         "Succeeded",
			},
			DataVersion:     "1.0",
			MetadataVersion: "1",
		},
	}

	body, err := json.Marshal(events)
	require.NoError(t, err)

	eventsCtx := &contexts.EventContext{}
	webhookCtx := &contexts.NodeWebhookContext{}
	logger := logrus.NewEntry(logrus.New())

	ctx := core.WebhookRequestContext{
		Body:          body,
		Headers:       http.Header{},
		Configuration: map[string]any{},
		Webhook:       webhookCtx,
		Events:        eventsCtx,
		Logger:        logger,
	}

	code, _, err := trigger.HandleWebhook(ctx)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	// Write events should be ignored by the start trigger
	assert.Equal(t, 0, eventsCtx.Count())
}

// TestExtractVMNameFromActionSubject verifies VM name extraction from action subjects
func TestExtractVMNameFromActionSubject(t *testing.T) {
	tests := []struct {
		name     string
		subject  string
		expected string
	}{
		{
			name:     "start action subject",
			subject:  "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/my-vm/start",
			expected: "my-vm",
		},
		{
			name:     "deallocate action subject",
			subject:  "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/my-vm/deallocate",
			expected: "my-vm",
		},
		{
			name:     "no virtualMachines segment",
			subject:  "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/test",
			expected: "",
		},
		{
			name:     "empty subject",
			subject:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVMNameFromActionSubject(tt.subject)
			assert.Equal(t, tt.expected, result)
		})
	}
}
