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

// TestOnVMWriteTrigger_Metadata verifies the trigger's metadata methods
func TestOnVMWriteTrigger_Metadata(t *testing.T) {
	trigger := &OnVMWriteTrigger{}

	assert.Equal(t, "azure.onVirtualMachineWrite", trigger.Name())
	assert.Equal(t, "Azure • On VM Write", trigger.Label())
	assert.Equal(t, "azure", trigger.Icon())
	assert.Equal(t, "blue", trigger.Color())
	assert.NotEmpty(t, trigger.Description())
	assert.NotEmpty(t, trigger.Documentation())
}

// TestOnVMWriteTrigger_Configuration verifies the trigger's configuration fields
func TestOnVMWriteTrigger_Configuration(t *testing.T) {
	trigger := &OnVMWriteTrigger{}
	config := trigger.Configuration()

	require.Len(t, config, 2)
	assert.Equal(t, "resourceGroup", config[0].Name)
	assert.Equal(t, "Resource Group", config[0].Label)
	assert.False(t, config[0].Required)
	assert.Equal(t, "nameFilter", config[1].Name)
	assert.Equal(t, "VM Name Filter", config[1].Label)
	assert.False(t, config[1].Required)
}

// TestOnVMWriteTrigger_ExampleData verifies the trigger's example output
func TestOnVMWriteTrigger_ExampleData(t *testing.T) {
	trigger := &OnVMWriteTrigger{}
	example := trigger.ExampleData()

	require.NotNil(t, example)
	assert.Contains(t, example, "id")
	assert.Contains(t, example, "eventType")
	assert.Contains(t, example, "subject")
	assert.Contains(t, example, "data")
}

// TestOnVMWriteTrigger_Setup verifies the trigger setup method
func TestOnVMWriteTrigger_Setup(t *testing.T) {
	trigger := &OnVMWriteTrigger{}

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

// TestOnVMWriteTrigger_Cleanup verifies the trigger cleanup method
func TestOnVMWriteTrigger_Cleanup(t *testing.T) {
	trigger := &OnVMWriteTrigger{}
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

// TestOnVMWriteTrigger_Actions verifies the trigger has no actions
func TestOnVMWriteTrigger_Actions(t *testing.T) {
	trigger := &OnVMWriteTrigger{}
	actions := trigger.Actions()
	assert.Empty(t, actions)
}

// TestOnVMWriteTrigger_HandleAction verifies the trigger's action handler
func TestOnVMWriteTrigger_HandleAction(t *testing.T) {
	trigger := &OnVMWriteTrigger{}
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

// TestOnVMWriteTrigger_HandleWebhook_SubscriptionValidation verifies subscription validation handling
func TestOnVMWriteTrigger_HandleWebhook_SubscriptionValidation(t *testing.T) {
	trigger := &OnVMWriteTrigger{}

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

// TestOnVMWriteTrigger_HandleWebhook_VMWriteSuccess verifies VM write event handling
func TestOnVMWriteTrigger_HandleWebhook_VMWriteSuccess(t *testing.T) {
	trigger := &OnVMWriteTrigger{}

	t.Run("VM write with no filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "vm-event-1",
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

		require.Equal(t, 1, eventsCtx.Count())
		assert.Equal(t, "azure.vm.write", eventsCtx.Payloads[0].Type)

		// The emitted payload is the full, raw Event Grid event
		payload, ok := eventsCtx.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "vm-event-1", payload["id"])
		assert.Equal(t, EventTypeResourceWriteSuccess, payload["eventType"])
		assert.Equal(t, "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm", payload["subject"])

		data, ok := payload["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Succeeded", data["status"])
		assert.Equal(t, "Microsoft.Compute/virtualMachines/write", data["operationName"])
	})

	t.Run("VM write with matching resource group filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "vm-event-2",
				Topic:     "/subscriptions/test-sub/resourceGroups/my-target-rg",
				Subject:   "/subscriptions/test-sub/resourceGroups/my-target-rg/providers/Microsoft.Compute/virtualMachines/test-vm-2",
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

		require.Equal(t, 1, eventsCtx.Count())
		assert.Equal(t, "azure.vm.write", eventsCtx.Payloads[0].Type)

		payload, ok := eventsCtx.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Contains(t, payload["subject"], "my-target-rg")
	})
}

// TestOnVMWriteTrigger_HandleWebhook_FilterMismatch verifies resource group filtering
func TestOnVMWriteTrigger_HandleWebhook_FilterMismatch(t *testing.T) {
	trigger := &OnVMWriteTrigger{}

	events := []EventGridEvent{
		{
			ID:        "vm-event-3",
			Topic:     "/subscriptions/test-sub/resourceGroups/rg-other",
			Subject:   "/subscriptions/test-sub/resourceGroups/rg-other/providers/Microsoft.Compute/virtualMachines/test-vm-other",
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
	webhookCtx := &contexts.WebhookContext{}
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

	code, err := trigger.HandleWebhook(ctx)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	assert.Equal(t, 0, eventsCtx.Count())
}

// TestOnVMWriteTrigger_HandleWebhook_NameFilter verifies VM name regex filtering
func TestOnVMWriteTrigger_HandleWebhook_NameFilter(t *testing.T) {
	trigger := &OnVMWriteTrigger{}

	t.Run("matching name filter", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "vm-event-name-1",
				Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
				Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/prod-web-01",
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
		webhookCtx := &contexts.WebhookContext{}
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

		code, err := trigger.HandleWebhook(ctx)
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
				Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/dev-web-01",
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
		webhookCtx := &contexts.WebhookContext{}
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

		code, err := trigger.HandleWebhook(ctx)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)

		assert.Equal(t, 0, eventsCtx.Count())
	})

	t.Run("empty name filter triggers for all", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "vm-event-name-3",
				Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
				Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/any-vm",
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

		require.Equal(t, 1, eventsCtx.Count())
	})
}

// TestOnVMWriteTrigger_HandleWebhook_NonVMResource verifies non-VM resource filtering
func TestOnVMWriteTrigger_HandleWebhook_NonVMResource(t *testing.T) {
	trigger := &OnVMWriteTrigger{}

	t.Run("storage account creation", func(t *testing.T) {
		events := []EventGridEvent{
			{
				ID:        "storage-event-1",
				Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
				Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorage",
				EventType: EventTypeResourceWriteSuccess,
				EventTime: time.Now(),
				Data: map[string]any{
					"subscriptionId": "test-sub",
					"operationName":  "Microsoft.Storage/storageAccounts/write",
					"status":         "Succeeded",
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
					"subscriptionId": "test-sub",
					"operationName":  "Microsoft.Network/networkInterfaces/write",
					"status":         "Succeeded",
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

		assert.Equal(t, 0, eventsCtx.Count())
	})
}

// TestOnVMWriteTrigger_HandleWebhook_FailedStatus verifies failed VM operation handling
func TestOnVMWriteTrigger_HandleWebhook_FailedStatus(t *testing.T) {
	trigger := &OnVMWriteTrigger{}

	events := []EventGridEvent{
		{
			ID:        "vm-event-failed",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/failed-vm",
			EventType: EventTypeResourceWriteSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"subscriptionId": "test-sub",
				"operationName":  "Microsoft.Compute/virtualMachines/write",
				"status":         "Failed",
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

	assert.Equal(t, 0, eventsCtx.Count())
}

// TestOnVMWriteTrigger_HandleWebhook_MultipleEvents verifies handling multiple events in one batch
func TestOnVMWriteTrigger_HandleWebhook_MultipleEvents(t *testing.T) {
	trigger := &OnVMWriteTrigger{}

	events := []EventGridEvent{
		{
			ID:        "vm-event-1",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm-1",
			EventType: EventTypeResourceWriteSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"status":         ProvisioningStateSucceeded,
				"subscriptionId": "test-sub",
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
				"status":         ProvisioningStateSucceeded,
				"subscriptionId": "test-sub",
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
				"status":         ProvisioningStateSucceeded,
				"subscriptionId": "test-sub",
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

	assert.Equal(t, 2, eventsCtx.Count())
	assert.Equal(t, "azure.vm.write", eventsCtx.Payloads[0].Type)
	assert.Equal(t, "azure.vm.write", eventsCtx.Payloads[1].Type)
}

// TestOnVMWriteTrigger_HandleWebhook_InvalidJSON verifies error handling for invalid JSON
func TestOnVMWriteTrigger_HandleWebhook_InvalidJSON(t *testing.T) {
	trigger := &OnVMWriteTrigger{}

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

// TestOnVMWriteTrigger_HandleWebhook_InvalidConfiguration verifies error handling for invalid configuration
func TestOnVMWriteTrigger_HandleWebhook_InvalidConfiguration(t *testing.T) {
	trigger := &OnVMWriteTrigger{}

	events := []EventGridEvent{
		{
			ID:        "vm-event-1",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm",
			EventType: EventTypeResourceWriteSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"status": ProvisioningStateSucceeded,
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

// Tests for helper functions

func TestExtractVMName(t *testing.T) {
	tests := []struct {
		name       string
		resourceID string
		expected   string
	}{
		{
			name:       "valid VM resource ID",
			resourceID: "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/my-vm",
			expected:   "my-vm",
		},
		{
			name:       "empty resource ID",
			resourceID: "",
			expected:   "",
		},
		{
			name:       "single segment",
			resourceID: "vm-name",
			expected:   "vm-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVMName(tt.resourceID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractResourceGroup(t *testing.T) {
	tests := []struct {
		name       string
		resourceID string
		expected   string
	}{
		{
			name:       "valid resource ID",
			resourceID: "/subscriptions/test-sub/resourceGroups/my-rg/providers/Microsoft.Compute/virtualMachines/my-vm",
			expected:   "my-rg",
		},
		{
			name:       "lowercase resourcegroups from Azure Event Grid",
			resourceID: "/subscriptions/test-sub/resourcegroups/my-rg/providers/Microsoft.Compute/virtualMachines/my-vm",
			expected:   "my-rg",
		},
		{
			name:       "no resource group",
			resourceID: "/subscriptions/test-sub",
			expected:   "",
		},
		{
			name:       "empty resource ID",
			resourceID: "",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractResourceGroup(tt.resourceID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractSubscriptionID(t *testing.T) {
	tests := []struct {
		name       string
		resourceID string
		expected   string
	}{
		{
			name:       "valid resource ID",
			resourceID: "/subscriptions/my-subscription-id/resourceGroups/my-rg/providers/Microsoft.Compute/virtualMachines/my-vm",
			expected:   "my-subscription-id",
		},
		{
			name:       "no subscription",
			resourceID: "/resourceGroups/my-rg",
			expected:   "",
		},
		{
			name:       "empty resource ID",
			resourceID: "",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSubscriptionID(tt.resourceID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsVirtualMachineEvent(t *testing.T) {
	tests := []struct {
		name     string
		subject  string
		expected bool
	}{
		{
			name:     "VM event",
			subject:  "/subscriptions/test/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm1",
			expected: true,
		},
		{
			name:     "storage event",
			subject:  "/subscriptions/test/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/storage1",
			expected: false,
		},
		{
			name:     "empty subject",
			subject:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVirtualMachineEvent(tt.subject)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSuccessfulStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{
			name:     "succeeded",
			status:   ProvisioningStateSucceeded,
			expected: true,
		},
		{
			name:     "failed",
			status:   ProvisioningStateFailed,
			expected: false,
		},
		{
			name:     "creating",
			status:   ProvisioningStateCreating,
			expected: false,
		},
		{
			name:     "empty",
			status:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSuccessfulStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}
