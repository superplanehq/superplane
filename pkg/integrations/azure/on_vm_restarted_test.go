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

// TestOnVMRestarted_Metadata verifies the trigger's metadata methods
func TestOnVMRestarted_Metadata(t *testing.T) {
	trigger := &OnVMRestarted{}

	assert.Equal(t, "azure.onVirtualMachineRestarted", trigger.Name())
	assert.Equal(t, "On VM Restarted", trigger.Label())
	assert.Equal(t, "azure", trigger.Icon())
	assert.Equal(t, "blue", trigger.Color())
	assert.NotEmpty(t, trigger.Description())
	assert.NotEmpty(t, trigger.Documentation())
}

// TestOnVMRestarted_Configuration verifies the trigger's configuration fields
func TestOnVMRestarted_Configuration(t *testing.T) {
	trigger := &OnVMRestarted{}
	config := trigger.Configuration()

	require.Len(t, config, 2)
	assert.Equal(t, "resourceGroup", config[0].Name)
	assert.Equal(t, "Resource Group", config[0].Label)
	assert.False(t, config[0].Required)
	assert.Equal(t, "nameFilter", config[1].Name)
	assert.Equal(t, "VM Name Filter", config[1].Label)
	assert.False(t, config[1].Required)
}

// TestOnVMRestarted_ExampleData verifies the trigger's example output
func TestOnVMRestarted_ExampleData(t *testing.T) {
	trigger := &OnVMRestarted{}
	example := trigger.ExampleData()

	require.NotNil(t, example)
	assert.Contains(t, example, "id")
	assert.Contains(t, example, "eventType")
	assert.Equal(t, "Microsoft.Resources.ResourceActionSuccess", example["eventType"])
	assert.Contains(t, example, "subject")
	assert.Contains(t, example, "data")

	data, ok := example["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Microsoft.Compute/virtualMachines/restart/action", data["operationName"])
}

// TestOnVMRestarted_Setup verifies the trigger setup method
func TestOnVMRestarted_Setup(t *testing.T) {
	trigger := &OnVMRestarted{}

	t.Run("setup with no resource group filter", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		logger := logrus.NewEntry(logrus.New())

		ctx := core.TriggerContext{
			Logger:        logger,
			Configuration: map[string]any{},
			Metadata:      metadataCtx,
			Integration:   &mockIntegrationContext{},
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
			Integration: &mockIntegrationContext{},
		}

		err := trigger.Setup(ctx)
		assert.NoError(t, err)
	})
}

// TestOnVMRestarted_Cleanup verifies the trigger cleanup method
func TestOnVMRestarted_Cleanup(t *testing.T) {
	trigger := &OnVMRestarted{}
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

// TestOnVMRestarted_Actions verifies the trigger has no actions
func TestOnVMRestarted_Actions(t *testing.T) {
	trigger := &OnVMRestarted{}
	actions := trigger.Actions()
	assert.Empty(t, actions)
}

// TestOnVMRestarted_HandleAction verifies the trigger's action handler
func TestOnVMRestarted_HandleAction(t *testing.T) {
	trigger := &OnVMRestarted{}
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

// TestOnVMRestarted_HandleWebhook_SubscriptionValidation verifies subscription validation handling
func TestOnVMRestarted_HandleWebhook_SubscriptionValidation(t *testing.T) {
	trigger := &OnVMRestarted{}

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
	assert.Equal(t, 0, eventsCtx.Count())
}

// TestOnVMRestarted_HandleWebhook_VMRestartSuccess verifies VM restart event handling
func TestOnVMRestarted_HandleWebhook_VMRestartSuccess(t *testing.T) {
	trigger := &OnVMRestarted{}

	t.Run("VM restarted with no filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "vm-event-1",
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

		require.Equal(t, 1, eventsCtx.Count())
		assert.Equal(t, "azure.vm.restarted", eventsCtx.Payloads[0].Type)

		payload, ok := eventsCtx.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "vm-event-1", payload["id"])
		assert.Equal(t, EventTypeResourceActionSuccess, payload["eventType"])

		data, ok := payload["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Succeeded", data["status"])
		assert.Equal(t, "Microsoft.Compute/virtualMachines/restart/action", data["operationName"])
	})

	t.Run("VM restarted with matching resource group filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "vm-event-2",
				Topic:     "/subscriptions/test-sub/resourceGroups/my-target-rg",
				Subject:   "/subscriptions/test-sub/resourceGroups/my-target-rg/providers/Microsoft.Compute/virtualMachines/test-vm-2/restart",
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
		assert.Equal(t, "azure.vm.restarted", eventsCtx.Payloads[0].Type)
	})
}

// TestOnVMRestarted_HandleWebhook_FilterMismatch verifies resource group filtering
func TestOnVMRestarted_HandleWebhook_FilterMismatch(t *testing.T) {
	trigger := &OnVMRestarted{}

	events := []EventGridEvent{
		{
			ID:        "vm-event-3",
			Topic:     "/subscriptions/test-sub/resourceGroups/rg-other",
			Subject:   "/subscriptions/test-sub/resourceGroups/rg-other/providers/Microsoft.Compute/virtualMachines/test-vm-other/restart",
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

// TestOnVMRestarted_HandleWebhook_NameFilter verifies VM name regex filtering
func TestOnVMRestarted_HandleWebhook_NameFilter(t *testing.T) {
	trigger := &OnVMRestarted{}

	t.Run("matching name filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "vm-event-name-1",
				Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
				Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/prod-web-01/restart",
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
	})

	t.Run("non-matching name filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "vm-event-name-2",
				Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
				Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/dev-web-01/restart",
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
}

// TestOnVMRestarted_HandleWebhook_NonRestartAction verifies that non-restart actions are ignored
func TestOnVMRestarted_HandleWebhook_NonRestartAction(t *testing.T) {
	trigger := &OnVMRestarted{}

	t.Run("start action is ignored", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "vm-start-1",
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
		assert.Equal(t, 0, eventsCtx.Count())
	})

	t.Run("deallocate action is ignored", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "vm-dealloc-1",
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

	t.Run("powerOff action is ignored", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "vm-poweroff-1",
				Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
				Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm/powerOff",
				EventType: EventTypeResourceActionSuccess,
				EventTime: time.Now(),
				Data: map[string]any{
					"subscriptionId": "test-sub",
					"operationName":  "Microsoft.Compute/virtualMachines/powerOff/action",
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

// TestOnVMRestarted_HandleWebhook_FailedStatus verifies failed VM restart handling
func TestOnVMRestarted_HandleWebhook_FailedStatus(t *testing.T) {
	trigger := &OnVMRestarted{}

	events := []EventGridEvent{
		{
			ID:        "vm-event-failed",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/failed-vm/restart",
			EventType: EventTypeResourceActionSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"subscriptionId": "test-sub",
				"operationName":  "Microsoft.Compute/virtualMachines/restart/action",
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

// TestOnVMRestarted_HandleWebhook_MultipleEvents verifies handling multiple events in one batch
func TestOnVMRestarted_HandleWebhook_MultipleEvents(t *testing.T) {
	trigger := &OnVMRestarted{}

	events := []EventGridEvent{
		{
			ID:        "vm-event-1",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm-1/restart",
			EventType: EventTypeResourceActionSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"status":         ProvisioningStateSucceeded,
				"subscriptionId": "test-sub",
				"operationName":  "Microsoft.Compute/virtualMachines/restart/action",
			},
			DataVersion:     "1.0",
			MetadataVersion: "1",
		},
		{
			ID:        "vm-event-2",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm-2/restart",
			EventType: EventTypeResourceActionSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"status":         ProvisioningStateSucceeded,
				"subscriptionId": "test-sub",
				"operationName":  "Microsoft.Compute/virtualMachines/restart/action",
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
	assert.Equal(t, "azure.vm.restarted", eventsCtx.Payloads[0].Type)
	assert.Equal(t, "azure.vm.restarted", eventsCtx.Payloads[1].Type)
}

// TestOnVMRestarted_HandleWebhook_InvalidJSON verifies error handling for invalid JSON
func TestOnVMRestarted_HandleWebhook_InvalidJSON(t *testing.T) {
	trigger := &OnVMRestarted{}

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

// TestOnVMRestarted_HandleWebhook_NonVMResource verifies non-VM resource filtering
func TestOnVMRestarted_HandleWebhook_NonVMResource(t *testing.T) {
	trigger := &OnVMRestarted{}

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
