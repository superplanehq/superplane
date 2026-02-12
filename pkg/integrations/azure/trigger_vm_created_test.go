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

// TestOnVMCreatedTrigger_Metadata verifies the trigger's metadata methods
func TestOnVMCreatedTrigger_Metadata(t *testing.T) {
	trigger := &OnVMCreatedTrigger{}

	assert.Equal(t, "azure.onVirtualMachineCreated", trigger.Name())
	assert.Equal(t, "Azure • On VM Created", trigger.Label())
	assert.Equal(t, "azure", trigger.Icon())
	assert.Equal(t, "blue", trigger.Color())
	assert.NotEmpty(t, trigger.Description())
	assert.NotEmpty(t, trigger.Documentation())
}

// TestOnVMCreatedTrigger_Configuration verifies the trigger's configuration fields
func TestOnVMCreatedTrigger_Configuration(t *testing.T) {
	trigger := &OnVMCreatedTrigger{}
	config := trigger.Configuration()

	require.Len(t, config, 1)
	assert.Equal(t, "resourceGroup", config[0].Name)
	assert.Equal(t, "Resource Group", config[0].Label)
	assert.False(t, config[0].Required)
}

// TestOnVMCreatedTrigger_ExampleData verifies the trigger's example output
func TestOnVMCreatedTrigger_ExampleData(t *testing.T) {
	trigger := &OnVMCreatedTrigger{}
	example := trigger.ExampleData()

	require.NotNil(t, example)
	assert.Contains(t, example, "vmName")
	assert.Contains(t, example, "vmId")
	assert.Contains(t, example, "resourceGroup")
	assert.Contains(t, example, "subscriptionId")
	assert.Contains(t, example, "provisioningState")
}

// TestOnVMCreatedTrigger_Setup verifies the trigger setup method
func TestOnVMCreatedTrigger_Setup(t *testing.T) {
	trigger := &OnVMCreatedTrigger{}

	t.Run("setup with no resource group filter", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		logger := logrus.NewEntry(logrus.New())

		ctx := core.TriggerContext{
			Logger:        logger,
			Configuration: map[string]any{},
			Metadata:      metadataCtx,
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
			Metadata: metadataCtx,
		}

		err := trigger.Setup(ctx)
		assert.NoError(t, err)
	})
}

// TestOnVMCreatedTrigger_Cleanup verifies the trigger cleanup method
func TestOnVMCreatedTrigger_Cleanup(t *testing.T) {
	trigger := &OnVMCreatedTrigger{}
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

// TestOnVMCreatedTrigger_Actions verifies the trigger has no actions
func TestOnVMCreatedTrigger_Actions(t *testing.T) {
	trigger := &OnVMCreatedTrigger{}
	actions := trigger.Actions()
	assert.Empty(t, actions)
}

// TestOnVMCreatedTrigger_HandleAction verifies the trigger's action handler
func TestOnVMCreatedTrigger_HandleAction(t *testing.T) {
	trigger := &OnVMCreatedTrigger{}
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

// TestOnVMCreatedTrigger_HandleWebhook_SubscriptionValidation verifies subscription validation handling
func TestOnVMCreatedTrigger_HandleWebhook_SubscriptionValidation(t *testing.T) {
	trigger := &OnVMCreatedTrigger{}

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
			},
		},
	}

	body, err := json.Marshal(events)
	require.NoError(t, err)

	eventsCtx := &contexts.EventContext{}
	webhookCtx := &contexts.WebhookContext{}
	logger := logrus.NewEntry(logrus.New())

	ctx := core.WebhookRequestContext{
		Body:          body,
		Headers:       http.Header{},
		Configuration: map[string]any{},
		Webhook:       webhookCtx,
		Events:        eventsCtx,
		Logger:        logger,
	}

	code, err := trigger.HandleWebhook(ctx)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	// Subscription validation should not emit any events to the workflow
	assert.Equal(t, 0, eventsCtx.Count())
}

// TestOnVMCreatedTrigger_HandleWebhook_VMCreatedSuccess verifies VM creation event handling
func TestOnVMCreatedTrigger_HandleWebhook_VMCreatedSuccess(t *testing.T) {
	trigger := &OnVMCreatedTrigger{}

	t.Run("VM created with no filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "vm-event-1",
				Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
				Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm",
				EventType: EventTypeResourceWriteSuccess,
				EventTime: time.Now(),
				Data: map[string]any{
					"provisioningState": ProvisioningStateSucceeded,
					"subscriptionId":    "test-sub",
					"operationName":     "Microsoft.Compute/virtualMachines/write",
					"status":            "Succeeded",
				},
				DataVersion:     "1.0",
				MetadataVersion: "1",
			},
		}

		body, err := json.Marshal(events)
		require.NoError(t, err)

		eventsCtx := &contexts.EventContext{}
		webhookCtx := &contexts.WebhookContext{}
		logger := logrus.NewEntry(logrus.New())

		ctx := core.WebhookRequestContext{
			Body:          body,
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Webhook:       webhookCtx,
			Events:        eventsCtx,
			Logger:        logger,
		}

		code, err := trigger.HandleWebhook(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)

		// Should emit one event
		require.Equal(t, 1, eventsCtx.Count())
		assert.Equal(t, "azure.vm.created", eventsCtx.Payloads[0].Type)

		// Verify payload
		payload, ok := eventsCtx.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "test-vm", payload["vmName"])
		assert.Equal(t, "test-rg", payload["resourceGroup"])
		assert.Equal(t, "test-sub", payload["subscriptionId"])
		assert.Equal(t, ProvisioningStateSucceeded, payload["provisioningState"])
	})

	t.Run("VM created with matching resource group filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "vm-event-2",
				Topic:     "/subscriptions/test-sub/resourceGroups/my-target-rg",
				Subject:   "/subscriptions/test-sub/resourceGroups/my-target-rg/providers/Microsoft.Compute/virtualMachines/test-vm-2",
				EventType: EventTypeResourceWriteSuccess,
				EventTime: time.Now(),
				Data: map[string]any{
					"provisioningState": ProvisioningStateSucceeded,
					"subscriptionId":    "test-sub",
					"operationName":     "Microsoft.Compute/virtualMachines/write",
					"status":            "Succeeded",
				},
				DataVersion:     "1.0",
				MetadataVersion: "1",
			},
		}

		body, err := json.Marshal(events)
		require.NoError(t, err)

		eventsCtx := &contexts.EventContext{}
		webhookCtx := &contexts.WebhookContext{}
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

		code, err := trigger.HandleWebhook(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)

		// Should emit one event
		require.Equal(t, 1, eventsCtx.Count())
		assert.Equal(t, "azure.vm.created", eventsCtx.Payloads[0].Type)

		// Verify resource group in payload
		payload, ok := eventsCtx.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "my-target-rg", payload["resourceGroup"])
	})
}

// TestOnVMCreatedTrigger_HandleWebhook_FilterMismatch verifies resource group filtering
func TestOnVMCreatedTrigger_HandleWebhook_FilterMismatch(t *testing.T) {
	trigger := &OnVMCreatedTrigger{}

	events := []EventGridEvent{
		{
			ID:        "vm-event-3",
			Topic:     "/subscriptions/test-sub/resourceGroups/rg-other",
			Subject:   "/subscriptions/test-sub/resourceGroups/rg-other/providers/Microsoft.Compute/virtualMachines/test-vm-other",
			EventType: EventTypeResourceWriteSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"provisioningState": ProvisioningStateSucceeded,
				"subscriptionId":    "test-sub",
				"operationName":     "Microsoft.Compute/virtualMachines/write",
				"status":            "Succeeded",
			},
			DataVersion:     "1.0",
			MetadataVersion: "1",
		},
	}

	body, err := json.Marshal(events)
	require.NoError(t, err)

	eventsCtx := &contexts.EventContext{}
	webhookCtx := &contexts.WebhookContext{}
	logger := logrus.NewEntry(logrus.New())

	ctx := core.WebhookRequestContext{
		Body:    body,
		Headers: http.Header{},
		Configuration: map[string]any{
			"resourceGroup": "rg-target", // Different from rg-other
		},
		Webhook: webhookCtx,
		Events:  eventsCtx,
		Logger:  logger,
	}

	code, err := trigger.HandleWebhook(ctx)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	// Should NOT emit any events due to filter mismatch
	assert.Equal(t, 0, eventsCtx.Count())
}

// TestOnVMCreatedTrigger_HandleWebhook_NonVMResource verifies non-VM resource filtering
func TestOnVMCreatedTrigger_HandleWebhook_NonVMResource(t *testing.T) {
	trigger := &OnVMCreatedTrigger{}

	t.Run("storage account creation", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "storage-event-1",
				Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
				Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorage",
				EventType: EventTypeResourceWriteSuccess,
				EventTime: time.Now(),
				Data: map[string]any{
					"provisioningState": ProvisioningStateSucceeded,
					"subscriptionId":    "test-sub",
					"operationName":     "Microsoft.Storage/storageAccounts/write",
					"status":            "Succeeded",
				},
				DataVersion:     "1.0",
				MetadataVersion: "1",
			},
		}

		body, err := json.Marshal(events)
		require.NoError(t, err)

		eventsCtx := &contexts.EventContext{}
		webhookCtx := &contexts.WebhookContext{}
		logger := logrus.NewEntry(logrus.New())

		ctx := core.WebhookRequestContext{
			Body:          body,
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Webhook:       webhookCtx,
			Events:        eventsCtx,
			Logger:        logger,
		}

		code, err := trigger.HandleWebhook(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)

		// Should NOT emit any events - not a VM
		assert.Equal(t, 0, eventsCtx.Count())
	})

	t.Run("network interface creation", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "nic-event-1",
				Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
				Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Network/networkInterfaces/test-nic",
				EventType: EventTypeResourceWriteSuccess,
				EventTime: time.Now(),
				Data: map[string]any{
					"provisioningState": ProvisioningStateSucceeded,
					"subscriptionId":    "test-sub",
					"operationName":     "Microsoft.Network/networkInterfaces/write",
					"status":            "Succeeded",
				},
				DataVersion:     "1.0",
				MetadataVersion: "1",
			},
		}

		body, err := json.Marshal(events)
		require.NoError(t, err)

		eventsCtx := &contexts.EventContext{}
		webhookCtx := &contexts.WebhookContext{}
		logger := logrus.NewEntry(logrus.New())

		ctx := core.WebhookRequestContext{
			Body:          body,
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Webhook:       webhookCtx,
			Events:        eventsCtx,
			Logger:        logger,
		}

		code, err := trigger.HandleWebhook(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)

		// Should NOT emit any events - not a VM
		assert.Equal(t, 0, eventsCtx.Count())
	})
}

// TestOnVMCreatedTrigger_HandleWebhook_ProvisioningStateFailed verifies failed VM creation handling
func TestOnVMCreatedTrigger_HandleWebhook_ProvisioningStateFailed(t *testing.T) {
	trigger := &OnVMCreatedTrigger{}

	events := []EventGridEvent{
		{
			ID:        "vm-event-failed",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/failed-vm",
			EventType: EventTypeResourceWriteSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"provisioningState": ProvisioningStateFailed,
				"subscriptionId":    "test-sub",
				"operationName":     "Microsoft.Compute/virtualMachines/write",
				"status":            "Failed",
			},
			DataVersion:     "1.0",
			MetadataVersion: "1",
		},
	}

	body, err := json.Marshal(events)
	require.NoError(t, err)

	eventsCtx := &contexts.EventContext{}
	webhookCtx := &contexts.WebhookContext{}
	logger := logrus.NewEntry(logrus.New())

	ctx := core.WebhookRequestContext{
		Body:          body,
		Headers:       http.Header{},
		Configuration: map[string]any{},
		Webhook:       webhookCtx,
		Events:        eventsCtx,
		Logger:        logger,
	}

	code, err := trigger.HandleWebhook(ctx)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	// Should NOT emit any events - provisioning failed
	assert.Equal(t, 0, eventsCtx.Count())
}

// TestOnVMCreatedTrigger_HandleWebhook_MultipleEvents verifies handling multiple events in one batch
func TestOnVMCreatedTrigger_HandleWebhook_MultipleEvents(t *testing.T) {
	trigger := &OnVMCreatedTrigger{}

	events := []EventGridEvent{
		{
			ID:        "vm-event-1",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm-1",
			EventType: EventTypeResourceWriteSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"provisioningState": ProvisioningStateSucceeded,
				"subscriptionId":    "test-sub",
			},
			DataVersion:     "1.0",
			MetadataVersion: "1",
		},
		{
			ID:        "vm-event-2",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm-2",
			EventType: EventTypeResourceWriteSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"provisioningState": ProvisioningStateSucceeded,
				"subscriptionId":    "test-sub",
			},
			DataVersion:     "1.0",
			MetadataVersion: "1",
		},
		{
			ID:        "storage-event",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorage",
			EventType: EventTypeResourceWriteSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"provisioningState": ProvisioningStateSucceeded,
				"subscriptionId":    "test-sub",
			},
			DataVersion:     "1.0",
			MetadataVersion: "1",
		},
	}

	body, err := json.Marshal(events)
	require.NoError(t, err)

	eventsCtx := &contexts.EventContext{}
	webhookCtx := &contexts.WebhookContext{}
	logger := logrus.NewEntry(logrus.New())

	ctx := core.WebhookRequestContext{
		Body:          body,
		Headers:       http.Header{},
		Configuration: map[string]any{},
		Webhook:       webhookCtx,
		Events:        eventsCtx,
		Logger:        logger,
	}

	code, err := trigger.HandleWebhook(ctx)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	// Should emit two events (only the VM events, not the storage account)
	assert.Equal(t, 2, eventsCtx.Count())
	assert.Equal(t, "azure.vm.created", eventsCtx.Payloads[0].Type)
	assert.Equal(t, "azure.vm.created", eventsCtx.Payloads[1].Type)
}

// TestOnVMCreatedTrigger_HandleWebhook_InvalidJSON verifies error handling for invalid JSON
func TestOnVMCreatedTrigger_HandleWebhook_InvalidJSON(t *testing.T) {
	trigger := &OnVMCreatedTrigger{}

	eventsCtx := &contexts.EventContext{}
	webhookCtx := &contexts.WebhookContext{}
	logger := logrus.NewEntry(logrus.New())

	ctx := core.WebhookRequestContext{
		Body:          []byte("invalid json"),
		Headers:       http.Header{},
		Configuration: map[string]any{},
		Webhook:       webhookCtx,
		Events:        eventsCtx,
		Logger:        logger,
	}

	code, err := trigger.HandleWebhook(ctx)
	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, 0, eventsCtx.Count())
}

// TestOnVMCreatedTrigger_HandleWebhook_InvalidConfiguration verifies error handling for invalid configuration
func TestOnVMCreatedTrigger_HandleWebhook_InvalidConfiguration(t *testing.T) {
	trigger := &OnVMCreatedTrigger{}

	events := []EventGridEvent{
		{
			ID:        "vm-event-1",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm",
			EventType: EventTypeResourceWriteSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"provisioningState": ProvisioningStateSucceeded,
			},
			DataVersion:     "1.0",
			MetadataVersion: "1",
		},
	}

	body, err := json.Marshal(events)
	require.NoError(t, err)

	eventsCtx := &contexts.EventContext{}
	webhookCtx := &contexts.WebhookContext{}
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

	code, err := trigger.HandleWebhook(ctx)
	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, code)
}

