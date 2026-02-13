package azure

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleWebhook_SubscriptionValidation(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	// Create a subscription validation event
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

	// Create request
	body, err := json.Marshal(events)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Handle webhook
	err = HandleWebhook(rec, req, logger)
	assert.NoError(t, err)

	// Verify response
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	// Parse response
	var response SubscriptionValidationResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, validationCode, response.ValidationResponse)
}

func TestHandleWebhook_ResourceWriteSuccess_VM(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	// Create a VM creation success event
	events := []EventGridEvent{
		{
			ID:        "vm-event-1",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm",
			EventType: EventTypeResourceWriteSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"provisioningState": "Succeeded",
				"resourceProvider":  "Microsoft.Compute",
				"resourceUri":       "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm",
				"operationName":     "Microsoft.Compute/virtualMachines/write",
				"status":            "Succeeded",
				"subscriptionId":    "test-sub",
				"tenantId":          "test-tenant",
			},
			DataVersion:     "1.0",
			MetadataVersion: "1",
		},
	}

	// Create request
	body, err := json.Marshal(events)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Handle webhook
	err = HandleWebhook(rec, req, logger)
	assert.NoError(t, err)

	// Verify response
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleWebhook_ResourceWriteSuccess_NonVM(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	// Create a storage account event (should be ignored)
	events := []EventGridEvent{
		{
			ID:        "storage-event-1",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststore",
			EventType: EventTypeResourceWriteSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"provisioningState": "Succeeded",
				"resourceProvider":  "Microsoft.Storage",
				"resourceUri":       "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststore",
			},
			DataVersion:     "1.0",
			MetadataVersion: "1",
		},
	}

	// Create request
	body, err := json.Marshal(events)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Handle webhook
	err = HandleWebhook(rec, req, logger)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleWebhook_MultipleEvents(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	// Create multiple events
	events := []EventGridEvent{
		{
			ID:        "vm-event-1",
			Topic:     "/subscriptions/test-sub/resourceGroups/test-rg",
			Subject:   "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm-1",
			EventType: EventTypeResourceWriteSuccess,
			EventTime: time.Now(),
			Data: map[string]any{
				"provisioningState": "Succeeded",
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
				"provisioningState": "Succeeded",
				"subscriptionId":    "test-sub",
			},
			DataVersion:     "1.0",
			MetadataVersion: "1",
		},
	}

	// Create request
	body, err := json.Marshal(events)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Handle webhook
	err = HandleWebhook(rec, req, logger)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleWebhook_InvalidJSON(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	// Create invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Handle webhook
	err := HandleWebhook(rec, req, logger)
	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleWebhook_EmptyValidationCode(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	// Create validation event with empty code
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
				"validationCode": "",
			},
		},
	}

	body, err := json.Marshal(events)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	err = HandleWebhook(rec, req, logger)
	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

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

func TestIsSuccessfulProvisioning(t *testing.T) {
	tests := []struct {
		name              string
		provisioningState string
		expected          bool
	}{
		{
			name:              "succeeded",
			provisioningState: ProvisioningStateSucceeded,
			expected:          true,
		},
		{
			name:              "failed",
			provisioningState: ProvisioningStateFailed,
			expected:          false,
		},
		{
			name:              "creating",
			provisioningState: ProvisioningStateCreating,
			expected:          false,
		},
		{
			name:              "empty",
			provisioningState: "",
			expected:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSuccessfulProvisioning(tt.provisioningState)
			assert.Equal(t, tt.expected, result)
		})
	}
}
