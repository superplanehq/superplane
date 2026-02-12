package azure

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
)

// HandleWebhook processes Event Grid webhook requests.
func HandleWebhook(w http.ResponseWriter, r *http.Request, logger *logrus.Entry) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Errorf("Failed to read request body: %v", err)
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return fmt.Errorf("failed to read request body: %w", err)
	}
	defer r.Body.Close()

	var events []EventGridEvent
	if err := json.Unmarshal(body, &events); err != nil {
		logger.Errorf("Failed to parse Event Grid events: %v", err)
		http.Error(w, "invalid event format", http.StatusBadRequest)
		return fmt.Errorf("failed to parse events: %w", err)
	}

	logger.Infof("Received %d Event Grid event(s)", len(events))

	for _, event := range events {
		logger.Infof("Processing event: type=%s, subject=%s, id=%s", event.EventType, event.Subject, event.ID)

		switch event.EventType {
		case EventTypeSubscriptionValidation:
			return handleSubscriptionValidation(w, event, logger)

		case EventTypeResourceWriteSuccess:
			if err := handleResourceWriteSuccess(event, logger); err != nil {
				logger.Errorf("Failed to handle resource write success: %v", err)
			}

		default:
			logger.Infof("Ignoring event type: %s", event.EventType)
		}
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

// handleSubscriptionValidation responds to Event Grid validation events.
func handleSubscriptionValidation(w http.ResponseWriter, event EventGridEvent, logger *logrus.Entry) error {
	logger.Info("Handling Event Grid subscription validation")

	var validationData SubscriptionValidationEventData
	if err := mapstructure.Decode(event.Data, &validationData); err != nil {
		logger.Errorf("Failed to decode validation data: %v", err)
		http.Error(w, "invalid validation data", http.StatusBadRequest)
		return fmt.Errorf("failed to decode validation data: %w", err)
	}

	if validationData.ValidationCode == "" {
		logger.Error("Validation code is empty")
		http.Error(w, "validation code is empty", http.StatusBadRequest)
		return fmt.Errorf("validation code is empty")
	}

	logger.Infof("Validation code received: %s", validationData.ValidationCode)

	response := SubscriptionValidationResponse{
		ValidationResponse: validationData.ValidationCode,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Errorf("Failed to encode validation response: %v", err)
		return fmt.Errorf("failed to encode validation response: %w", err)
	}

	logger.Info("Subscription validation response sent successfully")
	return nil
}

// handleResourceWriteSuccess processes VM resource write success events.
func handleResourceWriteSuccess(event EventGridEvent, logger *logrus.Entry) error {
	if !strings.Contains(event.Subject, ResourceTypeVirtualMachine) {
		logger.Infof("Skipping non-VM resource: %s", event.Subject)
		return nil
	}

	var resourceData ResourceWriteSuccessData
	if err := mapstructure.Decode(event.Data, &resourceData); err != nil {
		return fmt.Errorf("failed to decode resource data: %w", err)
	}

	logger.Infof("Resource write success: provisioning_state=%s, resource=%s",
		resourceData.ProvisioningState, event.Subject)

	if resourceData.ProvisioningState != ProvisioningStateSucceeded {
		logger.Infof("VM not in Succeeded state, current state: %s", resourceData.ProvisioningState)
		return nil
	}

	vmID := event.Subject
	vmName := extractVMName(vmID)

	logger.Infof("VM created successfully: name=%s, id=%s, subscription=%s",
		vmName, vmID, resourceData.SubscriptionID)

	if resourceData.Authorization != nil {
		logger.Infof("Operation authorized: action=%s, scope=%s",
			resourceData.Authorization.Action, resourceData.Authorization.Scope)
	}

	return nil
}

// extractVMName returns VM name from ARM resource ID.
func extractVMName(resourceID string) string {
	parts := strings.Split(resourceID, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// extractResourceGroup returns resource group from ARM resource ID.
func extractResourceGroup(resourceID string) string {
	parts := strings.Split(resourceID, "/")
	for i, part := range parts {
		if part == "resourceGroups" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// extractSubscriptionID returns subscription ID from ARM resource ID.
func extractSubscriptionID(resourceID string) string {
	parts := strings.Split(resourceID, "/")
	for i, part := range parts {
		if part == "subscriptions" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// isVirtualMachineEvent reports whether an event subject targets a VM.
func isVirtualMachineEvent(subject string) bool {
	return strings.Contains(subject, ResourceTypeVirtualMachine)
}

// isSuccessfulProvisioning reports whether provisioning succeeded.
func isSuccessfulProvisioning(provisioningState string) bool {
	return provisioningState == ProvisioningStateSucceeded
}
